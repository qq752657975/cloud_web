package web

import (
	"encoding/base64"
	"net/http"
)

// Accounts 类型包含用户信息和未授权处理器
type Accounts struct {
	Users         map[string]string  // 存储用户名和密码的映射
	UnAuthHandler func(ctx *Context) // 未授权时的处理函数
}

// BasicAuth 中间件函数，进行基本身份验证
func (a *Accounts) BasicAuth(next HandlerFunc) HandlerFunc {
	return func(ctx *Context) {
		// 判断请求中是否有 Authorization 的 Header，并解析用户名和密码
		username, password, ok := ctx.R.BasicAuth()
		if !ok {
			// 如果没有提供 Authorization Header，调用未授权处理函数
			a.UnAuthHandlers(ctx)
			return
		}

		// 检查用户名是否存在
		pw, ok := a.Users[username]
		if !ok {
			// 如果用户名不存在，调用未授权处理函数
			a.UnAuthHandlers(ctx)
			return
		}

		// 检查密码是否正确
		if pw != password {
			// 如果密码不正确，调用未授权处理函数
			a.UnAuthHandlers(ctx)
			return
		}

		// 验证成功，设置上下文中的用户信息
		ctx.Set("user", username)
		// 调用下一个处理函数
		next(ctx)
	}
}

// UnAuthHandlers 处理未授权的请求
func (a *Accounts) UnAuthHandlers(ctx *Context) {
	if a.UnAuthHandler != nil {
		// 如果有自定义的未授权处理函数，则调用它
		a.UnAuthHandler(ctx)
	} else {
		// 否则返回 401 Unauthorized 状态码
		ctx.W.WriteHeader(http.StatusUnauthorized)
	}
}

// BasicAuth 返回一个基本身份验证的字符串（Base64 编码）
func basicAuth(username, password string) string {
	// 拼接用户名和密码，格式为 "username:password"
	auth := username + ":" + password
	// 将拼接后的字符串进行 Base64 编码
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

// SetBasicAuth 在请求头中设置基本身份验证信息
func (c *Context) SetBasicAuth(username, password string) {
	// 使用 BasicAuth 函数生成 Base64 编码的身份验证字符串
	authValue := "Basic " + basicAuth(username, password)

	// 在请求的 Header 中设置 Authorization 字段为生成的身份验证字符串
	c.R.Header.Set("Authorization", authValue)
}
