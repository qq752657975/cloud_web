package register

import (
	"context"
	"errors"
	"fmt"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

// CreateEtcdCli 创建并返回一个etcd客户端
func CreateEtcdCli(option Option) (*clientv3.Client, error) {
	// 使用传入的选项创建一个etcd客户端
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   option.Endpoints,   // etcd节点列表
		DialTimeout: option.DialTimeout, // 连接超时时间
	})
	return cli, err // 返回客户端和错误（如果有）
}

// RegEtcdService 在etcd中注册服务
func RegEtcdService(cli *clientv3.Client, serviceName string, host string, port int) error {
	// 创建上下文，设置超时时间为1秒
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel() // 确保函数返回前取消上下文
	// 在etcd中注册服务，键为服务名称，值为服务地址和端口
	_, err := cli.Put(ctx, serviceName, fmt.Sprintf("%s:%d", host, port))
	return err // 返回注册服务时的错误（如果有）
}

// GetEtcdValue 从etcd中获取服务的值
func GetEtcdValue(cli *clientv3.Client, serviceName string) (string, error) {
	// 创建上下文，设置超时时间为1秒
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel() // 确保函数返回前取消上下文
	// 从etcd中获取服务的值
	v, err := cli.Get(ctx, serviceName)
	if err != nil {
		return "", err // 如果获取值失败，返回错误
	}
	// 获取键值对列表
	kvs := v.Kvs
	if len(kvs) == 0 {
		return "", errors.New("no value") // 如果没有值，返回错误
	}
	return string(kvs[0].Value), err // 返回第一个键值对的值和错误（如果有）
}

// MsEtcdRegister 代表一个etcd注册器
type MsEtcdRegister struct {
	cli *clientv3.Client // etcd客户端
}

// CreateCli 创建etcd客户端
func (r *MsEtcdRegister) CreateCli(option Option) error {
	// 创建etcd客户端
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   option.Endpoints,   // etcd节点列表
		DialTimeout: option.DialTimeout, // 连接超时时间
	})
	r.cli = cli // 将创建的客户端赋值给结构体的cli字段
	return err  // 返回可能的错误
}

// RegisterService 在etcd中注册服务
func (r *MsEtcdRegister) RegisterService(serviceName string, host string, port int) error {
	// 创建一个上下文，设置超时时间为1秒
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel() // 确保函数返回前取消上下文
	// 在etcd中注册服务，键为服务名称，值为服务地址和端口
	_, err := r.cli.Put(ctx, serviceName, fmt.Sprintf("%s:%d", host, port))
	return err // 返回注册服务时的错误（如果有）
}

// GetValue 从etcd中获取服务的值
func (r *MsEtcdRegister) GetValue(serviceName string) (string, error) {
	// 创建一个上下文，设置超时时间为1秒
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel() // 确保函数返回前取消上下文
	// 从etcd中获取服务的值
	v, err := r.cli.Get(ctx, serviceName)
	if err != nil {
		return "", err // 如果获取值失败，返回错误
	}
	// 获取键值对列表
	kvs := v.Kvs
	if len(kvs) == 0 {
		return "", errors.New("no value") // 如果没有值，返回错误
	}
	return string(kvs[0].Value), err // 返回第一个键值对的值和错误（如果有）
}

// Close 关闭etcd客户端
func (r *MsEtcdRegister) Close() error {
	return r.cli.Close() // 关闭etcd客户端
}
