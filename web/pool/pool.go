package pool

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

type sig struct{}

const DefaultExpire = 3

var (
	ErrorInValidCap    = errors.New("pool cap can not <= 0")
	ErrorInValidExpire = errors.New("pool expire can not <= 0")
	ErrorHasClosed     = errors.New("pool has bean released")
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
	go expireWorker()
	return p, nil
}

func expireWorker() {
	//定时清理过期的空闲worker

}

func (p *Pool) Submit(task func()) error {
	if len(p.release) > 0 {
		return ErrorHasClosed
	}
	//获取池里面的一个worker，然后执行任务就可以了
	w := p.GetWorker()
	w.task <- task
	w.pool.incRunning()
	return nil
}

func (p *Pool) incRunning() {
	atomic.AddInt32(&p.running, 1)
}

func (p *Pool) GetWorker() *Worker {
	return nil
}
