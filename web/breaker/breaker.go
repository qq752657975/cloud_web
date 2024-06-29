package breaker

import (
	"errors"
	"sync"
	"time"
)

// State 状态
type State int

const (
	StateClosed   State = iota // 关闭状态
	StateHalfOpen              // 半开状态
	StateOpen                  // 打开状态
)

// Counts 计数器结构体
type Counts struct {
	Requests             uint32 // 请求数量
	TotalSuccesses       uint32 // 总成功数
	TotalFailures        uint32 // 总失败数
	ConsecutiveSuccesses uint32 // 连续成功数量
	ConsecutiveFailures  uint32 // 连续失败数量
}

// OnRequest 增加请求数量
func (c *Counts) OnRequest() {
	c.Requests++ // 请求数加一
}

// OnSuccess 记录成功请求
func (c *Counts) OnSuccess() {
	c.TotalSuccesses++        // 总成功数加一
	c.ConsecutiveSuccesses++  // 连续成功数加一
	c.ConsecutiveFailures = 0 // 连续失败数重置为零
}

// OnFail 记录失败请求
func (c *Counts) OnFail() {
	c.TotalFailures++          // 总失败数加一
	c.ConsecutiveFailures++    // 连续失败数加一
	c.ConsecutiveSuccesses = 0 // 连续成功数重置为零
}

// Clear 清空计数器
func (c *Counts) Clear() {
	c.Requests = 0             // 请求数重置为零
	c.TotalSuccesses = 0       // 总成功数重置为零
	c.ConsecutiveSuccesses = 0 // 连续成功数重置为零
	c.TotalFailures = 0        // 总失败数重置为零
	c.ConsecutiveFailures = 0  // 连续失败数重置为零
}

// Settings 熔断器设置
type Settings struct {
	Name          string                                  // 名字
	MaxRequests   uint32                                  // 最大请求数
	Interval      time.Duration                           // 间隔时间
	Timeout       time.Duration                           // 超时时间
	ReadyToTrip   func(counts Counts) bool                // 执行熔断
	OnStateChange func(name string, from State, to State) // 状态变更回调
	IsSuccessful  func(err error) bool                    // 判断是否成功
	Fallback      func(err error) (any, error)            // 回退函数
}

// CircuitBreaker 断路器
type CircuitBreaker struct {
	name          string                                  // 名字
	maxRequests   uint32                                  // 最大请求数，当连续请求成功数大于此时，断路器关闭
	interval      time.Duration                           // 间隔时间
	timeout       time.Duration                           // 超时时间
	readyToTrip   func(counts Counts) bool                // 是否执行熔断
	isSuccessful  func(err error) bool                    // 判断请求是否成功
	onStateChange func(name string, from State, to State) // 状态变更回调

	mutex      sync.Mutex                   // 互斥锁，用于保护并发访问
	state      State                        // 当前状态
	generation uint64                       // 当前代数，每次状态变更时增加
	counts     Counts                       // 计数器，记录请求数量和成功失败情况
	expiry     time.Time                    // 到期时间，用于检查是否从开到半开
	fallback   func(err error) (any, error) // 回退函数，当请求失败时调用
}

// NewGeneration 创建新的代数并清除计数器
func (cb *CircuitBreaker) NewGeneration() {
	cb.mutex.Lock()         // 加锁，防止并发访问
	defer cb.mutex.Unlock() // 函数退出时解锁
	cb.generation++         // 增加当前代数
	cb.counts.Clear()       // 清空计数器
	var zero time.Time
	switch cb.state {
	case StateClosed:
		// 如果状态为关闭，根据 interval 设置到期时间
		if cb.interval == 0 {
			cb.expiry = zero
		} else {
			cb.expiry = time.Now().Add(cb.interval)
		}
	case StateOpen:
		// 如果状态为打开，根据 timeout 设置到期时间
		cb.expiry = time.Now().Add(cb.timeout)
	case StateHalfOpen:
		// 如果状态为半开，设置到期时间为零
		cb.expiry = zero
	}
}

// NewCircuitBreaker 创建一个新的断路器实例
func NewCircuitBreaker(st Settings) *CircuitBreaker {
	cb := new(CircuitBreaker)           // 创建一个新的 CircuitBreaker 实例
	cb.name = st.Name                   // 设置断路器的名称
	cb.onStateChange = st.OnStateChange // 设置状态变更回调函数
	cb.fallback = st.Fallback           // 设置回退函数

	// 设置最大请求数，默认为 1
	if st.MaxRequests == 0 {
		cb.maxRequests = 1
	} else {
		cb.maxRequests = st.MaxRequests
	}

	// 设置间隔时间，默认为 0 秒
	if st.Interval == 0 {
		cb.interval = time.Duration(0) * time.Second
	} else {
		cb.interval = st.Interval
	}

	// 设置超时时间，默认为 20 秒
	if st.Timeout == 0 {
		cb.timeout = time.Duration(20) * time.Second
	} else {
		cb.timeout = st.Timeout
	}

	// 设置熔断条件，默认为连续失败次数大于 5
	if st.ReadyToTrip == nil {
		cb.readyToTrip = func(counts Counts) bool {
			return counts.ConsecutiveFailures > 5
		}
	} else {
		cb.readyToTrip = st.ReadyToTrip
	}

	// 设置判断请求是否成功的函数，默认为错误为空时成功
	if st.IsSuccessful == nil {
		cb.isSuccessful = func(err error) bool {
			return err == nil
		}
	} else {
		cb.isSuccessful = st.IsSuccessful
	}

	cb.NewGeneration() // 初始化新的代数
	return cb          // 返回断路器实例
}

// Execute 执行传入的请求函数，并根据断路器的状态处理请求和返回结果
func (cb *CircuitBreaker) Execute(req func() (any, error)) (any, error) {
	// 请求之前判断断路器状态
	err, generation := cb.beforeRequest()
	if err != nil {
		// 如果断路器打开或请求过多，执行回退函数
		if cb.fallback != nil {
			return cb.fallback(err)
		}
		return nil, err
	}

	// 执行请求函数
	result, err := req()
	cb.counts.OnRequest() // 增加请求计数

	// 请求之后，判断是否需要变更断路器状态
	cb.afterRequest(generation, cb.isSuccessful(err))
	return result, err
}

// beforeRequest 在请求执行前判断断路器的当前状态并进行处理
func (cb *CircuitBreaker) beforeRequest() (error, uint64) {
	now := time.Now()
	state, generation := cb.currentState(now) // 获取当前断路器状态及代数

	// 如果断路器是打开状态，返回错误
	if state == StateOpen {
		return errors.New("断路器是打开状态"), generation
	}

	// 如果断路器是半开状态且请求数量超过最大请求数，返回错误
	if state == StateHalfOpen {
		if cb.counts.Requests > cb.maxRequests {
			return errors.New("请求数量过多"), generation
		}
	}

	// 返回 nil 表示可以继续请求
	return nil, generation
}

// afterRequest 在请求执行后，根据请求结果（成功或失败）更新断路器的状态
func (cb *CircuitBreaker) afterRequest(before uint64, success bool) {
	now := time.Now()
	state, generation := cb.currentState(now) // 获取当前断路器状态及代数
	if generation != before {
		// 如果当前代数与请求之前的代数不同，直接返回
		return
	}
	if success {
		// 请求成功，调用 OnSuccess 更新断路器状态
		cb.OnSuccess(state)
	} else {
		// 请求失败，调用 OnFail 更新断路器状态
		cb.OnFail(state)
	}
}

// currentState 获取断路器的当前状态及代数
func (cb *CircuitBreaker) currentState(now time.Time) (State, uint64) {
	switch cb.state {
	case StateClosed:
		// 如果断路器是关闭状态，检查是否需要开启新的一代
		if !cb.expiry.IsZero() && cb.expiry.Before(now) {
			cb.NewGeneration() // 开启新的一代
		}
	case StateOpen:
		// 如果断路器是打开状态，检查是否需要变为半开状态
		if cb.expiry.Before(now) {
			cb.SetState(StateHalfOpen) // 设置为半开状态
		}
	default:
		// 如果遇到未处理的状态，抛出异常
		panic("unhandled default case")
	}
	return cb.state, cb.generation
}

// SetState 设置断路器的状态
func (cb *CircuitBreaker) SetState(target State) {
	if cb.state == target {
		return // 如果目标状态与当前状态相同，直接返回
	}
	before := cb.state // 记录状态变更前的状态
	cb.state = target  // 设置新的目标状态
	// 状态变更之后，重新计数
	cb.NewGeneration()

	if cb.onStateChange != nil {
		// 如果设置了状态变更回调函数，调用该函数
		cb.onStateChange(cb.name, before, target)
	}
}

// OnSuccess 处理成功的请求，根据状态进行处理
func (cb *CircuitBreaker) OnSuccess(state State) {
	switch state {
	case StateClosed:
		cb.counts.OnSuccess() // 记录成功请求
	case StateHalfOpen:
		cb.counts.OnSuccess() // 记录成功请求
		// 如果连续成功请求数大于最大请求数，关闭断路器
		if cb.counts.ConsecutiveSuccesses > cb.maxRequests {
			cb.SetState(StateClosed) // 设置断路器为关闭状态
		}
	default:
		panic("unhandled default case") // 未处理的状态抛出异常
	}
}

// OnFail 处理失败的请求，根据状态进行处理
func (cb *CircuitBreaker) OnFail(state State) {
	switch state {
	case StateClosed:
		cb.counts.OnFail() // 记录失败请求
		// 如果满足触发熔断的条件，打开断路器
		if cb.readyToTrip(cb.counts) {
			cb.SetState(StateOpen) // 设置断路器为打开状态
		}
	case StateHalfOpen:
		cb.SetState(StateOpen) // 半开状态下，失败则打开断路器
	default:
		panic("unhandled default case") // 未处理的状态抛出异常
	}
}
