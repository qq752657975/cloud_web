package pool

import (
	myLog "github.com/ygb616/web/log"
	"time"
)

type Worker struct {
	pool *Pool
	//task 任务队列
	task chan func()
	//lastTime 执行任务的最后的时间
	lastTime time.Time
}

func (w *Worker) run() {
	w.pool.incRunning() // 将池中正在运行的 worker 数量加一
	go w.running()
}

// 运行 worker 的任务循环
func (w *Worker) running() {
	defer func() {
		// 减少池中正在运行的 worker 数量
		w.pool.decRunning()
		// 将当前 worker 放入池的缓存中
		w.pool.workerCache.Put(w)
		// 捕获任务发生的 panic
		if err := recover(); err != nil {
			// 如果池中定义了 panic 处理函数，调用它
			if w.pool.PanicHandler != nil {
				w.pool.PanicHandler()
			} else {
				// 否则，记录错误日志
				myLog.Default().Error(err)
			}
		}
		// 发送信号，通知其他等待的 goroutine
		w.pool.cond.Signal()
	}()

	// 无限循环监听任务通道，当通道被关闭时，循环会自动结束
	for f := range w.task {
		if f == nil {
			// 如果从任务通道中接收到 nil，表示需要停止此 worker
			w.pool.workerCache.Put(w) // 将此 worker 放入池的缓存中，可能用于快速重用
			return                    // 结束此方法，停止当前 goroutine
		}
		// 调用接收到的函数，执行实际的任务
		f()

		// 任务运行完成后，以下代码处理 worker 的状态
		w.pool.PutWorker(w) // 将 worker 放回池中，标记为空闲

	}
}
