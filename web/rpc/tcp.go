package rpc

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ygb616/web/register"
	"golang.org/x/time/rate"
	"google.golang.org/protobuf/types/known/structpb"
	"io"
	"log"
	"net"
	"reflect"
	"sync/atomic"
	"time"

	"google.golang.org/protobuf/proto"
)

//TCP 客户端 服务端
//客户端 1. 连接服务端 2. 发送请求数据 （编码） 二进制 通过网络发送 3. 等待回复 接收到响应（解码）
//服务端 1. 启动服务 2. 接收请求 （解码），根据请求 调用对应的服务 得到响应数据 3. 将响应数据发送给客户端（编码）

// Serializer 接口定义了序列化和反序列化方法
type Serializer interface {
	Serialize(data any) ([]byte, error)        // 序列化方法
	DeSerialize(data []byte, target any) error // 反序列化方法
}

// GobSerializer 使用 gob 协议实现 Serializer 接口
type GobSerializer struct{}

// Serialize 方法将数据序列化为字节切片
func (c GobSerializer) Serialize(data any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)           // 创建 gob 编码器
	if err := encoder.Encode(data); err != nil { // 序列化数据
		return nil, err // 如果发生错误，返回错误信息
	}
	return buffer.Bytes(), nil // 返回序列化后的字节切片
}

// DeSerialize 方法将字节切片反序列化为数据
func (c GobSerializer) DeSerialize(data []byte, target any) error {
	buffer := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buffer) // 创建 gob 解码器
	return decoder.Decode(target)     // 反序列化数据并返回错误信息
}

// ProtobufSerializer 使用 protobuf 协议实现 Serializer 接口
type ProtobufSerializer struct{}

// Serialize 方法将数据序列化为字节切片
func (c ProtobufSerializer) Serialize(data any) ([]byte, error) {
	marshal, err := proto.Marshal(data.(proto.Message)) // 使用 protobuf 序列化数据
	if err != nil {                                     // 如果发生错误
		return nil, err // 返回错误信息
	}
	return marshal, nil // 返回序列化后的字节切片
}

// DeSerialize 方法将字节切片反序列化为数据
func (c ProtobufSerializer) DeSerialize(data []byte, target any) error {
	message := target.(proto.Message)     // 将目标转换为 protobuf 消息
	return proto.Unmarshal(data, message) // 使用 protobuf 反序列化数据并返回错误信息
}

// SerializerType 定义了序列化类型
type SerializerType byte

const (
	Gob       SerializerType = iota // Gob 序列化
	ProtoBuff                       // Protobuf 序列化
)

// CompressInterface 接口定义了压缩和解压缩方法
type CompressInterface interface {
	Compress([]byte) ([]byte, error)   // 压缩方法
	UnCompress([]byte) ([]byte, error) // 解压缩方法
}

// CompressType 定义了压缩类型
type CompressType byte

const (
	Gzip CompressType = iota // Gzip 压缩
)

// GzipCompress 实现了 CompressInterface 接口，使用 Gzip 进行压缩和解压缩
type GzipCompress struct{}

// Compress 方法将数据压缩为 Gzip 格式
func (c GzipCompress) Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf) // 创建 Gzip 写入器

	_, err := w.Write(data) // 将数据写入 Gzip 压缩器
	if err != nil {         // 如果写入过程中发生错误
		return nil, err // 返回错误
	}
	if err := w.Close(); err != nil { // 关闭 Gzip 写入器
		return nil, err // 如果关闭过程中发生错误，返回错误
	}
	return buf.Bytes(), nil // 返回压缩后的字节切片
}

// UnCompress 方法将 Gzip 格式的数据解压缩
func (c GzipCompress) UnCompress(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data)) // 创建 Gzip 读取器
	defer reader.Close()                                 // 确保在函数返回前关闭读取器
	if err != nil {                                      // 如果创建读取器时发生错误
		return nil, err // 返回错误
	}
	buf := new(bytes.Buffer)
	// 从读取器中读取数据
	if _, err := buf.ReadFrom(reader); err != nil { // 从 Gzip 读取器中读取解压缩后的数据
		return nil, err // 如果读取过程中发生错误，返回错误
	}
	return buf.Bytes(), nil // 返回解压缩后的字节切片
}

// 定义常量
const MagicNumber byte = 0x1d // 魔术数字，用于标识协议
const Version = 0x01          // 版本号

// 定义消息类型
type MessageType byte

const (
	msgRequest  MessageType = iota // 请求消息
	msgResponse                    // 响应消息
	msgPing                        // Ping 消息
	msgPong                        // Pong 消息
)

// 定义消息头结构体
type Header struct {
	MagicNumber   byte           // 魔术数字
	Version       byte           // 版本号
	FullLength    int32          // 消息总长度
	MessageType   MessageType    // 消息类型
	CompressType  CompressType   // 压缩类型
	SerializeType SerializerType // 序列化类型
	RequestId     int64          // 请求 ID
}

// 定义 RPC 消息结构体
type MsRpcMessage struct {
	Header *Header // 消息头
	Data   any     // 消息体
}

// 定义 RPC 请求结构体
type MsRpcRequest struct {
	RequestId   int64  // 请求 ID
	ServiceName string // 服务名称
	MethodName  string // 方法名称
	Args        []any  // 参数
}

// 定义 RPC 响应结构体
type MsRpcResponse struct {
	RequestId     int64          // 请求 ID
	Code          int16          // 响应代码
	Msg           string         // 响应消息
	CompressType  CompressType   // 压缩类型
	SerializeType SerializerType // 序列化类型
	Data          any            // 响应数据
}

// 定义 RPC 服务器接口
type MsRpcServer interface {
	Register(name string, service interface{}) // 注册服务
	Run()                                      // 运行服务器
	Stop()                                     // 停止服务器
}

// 定义 TCP 服务器结构体
type MsTcpServer struct {
	host           string              // 主机地址
	port           int                 // 端口号
	listen         net.Listener        // 网络监听器
	serviceMap     map[string]any      // 服务映射表
	RegisterType   string              // 注册类型
	RegisterOption register.Option     // 注册选项
	RegisterCli    register.MsRegister // 注册客户端
	LimiterTimeOut time.Duration       // 限流超时时间
	Limiter        *rate.Limiter       // 限流器
}

// NewTcpServer 函数创建新的 TCP 服务器
func NewTcpServer(host string, port int) (*MsTcpServer, error) {
	listen, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port)) // 创建 TCP 监听器
	if err != nil {                                                    // 如果监听器创建失败
		return nil, err // 返回错误
	}
	m := &MsTcpServer{serviceMap: make(map[string]any)} // 创建 MsTcpServer 实例
	m.listen = listen                                   // 赋值监听器
	m.port = port                                       // 赋值端口
	m.host = host                                       // 赋值主机
	return m, nil                                       // 返回 MsTcpServer 实例
}

// SetLimiter 方法设置限流器
func (s *MsTcpServer) SetLimiter(limit, cap int) {
	s.Limiter = rate.NewLimiter(rate.Limit(limit), cap) // 创建新的限流器
}

// Register 方法注册服务
func (s *MsTcpServer) Register(name string, service interface{}) {
	t := reflect.TypeOf(service)     // 获取服务的类型
	if t.Kind() != reflect.Pointer { // 如果服务不是指针类型
		panic("service must be pointer") // 抛出错误
	}
	s.serviceMap[name] = service // 将服务添加到服务映射表

	err := s.RegisterCli.CreateCli(s.RegisterOption) // 创建注册客户端
	if err != nil {                                  // 如果创建失败
		panic(err) // 抛出错误
	}
	err = s.RegisterCli.RegisterService(name, s.host, s.port) // 注册服务
	if err != nil {                                           // 如果注册失败
		panic(err) // 抛出错误
	}
}

// MsTcpConn 定义了 TCP 连接结构体
type MsTcpConn struct {
	conn    net.Conn            // 网络连接
	rspChan chan *MsRpcResponse // 响应通道
}

// Send 方法发送 RPC 响应
func (c MsTcpConn) Send(rsp *MsRpcResponse) error {
	if rsp.Code != 200 { // 如果响应代码不是 200
		// 进行默认的数据发送
	}
	// 编码并发送数据
	headers := make([]byte, 17)
	headers[0] = MagicNumber                                       // 魔术数字
	headers[1] = Version                                           // 版本号
	headers[6] = byte(msgResponse)                                 // 消息类型
	headers[7] = byte(rsp.CompressType)                            // 压缩类型
	headers[8] = byte(rsp.SerializeType)                           // 序列化类型
	binary.BigEndian.PutUint64(headers[9:], uint64(rsp.RequestId)) // 请求 ID

	// 编码：先序列化再压缩
	se := loadSerializer(rsp.SerializeType)
	var body []byte
	var err error
	if rsp.SerializeType == ProtoBuff { // 如果使用 ProtoBuff 序列化
		pRsp := &Response{}
		pRsp.SerializeType = int32(rsp.SerializeType)
		pRsp.CompressType = int32(rsp.CompressType)
		pRsp.Code = int32(rsp.Code)
		pRsp.Msg = rsp.Msg
		pRsp.RequestId = rsp.RequestId
		m := make(map[string]any)
		marshal, _ := json.Marshal(rsp.Data)
		_ = json.Unmarshal(marshal, &m)
		value, err := structpb.NewStruct(m)
		log.Println(err)
		pRsp.Data = structpb.NewStructValue(value)
		body, err = se.Serialize(pRsp)
	} else { // 否则使用默认序列化
		body, err = se.Serialize(rsp)
	}
	if err != nil {
		return err // 返回错误
	}
	com := loadCompress(rsp.CompressType) // 加载压缩器
	body, err = com.Compress(body)        // 压缩数据
	if err != nil {
		return err // 返回错误
	}
	fullLen := 17 + len(body)                                 // 计算消息总长度
	binary.BigEndian.PutUint32(headers[2:6], uint32(fullLen)) // 设置消息总长度

	_, err = c.conn.Write(headers[:]) // 发送消息头
	if err != nil {
		return err // 返回错误
	}
	_, err = c.conn.Write(body[:]) // 发送消息体
	if err != nil {
		return err // 返回错误
	}
	return nil // 返回 nil 表示成功
}

// Stop 方法用于停止 TCP 服务器
func (s *MsTcpServer) Stop() {
	err := s.listen.Close() // 关闭监听器
	if err != nil {         // 如果关闭监听器时发生错误
		log.Println(err) // 打印错误日志
	}
}

// Run 方法用于运行 TCP 服务器
func (s *MsTcpServer) Run() {
	for {
		conn, err := s.listen.Accept() // 接受新的连接
		if err != nil {                // 如果接受连接时发生错误
			log.Println(err) // 打印错误日志
			continue         // 继续接受下一个连接
		}
		msConn := &MsTcpConn{conn: conn, rspChan: make(chan *MsRpcResponse, 1)} // 创建新的 MsTcpConn 实例
		// 1. 一直接收数据 解码工作 请求业务获取结果 发送到rspChan
		// 2. 获得结果 编码 发送数据
		go s.readHandle(msConn)  // 启动协程处理读取操作
		go s.writeHandle(msConn) // 启动协程处理写入操作
	}
}

// readHandle 方法用于处理读取操作
func (s *MsTcpServer) readHandle(conn *MsTcpConn) {
	defer func() {
		if err := recover(); err != nil {
			log.Println("readHandle recover ", err) // 打印恢复的错误日志
			conn.conn.Close()                       // 关闭连接
		}
	}()
	// 在这加一个限流
	ctx, cancel := context.WithTimeout(context.Background(), s.LimiterTimeOut) // 创建带超时的上下文
	defer cancel()                                                             // 确保在函数返回前取消上下文
	err2 := s.Limiter.WaitN(ctx, 1)                                            // 等待限流
	if err2 != nil {                                                           // 如果限流发生错误
		rsp := &MsRpcResponse{} // 创建新的 RPC 响应
		rsp.Code = 700          // 被限流的错误代码
		rsp.Msg = err2.Error()  // 错误信息
		conn.rspChan <- rsp     // 发送响应到响应通道
		return
	}
	// 接收数据
	// 解码
	msg, err := decodeFrame(conn.conn) // 解码消息
	if err != nil {                    // 如果解码时发生错误
		rsp := &MsRpcResponse{} // 创建新的 RPC 响应
		rsp.Code = 500          // 错误代码
		rsp.Msg = err.Error()   // 错误信息
		conn.rspChan <- rsp     // 发送响应到响应通道
		return
	}
	if msg.Header.MessageType == msgRequest { // 如果消息类型是请求
		if msg.Header.SerializeType == ProtoBuff { // 如果序列化类型是 ProtoBuff
			req := msg.Data.(*Request) // 将消息体转换为请求
			rsp := &MsRpcResponse{RequestId: req.RequestId}
			rsp.SerializeType = msg.Header.SerializeType
			rsp.CompressType = msg.Header.CompressType
			serviceName := req.ServiceName
			service, ok := s.serviceMap[serviceName]
			if !ok { // 如果找不到服务
				rsp := &MsRpcResponse{}                          // 创建新的 RPC 响应
				rsp.Code = 500                                   // 错误代码
				rsp.Msg = errors.New("no service found").Error() // 错误信息
				conn.rspChan <- rsp                              // 发送响应到响应通道
				return
			}
			methodName := req.MethodName
			method := reflect.ValueOf(service).MethodByName(methodName) // 获取服务的方法
			if method.IsNil() {                                         // 如果找不到方法
				rsp := &MsRpcResponse{}                                 // 创建新的 RPC 响应
				rsp.Code = 500                                          // 错误代码
				rsp.Msg = errors.New("no service method found").Error() // 错误信息
				conn.rspChan <- rsp                                     // 发送响应到响应通道
				return
			}
			// 调用方法
			args := make([]reflect.Value, len(req.Args))
			for i := range req.Args { // 将请求参数转换为 reflect.Value
				of := reflect.ValueOf(req.Args[i].AsInterface())
				of = of.Convert(method.Type().In(i))
				args[i] = of
			}
			result := method.Call(args) // 调用方法并获取结果

			results := make([]any, len(result))
			for i, v := range result { // 将结果转换为接口
				results[i] = v.Interface()
			}
			err, ok := results[len(result)-1].(error) // 检查最后一个返回值是否是错误
			if ok {                                   // 如果是错误
				rsp.Code = 500        // 错误代码
				rsp.Msg = err.Error() // 错误信息
				conn.rspChan <- rsp   // 发送响应到响应通道
				return
			}
			rsp.Code = 200        // 成功代码
			rsp.Data = results[0] // 设置响应数据
			conn.rspChan <- rsp   // 发送响应到响应通道
		} else { // 否则使用默认序列化
			req := msg.Data.(*MsRpcRequest) // 将消息体转换为 RPC 请求
			rsp := &MsRpcResponse{RequestId: req.RequestId}
			rsp.SerializeType = msg.Header.SerializeType
			rsp.CompressType = msg.Header.CompressType
			serviceName := req.ServiceName
			service, ok := s.serviceMap[serviceName]
			if !ok { // 如果找不到服务
				rsp := &MsRpcResponse{}                          // 创建新的 RPC 响应
				rsp.Code = 500                                   // 错误代码
				rsp.Msg = errors.New("no service found").Error() // 错误信息
				conn.rspChan <- rsp                              // 发送响应到响应通道
				return
			}
			methodName := req.MethodName
			method := reflect.ValueOf(service).MethodByName(methodName) // 获取服务的方法
			if method.IsNil() {                                         // 如果找不到方法
				rsp := &MsRpcResponse{}                                 // 创建新的 RPC 响应
				rsp.Code = 500                                          // 错误代码
				rsp.Msg = errors.New("no service method found").Error() // 错误信息
				conn.rspChan <- rsp                                     // 发送响应到响应通道
				return
			}
			// 调用方法
			args := req.Args
			var valuesArg []reflect.Value
			for _, v := range args { // 将请求参数转换为 reflect.Value
				valuesArg = append(valuesArg, reflect.ValueOf(v))
			}
			result := method.Call(valuesArg) // 调用方法并获取结果

			results := make([]any, len(result))
			for I, v := range result { // 将结果转换为接口
				results[I] = v.Interface()
			}
			err, ok := results[len(result)-1].(error) // 检查最后一个返回值是否是错误
			if ok {                                   // 如果是错误
				rsp.Code = 500        // 错误代码
				rsp.Msg = err.Error() // 错误信息
				conn.rspChan <- rsp   // 发送响应到响应通道
				return
			}
			rsp.Code = 200        // 成功代码
			rsp.Data = results[0] // 设置响应数据
			conn.rspChan <- rsp   // 发送响应到响应通道
		}
	}
}

// writeHandle 方法用于处理写入操作
func (s *MsTcpServer) writeHandle(conn *MsTcpConn) {
	select {
	case rsp := <-conn.rspChan: // 从响应通道接收响应
		defer conn.conn.Close() // 确保连接关闭
		// 发送数据
		err := conn.Send(rsp) // 发送响应
		if err != nil {
			log.Println(err) // 打印错误日志
		}
	}
}

// SetRegister 方法设置注册类型和选项
func (s *MsTcpServer) SetRegister(registerType string, option register.Option) {
	s.RegisterType = registerType // 设置注册类型
	s.RegisterOption = option     // 设置注册选项
	if registerType == "nacos" {  // 如果注册类型是 nacos
		s.RegisterCli = &register.MsNacosRegister{} // 设置注册客户端为 MsNacosRegister
	}
	if registerType == "etcd" { // 如果注册类型是 etcd
		s.RegisterCli = &register.MsEtcdRegister{} // 设置注册客户端为 MsEtcdRegister
	}
}

// decodeFrame 函数解码消息帧
func decodeFrame(conn net.Conn) (*MsRpcMessage, error) {
	// 1+1+4+1+1+1+8 = 17 字节
	headers := make([]byte, 17)          // 创建消息头缓冲区
	_, err := io.ReadFull(conn, headers) // 读取消息头
	if err != nil {                      // 如果读取消息头时发生错误
		return nil, err // 返回错误
	}
	mn := headers[0]       // 获取魔术数字
	if mn != MagicNumber { // 如果魔术数字不匹配
		return nil, errors.New("magic number error") // 返回错误
	}
	vs := headers[1] // 获取版本号
	// 解析消息头中的其他字段
	fullLength := int32(binary.BigEndian.Uint32(headers[2:6])) // 获取消息总长度
	messageType := headers[6]                                  // 获取消息类型
	compressType := headers[7]                                 // 获取压缩类型
	seType := headers[8]                                       // 获取序列化类型
	requestId := int64(binary.BigEndian.Uint32(headers[9:]))   // 获取请求 ID

	// 创建消息
	msg := &MsRpcMessage{
		Header: &Header{},
	}
	msg.Header.MagicNumber = mn                          // 设置魔术数字
	msg.Header.Version = vs                              // 设置版本号
	msg.Header.FullLength = fullLength                   // 设置消息总长度
	msg.Header.MessageType = MessageType(messageType)    // 设置消息类型
	msg.Header.CompressType = CompressType(compressType) // 设置压缩类型
	msg.Header.SerializeType = SerializerType(seType)    // 设置序列化类型
	msg.Header.RequestId = requestId                     // 设置请求 ID

	// 读取消息体
	bodyLen := fullLength - 17       // 计算消息体长度
	body := make([]byte, bodyLen)    // 创建消息体缓冲区
	_, err = io.ReadFull(conn, body) // 读取消息体
	if err != nil {                  // 如果读取消息体时发生错误
		return nil, err // 返回错误
	}
	// 解码：先解压缩，再反序列化
	compress := loadCompress(CompressType(compressType)) // 加载压缩器
	if compress == nil {                                 // 如果压缩器不存在
		return nil, errors.New("no compress") // 返回错误
	}
	body, err = compress.UnCompress(body) // 解压缩消息体
	if err != nil {                       // 如果解压缩时发生错误
		return nil, err // 返回错误
	}
	serializer := loadSerializer(SerializerType(seType)) // 加载序列化器
	if serializer == nil {                               // 如果序列化器不存在
		return nil, errors.New("no serializer") // 返回错误
	}
	// 处理不同类型的消息
	if MessageType(messageType) == msgRequest { // 如果消息类型是请求
		if SerializerType(seType) == ProtoBuff { // 如果序列化类型是 ProtoBuff
			req := &Request{}                        // 创建请求
			err := serializer.DeSerialize(body, req) // 反序列化请求
			if err != nil {                          // 如果反序列化时发生错误
				return nil, err // 返回错误
			}
			msg.Data = req // 设置消息数据
		} else { // 否则
			req := &MsRpcRequest{}                   // 创建 RPC 请求
			err := serializer.DeSerialize(body, req) // 反序列化 RPC 请求
			if err != nil {                          // 如果反序列化时发生错误
				return nil, err // 返回错误
			}
			msg.Data = req // 设置消息数据
		}
		return msg, nil // 返回消息
	}
	if MessageType(messageType) == msgResponse { // 如果消息类型是响应
		if SerializerType(seType) == ProtoBuff { // 如果序列化类型是 ProtoBuff
			rsp := &Response{}                       // 创建响应
			err := serializer.DeSerialize(body, rsp) // 反序列化响应
			if err != nil {                          // 如果反序列化时发生错误
				return nil, err // 返回错误
			}
			msg.Data = rsp // 设置消息数据
		} else { // 否则
			rsp := &MsRpcResponse{}                  // 创建 RPC 响应
			err := serializer.DeSerialize(body, rsp) // 反序列化 RPC 响应
			if err != nil {                          // 如果反序列化时发生错误
				return nil, err // 返回错误
			}
			msg.Data = rsp // 设置消息数据
		}
		return msg, nil // 返回消息
	}
	return nil, errors.New("no message type") // 返回错误：未知消息类型
}

// loadSerializer 函数加载序列化器
func loadSerializer(serializerType SerializerType) Serializer {
	switch serializerType {
	case Gob: // 如果序列化类型是 Gob
		return GobSerializer{} // 返回 Gob 序列化器
	case ProtoBuff: // 如果序列化类型是 ProtoBuff
		return ProtobufSerializer{} // 返回 ProtoBuff 序列化器
	}
	return nil // 如果没有匹配的序列化器，返回 nil
}

// loadCompress 函数加载压缩器
func loadCompress(compressType CompressType) CompressInterface {
	switch compressType {
	case Gzip: // 如果压缩类型是 Gzip
		return GzipCompress{} // 返回 Gzip 压缩器
	}
	return nil // 如果没有匹配的压缩器，返回 nil
}

// MsRpcClient 接口定义了 RPC 客户端的基本操作
type MsRpcClient interface {
	Connect() error                                                                                 // 连接到 RPC 服务器
	Invoke(context context.Context, serviceName string, methodName string, args []any) (any, error) // 调用远程方法
	Close() error                                                                                   // 关闭连接
}

// MsTcpClient 结构体定义了 TCP 客户端
type MsTcpClient struct {
	conn        net.Conn            // 网络连接
	option      TcpClientOption     // 客户端选项
	ServiceName string              // 服务名称
	RegisterCli register.MsRegister // 注册客户端
}

// TcpClientOption 结构体定义了 TCP 客户端的选项
type TcpClientOption struct {
	Retries           int                 // 重试次数
	ConnectionTimeout time.Duration       // 连接超时时间
	SerializeType     SerializerType      // 序列化类型
	CompressType      CompressType        // 压缩类型
	Host              string              // 主机地址
	Port              int                 // 端口号
	RegisterType      string              // 注册类型
	RegisterOption    register.Option     // 注册选项
	RegisterCli       register.MsRegister // 注册客户端
}

// DefaultOption 定义了默认的 TCP 客户端选项
var DefaultOption = TcpClientOption{
	Host:              "127.0.0.1",     // 默认主机地址
	Port:              9222,            // 默认端口号
	Retries:           3,               // 默认重试次数
	ConnectionTimeout: 5 * time.Second, // 默认连接超时时间
	SerializeType:     Gob,             // 默认序列化类型
	CompressType:      Gzip,            // 默认压缩类型
}

// NewTcpClient 函数创建新的 TCP 客户端
func NewTcpClient(option TcpClientOption) *MsTcpClient {
	return &MsTcpClient{option: option} // 返回新的 MsTcpClient 实例
}

// Connect 方法用于连接到 RPC 服务器
func (c *MsTcpClient) Connect() error {
	var addr string
	err := c.RegisterCli.CreateCli(c.option.RegisterOption) // 创建注册客户端
	if err != nil {                                         // 如果创建注册客户端时发生错误
		panic(err) // 抛出错误
	}
	addr, err = c.RegisterCli.GetValue(c.ServiceName) // 获取服务地址
	if err != nil {                                   // 如果获取服务地址时发生错误
		panic(err) // 抛出错误
	}
	conn, err := net.DialTimeout("tcp", addr, c.option.ConnectionTimeout) // 连接到 RPC 服务器
	if err != nil {                                                       // 如果连接时发生错误
		return err // 返回错误
	}
	c.conn = conn // 设置网络连接
	return nil    // 返回 nil 表示成功
}

// Close 方法用于关闭连接
func (c *MsTcpClient) Close() error {
	if c.conn != nil { // 如果网络连接存在
		return c.conn.Close() // 关闭连接
	}
	return nil // 返回 nil 表示成功
}

// 全局请求ID变量
var reqId int64

// Invoke 方法用于调用远程服务
func (c *MsTcpClient) Invoke(ctx context.Context, serviceName string, methodName string, args []any) (any, error) {
	// 包装 request 对象，编码并发送
	req := &MsRpcRequest{}
	req.RequestId = atomic.AddInt64(&reqId, 1) // 生成请求 ID
	req.ServiceName = serviceName              // 设置服务名称
	req.MethodName = methodName                // 设置方法名称
	req.Args = args                            // 设置参数

	headers := make([]byte, 17)                                    // 创建消息头缓冲区
	headers[0] = MagicNumber                                       // 设置魔术数字
	headers[1] = Version                                           // 设置版本号
	headers[6] = byte(msgRequest)                                  // 设置消息类型
	headers[7] = byte(c.option.CompressType)                       // 设置压缩类型
	headers[8] = byte(c.option.SerializeType)                      // 设置序列化类型
	binary.BigEndian.PutUint64(headers[9:], uint64(req.RequestId)) // 设置请求 ID

	serializer := loadSerializer(c.option.SerializeType) // 加载序列化器
	if serializer == nil {                               // 如果序列化器不存在
		return nil, errors.New("no serializer") // 返回错误
	}

	var body []byte
	var err error
	if c.option.SerializeType == ProtoBuff { // 如果序列化类型是 ProtoBuff
		pReq := &Request{}
		pReq.RequestId = atomic.AddInt64(&reqId, 1) // 生成请求 ID
		pReq.ServiceName = serviceName              // 设置服务名称
		pReq.MethodName = methodName                // 设置方法名称
		listValue, err := structpb.NewList(args)    // 将参数转换为 structpb.List
		if err != nil {                             // 如果转换时发生错误
			return nil, err // 返回错误
		}
		pReq.Args = listValue.Values           // 设置参数
		body, err = serializer.Serialize(pReq) // 序列化请求
	} else { // 否则
		body, err = serializer.Serialize(req) // 序列化请求
	}

	if err != nil { // 如果序列化时发生错误
		return nil, err // 返回错误
	}

	compress := loadCompress(c.option.CompressType) // 加载压缩器
	if compress == nil {                            // 如果压缩器不存在
		return nil, errors.New("no compress") // 返回错误
	}
	body, err = compress.Compress(body) // 压缩消息体
	if err != nil {                     // 如果压缩时发生错误
		return nil, err // 返回错误
	}

	fullLen := 17 + len(body)                                 // 计算消息总长度
	binary.BigEndian.PutUint32(headers[2:6], uint32(fullLen)) // 设置消息总长度

	_, err = c.conn.Write(headers[:]) // 发送消息头
	if err != nil {                   // 如果发送时发生错误
		return nil, err // 返回错误
	}

	_, err = c.conn.Write(body[:]) // 发送消息体
	if err != nil {                // 如果发送时发生错误
		return nil, err // 返回错误
	}

	rspChan := make(chan *MsRpcResponse) // 创建响应通道
	go c.readHandle(rspChan)             // 启动协程读取响应
	rsp := <-rspChan                     // 从通道接收响应
	return rsp, nil                      // 返回响应
}

// readHandle 方法用于读取响应
func (c *MsTcpClient) readHandle(rspChan chan *MsRpcResponse) {
	defer func() {
		if err := recover(); err != nil {
			log.Println("MsTcpClient readHandle recover: ", err) // 打印恢复的错误日志
			c.conn.Close()                                       // 关闭连接
		}
	}()

	for {
		msg, err := decodeFrame(c.conn) // 解码消息
		if err != nil {
			log.Println("未解析出任何数据") // 打印错误日志
			rsp := &MsRpcResponse{}
			rsp.Code = 500        // 错误代码
			rsp.Msg = err.Error() // 错误信息
			rspChan <- rsp        // 发送响应到通道
			return
		}

		if msg.Header.MessageType == msgResponse { // 如果消息类型是响应
			if msg.Header.SerializeType == ProtoBuff { // 如果序列化类型是 ProtoBuff
				rsp := msg.Data.(*Response)             // 反序列化响应
				asInterface := rsp.Data.AsInterface()   // 获取响应数据
				marshal, _ := json.Marshal(asInterface) // 序列化响应数据为 JSON
				rsp1 := &MsRpcResponse{}
				json.Unmarshal(marshal, rsp1) // 反序列化 JSON 为 RPC 响应
				rspChan <- rsp1               // 发送响应到通道
			} else {
				rsp := msg.Data.(*MsRpcResponse) // 反序列化 RPC 响应
				rspChan <- rsp                   // 发送响应到通道
			}
			return
		}
	}
}

// decodeFrame 方法用于解码消息帧
func (c *MsTcpClient) decodeFrame(conn net.Conn) (*MsRpcMessage, error) {
	// 1+1+4+1+1+1+8 = 17 字节
	headers := make([]byte, 17)          // 创建消息头缓冲区
	_, err := io.ReadFull(conn, headers) // 读取消息头
	if err != nil {                      // 如果读取消息头时发生错误
		return nil, err // 返回错误
	}
	mn := headers[0]       // 获取魔术数字
	if mn != MagicNumber { // 如果魔术数字不匹配
		return nil, errors.New("magic number error") // 返回错误
	}
	vs := headers[1] // 获取版本号
	// 解析消息头中的其他字段
	fullLength := int32(binary.BigEndian.Uint32(headers[2:6])) // 获取消息总长度
	messageType := headers[6]                                  // 获取消息类型
	compressType := headers[7]                                 // 获取压缩类型
	seType := headers[8]                                       // 获取序列化类型
	requestId := int64(binary.BigEndian.Uint32(headers[9:]))   // 获取请求 ID

	// 创建消息
	msg := &MsRpcMessage{
		Header: &Header{},
	}
	msg.Header.MagicNumber = mn                          // 设置魔术数字
	msg.Header.Version = vs                              // 设置版本号
	msg.Header.FullLength = fullLength                   // 设置消息总长度
	msg.Header.MessageType = MessageType(messageType)    // 设置消息类型
	msg.Header.CompressType = CompressType(compressType) // 设置压缩类型
	msg.Header.SerializeType = SerializerType(seType)    // 设置序列化类型
	msg.Header.RequestId = requestId                     // 设置请求 ID

	// 读取消息体
	bodyLen := fullLength - 17       // 计算消息体长度
	body := make([]byte, bodyLen)    // 创建消息体缓冲区
	_, err = io.ReadFull(conn, body) // 读取消息体
	if err != nil {                  // 如果读取消息体时发生错误
		return nil, err // 返回错误
	}
	// 解码：先解压缩，再反序列化
	compress := loadCompress(CompressType(compressType)) // 加载压缩器
	if compress == nil {                                 // 如果压缩器不存在
		return nil, errors.New("no compress") // 返回错误
	}
	body, err = compress.UnCompress(body) // 解压缩消息体
	if compress == nil {                  // 如果解压缩时发生错误
		return nil, err // 返回错误
	}
	serializer := loadSerializer(SerializerType(seType)) // 加载序列化器
	if serializer == nil {                               // 如果序列化器不存在
		return nil, errors.New("no serializer") // 返回错误
	}
	if MessageType(messageType) == msgRequest { // 如果消息类型是请求
		req := &MsRpcRequest{}                   // 创建请求对象
		err := serializer.DeSerialize(body, req) // 反序列化请求
		if err != nil {                          // 如果反序列化时发生错误
			return nil, err // 返回错误
		}
		msg.Data = req  // 设置消息数据
		return msg, nil // 返回消息
	}
	if MessageType(messageType) == msgResponse { // 如果消息类型是响应
		rsp := &MsRpcResponse{}                  // 创建响应对象
		err := serializer.DeSerialize(body, rsp) // 反序列化响应
		if err != nil {                          // 如果反序列化时发生错误
			return nil, err // 返回错误
		}
		msg.Data = rsp  // 设置消息数据
		return msg, nil // 返回消息
	}
	return nil, errors.New("no message type") // 返回错误：未知消息类型
}

// MsTcpClientProxy 结构体定义了 TCP 客户端代理
type MsTcpClientProxy struct {
	client *MsTcpClient    // TCP 客户端
	option TcpClientOption // 客户端选项
}

// NewMsTcpClientProxy 函数创建新的 MsTcpClientProxy 实例
func NewMsTcpClientProxy(option TcpClientOption) *MsTcpClientProxy {
	return &MsTcpClientProxy{option: option} // 返回新的 MsTcpClientProxy 实例
}

// Call 方法用于调用远程服务
func (p *MsTcpClientProxy) Call(ctx context.Context, serviceName string, methodName string, args []any) (any, error) {
	client := NewTcpClient(p.option)      // 创建新的 TCP 客户端
	client.ServiceName = serviceName      // 设置服务名称
	if p.option.RegisterType == "nacos" { // 如果注册类型是 nacos
		client.RegisterCli = &register.MsNacosRegister{} // 设置注册客户端为 MsNacosRegister
	}
	if p.option.RegisterType == "etcd" { // 如果注册类型是 etcd
		client.RegisterCli = &register.MsEtcdRegister{} // 设置注册客户端为 MsEtcdRegister
	}
	p.client = client       // 设置代理的客户端
	err := client.Connect() // 连接到服务
	if err != nil {         // 如果连接时发生错误
		return nil, err // 返回错误
	}
	for i := 0; i < p.option.Retries; i++ { // 重试指定次数
		result, err := client.Invoke(ctx, serviceName, methodName, args) // 调用远程方法
		if err != nil {                                                  // 如果调用时发生错误
			if i >= p.option.Retries-1 { // 如果已达到最大重试次数
				log.Println(errors.New("already retry all time")) // 打印重试结束的错误日志
				client.Close()                                    // 关闭客户端连接
				return nil, err                                   // 返回错误
			}
			// 睡眠一小会（可以在此添加实际的睡眠代码，例如 time.Sleep）
			continue // 继续重试
		}
		client.Close()     // 关闭客户端连接
		return result, nil // 返回结果
	}
	return nil, errors.New("retry time is 0") // 如果重试次数为0，返回错误
}
