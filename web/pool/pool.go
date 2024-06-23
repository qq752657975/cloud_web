package pool

import (
	"errors"
	"fmt"
	"github.com/ygb616/web/config"
	"sync"
	"sync/atomic"
	"time"
)

type sig struct{}

const DefaultExpire = 3

var (
	ErrorInValidCap    = errors.New("pool cap can not <= 0")
	ErrorInValidExpire = errors.New("pool expire can not <= 0")
	ErrorHasClosed     = errors.New("pool has bean released!!")
)

type Pool struct {
	//cap 容量 pool max cap
	cap int32
	//running 正在运行的worker的数量
	running int32
	//空闲worker
	workers []*Worker
	//expire 过期时间 空闲的worker超过这个时间 回收掉
	expire time.Duration
	//release 释放资源  pool就不能使用了
	release chan sig
	//lock 去保护pool里面的相关资源的安全
	lock sync.Mutex
	//once 释放只能调用一次 不能多次调用
	once sync.Once
	// 缓存
	workerCache sync.Pool
	//cond
	cond *sync.Cond
	//PanicHandler
	PanicHandler func()
}

// NewPoolConf 从配置文件中创建一个新的连接池
func NewPoolConf() (*Pool, error) {
	// 从全局配置 config.Conf 中获取连接池配置 "c"
	c, ok := config.Conf.Pool["c"]
	if !ok {
		// 如果配置中没有找到 "c"，返回错误
		return nil, errors.New("c config not exist")
	}
	// 调用 NewTimePool 函数创建一个新的连接池，使用从配置中获取的值作为参数
	return NewTimePool(int(c.(int64)), DefaultExpire)
}

func NewPool(cap int) (*Pool, error) {
	return NewTimePool(cap, DefaultExpire)
}

func NewTimePool(cap int, expire int) (*Pool, error) {
	if cap <= 0 {
		return nil, ErrorInValidCap
	}
	if expire <= 0 {
		return nil, ErrorInValidExpire
	}
	p := &Pool{
		cap:     int32(cap),
		expire:  time.Duration(expire) * time.Second,
		release: make(chan sig, 1),
	}
	p.workerCache.New = func() any {
		return &Worker{
			pool: p,
			task: make(chan func(), 1),
		}
	}
	p.cond = sync.NewCond(&p.lock)
	go p.expireWorker()
	return p, nil
}

// 定期清理过期的空闲Worker
func (p *Pool) expireWorker() {
	// 创建一个定时器，每隔 p.expire 时间触发一次
	ticker := time.NewTicker(p.expire)
	for range ticker.C { // 循环监听定时器的通道
		if p.IsClosed() { // 如果线程池已关闭，则退出循环
			break
		}
		p.lock.Lock()             // 加锁，开始操作共享资源
		idleWorkers := p.workers  // 获取当前的空闲工作者列表
		n := len(idleWorkers) - 1 // 获取列表中最后一个元素的索引
		if n >= 0 {               // 如果列表不为空
			var clearN = -1                 // 初始化一个标记，用来记录需要清理的worker的最大索引
			for i, w := range idleWorkers { // 遍历空闲工作者列表
				// 如果当前时间与worker的最后活动时间的差值大于过期时间，则该worker过期
				if time.Now().Sub(w.lastTime) <= p.expire {
					break // 如果遇到未过期的worker，停止检查
				}
				clearN = i           // 更新需要清理的最大索引
				w.task <- nil        // 向worker的任务通道发送nil，触发worker停止
				idleWorkers[i] = nil // 将worker从列表中清除
			}
			// 如果有需要清理的worker
			if clearN != -1 {
				if clearN >= len(idleWorkers)-1 { // 如果清理的是列表中的所有worker
					p.workers = idleWorkers[:0] // 清空worker列表
				} else { // 如果不是清理所有worker
					// 从清理点的下一个开始，保留后面的worker
					p.workers = idleWorkers[clearN+1:]
				}
				// 打印清理完成后的状态
				fmt.Printf("清除完成,running:%d, workers:%v \n", p.running, p.workers)
			}
		}
		p.lock.Unlock() // 解锁
	}
}

// Submit 方法用于将一个任务提交到线程池
func (p *Pool) Submit(task func()) error {
	if len(p.release) > 0 {
		return ErrorHasClosed // 如果池已释放，则返回错误
	}
	w := p.GetWorker()  // 从池中获取一个worker
	w.task <- task      // 将任务发送给worker的任务队列
	w.pool.incRunning() // 增加正在运行的worker计数
	return nil
}

func (p *Pool) GetWorker() *Worker {
	//1. 目的获取pool里面的worker
	//2. 如果 有空闲的worker 直接获取
	p.lock.Lock()
	idleWorkers := p.workers
	n := len(idleWorkers) - 1
	if n >= 0 {
		w := idleWorkers[n]
		idleWorkers[n] = nil
		p.workers = idleWorkers[:n]
		p.lock.Unlock()
		return w
	}
	//3. 如果没有空闲的worker，要新建一个worker
	if p.running < p.cap {
		p.lock.Unlock()
		c := p.workerCache.Get()
		var w *Worker
		//还不够pool的容量，直接新建一个
		if c == nil {
			w = &Worker{
				pool: p,
				task: make(chan func(), 1),
			}
		} else {
			w = c.(*Worker)
		}
		w.run()
		return w
	}
	p.lock.Unlock()
	//4. 如果正在运行的workers 如果大于pool容量，阻塞等待，worker释放
	//for {
	//
	//}
	return p.waitIdleWorker()
}

// 等待空闲的 worker
func (p *Pool) waitIdleWorker() *Worker {
	// 加锁，确保线程安全
	p.lock.Lock()
	// 等待条件变量，直到有空闲 worker
	p.cond.Wait()

	// 获取当前池中的所有空闲 worker
	idleWorkers := p.workers
	// 获取最后一个空闲 worker 的索引
	n := len(idleWorkers) - 1
	// 如果没有空闲 worker
	if n < 0 {
		// 解锁
		p.lock.Unlock()
		// 如果当前运行的 worker 数量小于池的容量
		if p.running < p.cap {
			// 从缓存中获取一个 worker
			c := p.workerCache.Get()
			var w *Worker
			// 如果缓存中没有 worker，则新建一个
			if c == nil {
				w = &Worker{
					pool: p,
					task: make(chan func(), 1),
				}
			} else {
				// 如果缓存中有，则使用缓存中的 worker
				w = c.(*Worker)
			}
			// 运行这个 worker
			w.run()
			// 返回这个新创建的 worker
			return w
		}
		// 如果池已经满了，递归等待空闲的 worker
		return p.waitIdleWorker()
	}
	// 获取最后一个空闲的 worker
	w := idleWorkers[n]
	// 将这个 worker 从空闲列表中移除
	idleWorkers[n] = nil
	p.workers = idleWorkers[:n]
	// 解锁
	p.lock.Unlock()
	// 返回这个空闲的 worker
	return w
}

func (p *Pool) incRunning() {
	atomic.AddInt32(&p.running, 1)
}

// PutWorker 将 worker 放入池中
func (p *Pool) PutWorker(w *Worker) {
	// 设置 worker 的最后活跃时间为当前时间
	w.lastTime = time.Now()
	// 加锁，确保线程安全
	p.lock.Lock()
	// 将 worker 添加到池的 workers 切片中
	p.workers = append(p.workers, w)
	// 发送信号通知其他等待的 goroutine 有新的 worker 可用
	p.cond.Signal()
	// 解锁
	p.lock.Unlock()
}

// 减少运行中的 worker 数量
func (p *Pool) decRunning() {
	// 使用原子操作减少 p.running 的值
	atomic.AddInt32(&p.running, -1)
}

// Release 释放池中的所有资源
func (p *Pool) Release() {
	// 确保下面的代码只执行一次
	p.once.Do(func() {
		// 加锁，确保线程安全
		p.lock.Lock()
		// 获取当前池中的所有 workers
		workers := p.workers
		// 遍历每个 worker
		for i, w := range workers {
			// 将每个 worker 的任务置空
			w.task = nil
			// 将每个 worker 的池引用置空
			w.pool = nil
			// 将 worker 在切片中的引用置空
			workers[i] = nil
		}
		// 将池中的 workers 切片置空
		p.workers = nil
		// 解锁
		p.lock.Unlock()
		// 向 release 通道发送信号，表示释放操作已完成
		p.release <- sig{}
	})
}

// IsClosed 判断池是否已关闭
func (p *Pool) IsClosed() bool {
	// 如果 release 通道中有信号，表示池已关闭
	return len(p.release) > 0
}

// Restart 重启池
func (p *Pool) Restart() bool {
	// 如果 release 通道中没有信号，表示池未关闭，直接返回 true
	if len(p.release) <= 0 {
		return true
	}
	// 从 release 通道接收一个信号，表示释放已完成，可以重启
	_ = <-p.release
	return true
}

func (p *Pool) Running() int {
	return int(atomic.LoadInt32(&p.running))
}

func (p *Pool) Free() int {
	return int(p.cap - p.running)
}
