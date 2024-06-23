package config

import (
	"flag"                            // 引入 flag 包，用于解析命令行参数
	"github.com/BurntSushi/toml"      // 引入 toml 包，用于解析 TOML 格式的配置文件
	myLog "github.com/ygb616/web/log" // 引入自定义的日志包
	"os"                              // 引入 os 包，用于文件系统操作
)

// Conf 是全局的配置实例，初始化为默认配置
var conf = &WebConfig{
	logger: myLog.Default(), // 使用默认的日志记录器
}

// WebConfig 结构体用于存储应用的各种配置
type WebConfig struct {
	logger   *myLog.Logger  // 日志记录器
	Log      map[string]any // 日志相关配置
	Pool     map[string]any // 连接池相关配置
	Template map[string]any // 模板相关配置
	Mysql    map[string]any //数据库相关配置
}

// init 函数在包初始化时自动调用，用于加载配置文件
func init() {
	loadToml() // 加载 TOML 配置文件
}

// loadToml 函数加载 TOML 配置文件
func loadToml() {
	// 定义命令行参数，用于指定配置文件路径，默认值为 "conf/app.toml"
	configFile := flag.String("conf", "conf/app.toml", "app config file")
	flag.Parse() // 解析命令行参数

	// 检查配置文件是否存在
	if _, err := os.Stat(*configFile); err != nil {
		// 如果文件不存在，记录日志并返回
		conf.logger.Info("conf/app.toml file not load，because not exist")
		return
	}

	// 解析配置文件并将结果存储到 Conf 变量中
	_, err := toml.DecodeFile(*configFile, conf)
	if err != nil {
		// 如果解析失败，记录日志并返回
		conf.logger.Info("conf/app.toml decode fail check format")
		return
	}
}

func GetToml() *WebConfig {
	return conf
}
