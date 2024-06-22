package pool

import (
	"math"    // 导入数学包
	"runtime" // 导入运行时包，用于获取内存统计等信息
	"sync"    // 导入同步包，用于 WaitGroup 等同步原语
	"testing" // 导入测试包，用于编写测试代码
	"time"    // 导入时间包，用于处理时间相关操作
)

const (
	_   = 1 << (10 * iota) // 忽略第一个值，iota 从 0 开始递增，每次递增 1
	KiB                    // 1024             // 定义 KiB 为 1024 字节
	MiB                    // 1048576          // 定义 MiB 为 1024 KiB，即 1048576 字节
	// GiB // 1073741824     // 定义 GiB 为 1024 MiB，但这里被注释掉了
	// TiB // 1099511627776  // 定义 TiB 为 1024 GiB，超过了 int32 的范围，被注释掉了
	// PiB // 1125899906842624 // 定义 PiB 为 1024 TiB，被注释掉了
	// EiB // 1152921504606846976 // 定义 EiB 为 1024 PiB，被注释掉了
	// ZiB // 1180591620717411303424 // 定义 ZiB 为 1024 EiB，超过了 int64 的范围，被注释掉了
	// YiB // 1208925819614629174706176 // 定义 YiB 为 1024 ZiB，被注释掉了
)

const (
	Param    = 100     // 定义常量 Param，值为 100
	PoolSize = 1000    // 定义常量 PoolSize，值为 1000
	TestSize = 10000   // 定义常量 TestSize，值为 10000
	n        = 1000000 // 定义常量 n，值为 1000000
)

var curMem uint64 // 定义全局变量 curMem，用于存储当前内存使用量

const (
	RunTimes           = 1000000          // 定义常量 RunTimes，值为 1000000
	BenchParam         = 10               // 定义常量 BenchParam，值为 10
	DefaultExpiredTime = 10 * time.Second // 定义常量 DefaultExpiredTime，值为 10 秒
)

func demoFunc() {
	time.Sleep(time.Duration(BenchParam) * time.Millisecond) // 休眠 BenchParam 毫秒
}

func TestNoPool(t *testing.T) {
	var wg sync.WaitGroup // 定义 WaitGroup，用于等待一组协程完成
	for i := 0; i < n; i++ {
		wg.Add(1) // 增加 WaitGroup 计数器
		go func() {
			demoFunc() // 调用 demoFunc
			wg.Done()  // 完成一个协程，减少 WaitGroup 计数器
		}()
	}

	wg.Wait() // 等待所有协程完成
	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)           // 读取当前内存统计信息
	curMem = mem.TotalAlloc/MiB - curMem // 计算当前内存使用量
	t.Logf("memory usage:%d MB", curMem) // 打印内存使用量
}

func TestHasPool(t *testing.T) {
	pool, _ := NewPool(math.MaxInt32) // 创建一个新的协程池，大小为 math.MaxInt32
	defer pool.Release()              // 延迟释放协程池
	var wg sync.WaitGroup             // 定义 WaitGroup，用于等待一组协程完成
	for i := 0; i < n; i++ {
		wg.Add(1) // 增加 WaitGroup 计数器
		_ = pool.Submit(func() {
			demoFunc() // 调用 demoFunc
			wg.Done()  // 完成一个协程，减少 WaitGroup 计数器
		})
	}
	wg.Wait() // 等待所有协程完成

	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)                  // 读取当前内存统计信息
	curMem = mem.TotalAlloc/MiB - curMem        // 计算当前内存使用量
	t.Logf("memory usage:%d MB", curMem)        // 打印内存使用量
	t.Logf("running worker:%d", pool.Running()) // 打印正在运行的协程数
	t.Logf("free worker:%d ", pool.Free())      // 打印空闲的协程数
}
