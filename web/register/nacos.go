package register

import (
	"fmt"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

func CreateNacosClient() (naming_client.INamingClient, error) {
	// 创建 clientConfig 的另一种方式
	clientConfig := *constant.NewClientConfig(
		constant.WithNamespaceId(""),              // 当 namespace 是 public 时，此处填空字符串。
		constant.WithTimeoutMs(5000),              // 设置超时时间为 5000 毫秒
		constant.WithNotLoadCacheAtStart(true),    // 不在启动时加载缓存
		constant.WithLogDir("/tmp/nacos/log"),     // 设置日志目录
		constant.WithCacheDir("/tmp/nacos/cache"), // 设置缓存目录
		constant.WithLogLevel("debug"),            // 设置日志级别为 debug
	)

	// 创建 serverConfig 的另一种方式
	serverConfigs := []constant.ServerConfig{
		*constant.NewServerConfig(
			"127.0.0.1",                        // 服务器 IP 地址
			8848,                               // 端口号
			constant.WithScheme("http"),        // 使用 HTTP 协议
			constant.WithContextPath("/nacos"), // 设置上下文路径
		),
	}

	// 创建服务发现客户端
	// 创建服务发现客户端的另一种方式 (推荐)
	namingClient, err := clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig, // 客户端配置
			ServerConfigs: serverConfigs, // 服务器配置
		},
	)
	if err != nil {
		return nil, err // 如果创建客户端失败，返回错误
	}
	return namingClient, nil // 返回创建的命名客户端
}

func RegService(namingClient naming_client.INamingClient, serviceName string, host string, port int) error {
	// 注册服务实例
	_, err := namingClient.RegisterInstance(vo.RegisterInstanceParam{
		Ip:          host,                                 // 实例的 IP 地址
		Port:        uint64(port),                         // 实例的端口号
		ServiceName: serviceName,                          // 服务名称
		Weight:      10,                                   // 实例的权重
		Enable:      true,                                 // 实例是否启用
		Healthy:     true,                                 // 实例是否健康
		Ephemeral:   true,                                 // 实例是否为临时实例
		Metadata:    map[string]string{"idc": "shanghai"}, // 实例的元数据
		//ClusterName: "cluster-a",            // 集群名称，默认值为 DEFAULT
		//GroupName:   "group-a",              // 组名称，默认值为 DEFAULT_GROUP
	})

	return err // 返回注册结果中的错误信息
}

func GetInstance(namingClient naming_client.INamingClient, serviceName string) (string, uint64, error) {
	// SelectOneHealthyInstance 将会按加权随机轮询的负载均衡策略返回一个健康的实例
	// 实例必须满足的条件：health=true, enable=true 和 weight>0
	instance, err := namingClient.SelectOneHealthyInstance(vo.SelectOneHealthInstanceParam{
		ServiceName: serviceName, // 服务名称
		//GroupName:   "group-a",             // 组名称，默认值为 DEFAULT_GROUP
		//Clusters:    []string{"cluster-a"}, // 集群名称，默认值为 DEFAULT
	})
	if err != nil {
		return "", uint64(0), err // 如果获取实例失败，返回错误
	}
	return instance.Ip, instance.Port, nil // 返回实例的 IP 和端口号
}

// CreateCli(option Option)  error
// RegisterService(serviceName string, host string, port int) error
// GetValue(serviceName string) (string, error)
// Close() error

type MsNacosRegister struct {
	cli naming_client.INamingClient // Nacos 客户端
}

func (r *MsNacosRegister) CreateCli(option Option) error {
	// 创建 clientConfig 的另一种方式
	// clientConfig := *constant.NewClientConfig(
	//    constant.WithNamespaceId(""), // 当 namespace 是 public 时，此处填空字符串。
	//    constant.WithTimeoutMs(5000),
	//    constant.WithNotLoadCacheAtStart(true),
	//    constant.WithLogDir("/tmp/nacos/log"),
	//    constant.WithCacheDir("/tmp/nacos/cache"),
	//    constant.WithLogLevel("debug"),
	// )

	// 创建服务发现客户端
	// 创建服务发现客户端的另一种方式（推荐）
	namingClient, err := clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  option.NacosClientConfig, // Nacos 客户端配置
			ServerConfigs: option.NacosServerConfig, // Nacos 服务器配置
		},
	)
	if err != nil {
		return err // 返回错误
	}
	r.cli = namingClient // 赋值客户端
	return nil           // 返回 nil 表示成功
}

func (r *MsNacosRegister) RegisterService(serviceName string, host string, port int) error {
	// 注册服务实例
	_, err := r.cli.RegisterInstance(vo.RegisterInstanceParam{
		Ip:          host,                                 // 实例的 IP 地址
		Port:        uint64(port),                         // 实例的端口号
		ServiceName: serviceName,                          // 服务名称
		Weight:      10,                                   // 实例的权重
		Enable:      true,                                 // 实例是否启用
		Healthy:     true,                                 // 实例是否健康
		Ephemeral:   true,                                 // 实例是否为临时实例
		Metadata:    map[string]string{"idc": "shanghai"}, // 实例的元数据
		// ClusterName: "cluster-a",            // 集群名称，默认值为 DEFAULT
		// GroupName:   "group-a",              // 组名称，默认值为 DEFAULT_GROUP
	})
	return err // 返回注册结果中的错误信息
}

func (r *MsNacosRegister) GetValue(serviceName string) (string, error) {
	// 选择一个健康的实例
	instance, err := r.cli.SelectOneHealthyInstance(vo.SelectOneHealthInstanceParam{
		ServiceName: serviceName, // 服务名称
		// GroupName:   "group-a",             // 组名称，默认值为 DEFAULT_GROUP
		// Clusters:    []string{"cluster-a"}, // 集群名称，默认值为 DEFAULT
	})
	if err != nil {
		return "", err // 如果获取实例失败，返回错误
	}
	// 返回实例的 IP 和端口号
	return fmt.Sprintf("%s:%d", instance.Ip, instance.Port), nil
}

func (r *MsNacosRegister) Close() error {
	// 关闭客户端
	return nil
}
