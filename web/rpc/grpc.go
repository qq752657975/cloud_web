package rpc

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"net"
	"time"
)

// MsGrpcServer 定义了 gRPC 服务器结构体
type MsGrpcServer struct {
	listen   net.Listener           // 网络监听器
	g        *grpc.Server           // gRPC 服务器实例
	register []func(g *grpc.Server) // 注册函数切片
	ops      []grpc.ServerOption    // gRPC 服务器选项切片
}

// NewGrpcServer 创建新的 gRPC 服务器
func NewGrpcServer(addr string, ops ...MsGrpcOption) (*MsGrpcServer, error) {
	listen, err := net.Listen("tcp", addr) // 创建 TCP 监听器
	if err != nil {                        // 如果监听器创建失败
		return nil, err // 返回错误
	}
	ms := &MsGrpcServer{}   // 创建 MsGrpcServer 实例
	ms.listen = listen      // 赋值监听器
	for _, v := range ops { // 应用所有传入的选项
		v.Apply(ms)
	}
	server := grpc.NewServer(ms.ops...) // 创建 gRPC 服务器实例
	ms.g = server                       // 赋值 gRPC 服务器
	return ms, nil                      // 返回 MsGrpcServer 实例
}

// Run 方法启动 gRPC 服务器
func (s *MsGrpcServer) Run() error {
	for _, f := range s.register { // 执行所有注册函数
		f(s.g)
	}
	return s.g.Serve(s.listen) // 启动 gRPC 服务器
}

// Stop 方法停止 gRPC 服务器
func (s *MsGrpcServer) Stop() {
	s.g.Stop() // 停止 gRPC 服务器
}

// Register 方法用于注册 gRPC 服务
func (s *MsGrpcServer) Register(f func(g *grpc.Server)) {
	s.register = append(s.register, f) // 添加注册函数到切片
}

// MsGrpcOption 接口定义了 Apply 方法，用于应用选项
type MsGrpcOption interface {
	Apply(s *MsGrpcServer)
}

// DefaultMsGrpcOption 默认的 gRPC 选项结构体
type DefaultMsGrpcOption struct {
	f func(s *MsGrpcServer) // 应用选项的函数
}

// Apply 方法应用选项到 MsGrpcServer
func (d *DefaultMsGrpcOption) Apply(s *MsGrpcServer) {
	d.f(s)
}

// WithGrpcOptions 创建 gRPC 选项
func WithGrpcOptions(ops ...grpc.ServerOption) MsGrpcOption {
	return &DefaultMsGrpcOption{
		f: func(s *MsGrpcServer) {
			s.ops = append(s.ops, ops...) // 添加 gRPC 服务器选项
		},
	}
}

// MsGrpcClient 定义了 gRPC 客户端结构体
type MsGrpcClient struct {
	Conn *grpc.ClientConn // gRPC 客户端连接
}

// NewGrpcClient 创建新的 gRPC 客户端
func NewGrpcClient(config *MsGrpcClientConfig) (*MsGrpcClient, error) {
	var ctx = context.Background()       // 创建背景上下文
	var dialOptions = config.dialOptions // 获取拨号选项

	if config.Block { // 如果需要阻塞连接
		if config.DialTimeout > time.Duration(0) { // 如果设置了拨号超时时间
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, config.DialTimeout) // 创建带超时的上下文
			defer cancel()                                             // 在函数退出时取消上下文
		}
		dialOptions = append(dialOptions, grpc.WithBlock()) // 添加阻塞选项
	}
	if config.KeepAlive != nil { // 如果设置了 KeepAlive 参数
		dialOptions = append(dialOptions, grpc.WithKeepaliveParams(*config.KeepAlive)) // 添加 KeepAlive 参数
	}
	conn, err := grpc.DialContext(ctx, config.Address, dialOptions...) // 创建 gRPC 客户端连接
	if err != nil {                                                    // 如果连接创建失败
		return nil, err // 返回错误
	}
	return &MsGrpcClient{
		Conn: conn,
	}, nil // 返回 MsGrpcClient 实例
}

// MsGrpcClientConfig 定义了 gRPC 客户端配置结构体
type MsGrpcClientConfig struct {
	Address     string                      // 服务器地址
	Block       bool                        // 是否阻塞
	DialTimeout time.Duration               // 拨号超时时间
	ReadTimeout time.Duration               // 读取超时时间
	Direct      bool                        // 是否直连
	KeepAlive   *keepalive.ClientParameters // KeepAlive 参数
	dialOptions []grpc.DialOption           // 拨号选项切片
}

// DefaultGrpcClientConfig 返回默认的 gRPC 客户端配置
func DefaultGrpcClientConfig() *MsGrpcClientConfig {
	return &MsGrpcClientConfig{
		dialOptions: []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()), // 不安全的传输凭证
		},
		DialTimeout: time.Second * 3, // 默认拨号超时时间 3 秒
		ReadTimeout: time.Second * 2, // 默认读取超时时间 2 秒
		Block:       true,            // 默认阻塞连接
	}
}
