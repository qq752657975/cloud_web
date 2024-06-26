package rpc

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"
)

const (
	GET      = "GET"
	POSTForm = "POST_FORM"
	POSTJson = "POST_JSON"
	HTTP     = "http"
	HTTPS    = "https"
)

// MsHttpClient 结构体定义了一个自定义的 HTTP 客户端
type MsHttpClient struct {
	client     http.Client          // 嵌入 http.Client 对象，用于发送 HTTP 请求
	serviceMap map[string]MsService // 服务映射表，存储服务名称和对应的 MsService 实例
}

// MsService 接口定义了一个服务应该实现的方法
type MsService interface {
	Env() HttpConfig // 定义一个 Env 方法，返回 HttpConfig 类型
}

func (c *MsHttpClient) RegisterHttpService(name string, service MsService) {
	c.serviceMap[name] = service
}

// Session 方法用于创建一个新的 MsHttpClientSession 实例
func (c *MsHttpClient) Session() *MsHttpClientSession {
	// 返回一个新的 MsHttpClientSession 实例，初始化时包含当前的 MsHttpClient 实例
	return &MsHttpClientSession{
		c,   // 将当前的 MsHttpClient 实例传递给 MsHttpClientSession
		nil, // 初始化其他字段为 nil
	}
}

// HttpConfig 结构体定义了 HTTP 服务的配置信息
type HttpConfig struct {
	Protocol string // 协议，例如 "http" 或 "https"
	Host     string // 主机地址，例如 "localhost" 或 "example.com"
	Port     int    // 端口号，例如 80 或 443
}

type MsHttpClientSession struct {
	*MsHttpClient
	ReqHandler func(req *http.Request)
}

// NewHttpClient 方法用于创建一个新的 HTTP 客户端
func NewHttpClient() *MsHttpClient {
	// 创建一个 http.Client 对象，并设置相关参数
	client := http.Client{
		Timeout: time.Duration(3) * time.Second, // 设置请求超时时间为 3 秒
		Transport: &http.Transport{ // 配置请求分发的 Transport
			MaxIdleConnsPerHost:   5,                // 每个主机的最大空闲连接数为 5
			MaxConnsPerHost:       100,              // 每个主机的最大连接数为 100
			IdleConnTimeout:       90 * time.Second, // 空闲连接的超时时间为 90 秒
			TLSHandshakeTimeout:   10 * time.Second, // TLS 握手的超时时间为 10 秒
			ExpectContinueTimeout: 1 * time.Second,  // 100-continue 状态码的超时时间为 1 秒
		},
	}
	// 返回一个新的 MsHttpClient 对象，其中包含配置好的 http.Client 对象和一个空的 serviceMap
	return &MsHttpClient{client: client, serviceMap: make(map[string]MsService)}
}

// GetRequest 方法用于创建 GET 请求或其他带查询参数的请求
func (c *MsHttpClient) GetRequest(method string, url string, args map[string]any) (*http.Request, error) {
	if args != nil && len(args) > 0 { // 如果参数不为空且长度大于0
		url = url + "?" + c.toValues(args) // 将参数编码为查询字符串并附加到 URL
	}
	req, err := http.NewRequest(method, url, nil) // 创建新的 HTTP 请求，方法为 GET 或其他指定方法
	if err != nil {                               // 如果创建请求时发生错误
		return nil, err // 返回错误
	}
	return req, nil // 返回创建的请求和 nil 错误
}

// FormRequest 方法用于创建带表单数据的请求
func (c *MsHttpClient) FormRequest(method string, url string, args map[string]any) (*http.Request, error) {
	req, err := http.NewRequest(method, url, strings.NewReader(c.toValues(args))) // 创建新的 HTTP 请求，方法为指定方法，内容为表单数据
	if err != nil {                                                               // 如果创建请求时发生错误
		return nil, err // 返回错误
	}
	return req, nil // 返回创建的请求和 nil 错误
}

// JsonRequest 方法用于创建带 JSON 数据的请求
func (c *MsHttpClient) JsonRequest(method string, url string, args map[string]any) (*http.Request, error) {
	jsonStr, _ := json.Marshal(args)                                   // 将参数编码为 JSON 字符串
	req, err := http.NewRequest(method, url, bytes.NewReader(jsonStr)) // 创建新的 HTTP 请求，方法为指定方法，内容为 JSON 数据
	if err != nil {                                                    // 如果创建请求时发生错误
		return nil, err // 返回错误
	}
	return req, nil // 返回创建的请求和 nil 错误
}

// Get 方法用于发送 GET 请求
func (c *MsHttpClient) Get(url string, args map[string]any) ([]byte, error) {
	if args != nil && len(args) > 0 { // 如果参数不为空且长度大于0
		url = url + "?" + c.toValues(args) // 将参数编码为查询字符串并附加到 URL
	}
	req, err := http.NewRequest("GET", url, nil) // 创建新的 GET 请求
	if err != nil {                              // 如果创建请求时发生错误
		return nil, err // 返回错误
	}
	return c.handleResponse(req) // 处理请求并返回响应
}

// Response 方法用于处理 HTTP 请求并返回响应
func (c *MsHttpClient) Response(req *http.Request) ([]byte, error) {
	return c.handleResponse(req) // 调用 handleResponse 方法处理请求并返回响应
}

// handleResponse 方法用于处理 HTTP 响应
func (c *MsHttpClient) handleResponse(req *http.Request) ([]byte, error) {
	var err error                     // 声明错误变量
	response, err := c.client.Do(req) // 发送请求并获取响应
	if err != nil {                   // 如果发送请求时发生错误
		return nil, err // 返回错误
	}
	if response.StatusCode != 200 { // 如果响应状态码不是 200
		return nil, errors.New(response.Status) // 返回状态码错误
	}
	buffLen := 79                            // 定义缓冲区长度
	buff := make([]byte, buffLen)            // 创建缓冲区
	body := make([]byte, 0)                  // 创建用于存储响应体的切片
	reader := bufio.NewReader(response.Body) // 创建新的读取器，读取响应体
	for {                                    // 循环读取响应体
		n, err := reader.Read(buff)  // 读取缓冲区
		if err == io.EOF || n == 0 { // 如果读取到文件结束或没有更多数据
			break // 退出循环
		}
		body = append(body, buff[:n]...) // 将缓冲区数据追加到响应体
		if n < buffLen {                 // 如果读取的数据小于缓冲区长度
			break // 退出循环
		}
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(response.Body) // 确保在函数返回前关闭响应体
	//if err != nil {             // 如果读取响应体时发生错误
	//	return nil, err // 返回错误
	//}
	return body, nil // 返回响应体
}

// toValues 方法用于将参数转换为查询字符串
func (c *MsHttpClient) toValues(args map[string]any) string {
	if args != nil && len(args) > 0 { // 如果参数不为空且长度大于0
		params := url.Values{}   // 创建 URL 参数对象
		for k, v := range args { // 遍历参数
			params.Set(k, fmt.Sprintf("%v", v)) // 将参数设置为查询字符串
		}
		return params.Encode() // 返回编码后的查询字符串
	}
	return "" // 如果没有参数，返回空字符串
}

// PostForm 方法用于发送 POST 表单请求
func (c *MsHttpClient) PostForm(url string, args map[string]any) ([]byte, error) {
	// 创建 POST 请求，内容为表单数据
	req, err := http.NewRequest("POST", url, strings.NewReader(c.toValues(args)))
	if err != nil { // 如果创建请求时发生错误
		return nil, err // 返回错误
	}
	return c.handleResponse(req) // 处理请求并返回响应
}

// PostJson 方法用于发送 POST JSON 请求
func (c *MsHttpClient) PostJson(url string, args map[string]any) ([]byte, error) {
	// 将参数编码为 JSON 字符串
	jsonStr, _ := json.Marshal(args)
	// 创建 POST 请求，内容为 JSON 数据
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonStr))
	if err != nil { // 如果创建请求时发生错误
		return nil, err // 返回错误
	}
	return c.handleResponse(req) // 处理请求并返回响应
}

// Do 方法用于根据服务名称和方法名称查找服务，并为该方法设置具体的请求处理函数
func (c *MsHttpClientSession) Do(service string, method string) MsService {
	msService, ok := c.serviceMap[service] // 从服务映射表中查找指定服务
	if !ok {                               // 如果服务不存在
		panic(errors.New("service not found")) // 抛出服务未找到的错误
	}

	// 获取服务的类型和值
	t := reflect.TypeOf(msService)   // 获取服务的类型
	v := reflect.ValueOf(msService)  // 获取服务的值
	if t.Kind() != reflect.Pointer { // 如果服务不是指针类型
		panic(errors.New("service not pointer")) // 抛出错误，服务必须是指针类型
	}
	tVar := t.Elem() // 获取指针指向的元素类型
	vVar := v.Elem() // 获取指针指向的元素值

	// 查找方法的字段索引
	fieldIndex := -1                       // 初始化字段索引为 -1
	for i := 0; i < tVar.NumField(); i++ { // 遍历服务的所有字段
		name := tVar.Field(i).Name // 获取字段名称
		if name == method {        // 如果字段名称与方法名称匹配
			fieldIndex = i // 设置字段索引
			break          // 跳出循环
		}
	}
	if fieldIndex == -1 { // 如果未找到字段
		panic(errors.New("method not found")) // 抛出方法未找到的错误
	}

	// 获取字段的标签信息
	tag := tVar.Field(fieldIndex).Tag // 获取字段的标签
	rpcInfo := tag.Get("msrpc")       // 获取 msrpc 标签的值
	if rpcInfo == "" {                // 如果标签为空
		panic(errors.New("not msrpc tag")) // 抛出错误，标签不存在
	}
	split := strings.Split(rpcInfo, ",") // 按逗号分割标签信息
	if len(split) != 2 {                 // 如果分割后的长度不为2
		panic(errors.New("tag msrpc not valid")) // 抛出错误，标签格式无效
	}
	methodType := split[0]        // 获取请求方法类型
	path := split[1]              // 获取请求路径
	httpConfig := msService.Env() // 获取服务的 HTTP 配置信息

	// 定义请求处理函数
	f := func(args map[string]any) ([]byte, error) {
		if methodType == GET { // 如果请求方法类型为 GET
			return c.Get(httpConfig.Prefix()+path, args) // 调用 Get 方法
		}
		if methodType == POSTForm { // 如果请求方法类型为 POST 表单
			return c.PostForm(httpConfig.Prefix()+path, args) // 调用 PostForm 方法
		}
		if methodType == POSTJson { // 如果请求方法类型为 POST JSON
			return c.PostJson(httpConfig.Prefix()+path, args) // 调用 PostJson 方法
		}
		return nil, errors.New("no match method type") // 如果没有匹配的方法类型，返回错误
	}
	fValue := reflect.ValueOf(f)       // 获取请求处理函数的值
	vVar.Field(fieldIndex).Set(fValue) // 为服务的方法字段设置请求处理函数
	return msService                   // 返回服务实例
}

// Prefix 方法用于生成带有协议、主机和端口的 URL 前缀
func (c HttpConfig) Prefix() string {
	if c.Protocol == "" { // 如果协议为空
		c.Protocol = HTTP // 将协议设置为 HTTP
	}
	switch c.Protocol { // 根据协议选择生成相应的 URL 前缀
	case HTTP: // 如果协议是 HTTP
		return fmt.Sprintf("http://%s:%d", c.Host, c.Port) // 返回 HTTP 协议的 URL 前缀
	case HTTPS: // 如果协议是 HTTPS
		return fmt.Sprintf("https://%s:%d", c.Host, c.Port) // 返回 HTTPS 协议的 URL 前缀
	}
	return "" // 如果协议不匹配，返回空字符串
}
