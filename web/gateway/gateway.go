package gateway

import "net/http"

// GWConfig 定义了网关配置结构体
type GWConfig struct {
	Name        string                  // 服务名称
	Path        string                  // 路径
	Host        string                  // 主机地址
	Port        int                     // 端口号
	Header      func(req *http.Request) // 处理请求头的函数
	ServiceName string                  // 服务名称
}
