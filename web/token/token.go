package token

import (
	"errors"
	"github.com/golang-jwt/jwt/v4"
	"github.com/ygb616/web"
	"net/http"
	"time"
)

const JWTToken = "web_token"

type JwtHandler struct {
	//jwt的算法
	Alg string
	//过期时间
	TimeOut time.Duration
	//
	RefreshTimeOut time.Duration
	//时间函数
	TimeFuc func() time.Time
	//Key
	Key []byte
	//刷新key
	RefreshKey string
	//私钥
	PrivateKey string
	//
	SendCookie    bool
	Authenticator func(ctx *web.Context) (map[string]any, error)

	CookieName     string
	CookieMaxAge   int64
	CookieDomain   string
	SecureCookie   bool
	CookieHTTPOnly bool
	Header         string
	AuthHandler    func(ctx *web.Context, err error)
}

// JwtResponse 结构体用于存储 JWT 和刷新令牌
type JwtResponse struct {
	Token        string // 主 JWT
	RefreshToken string // 刷新令牌
}

// LoginHandler 方法用于用户登录认证，并生成 JWT 和刷新令牌
func (j *JwtHandler) LoginHandler(ctx *web.Context) (*JwtResponse, error) {
	// 调用认证函数进行用户认证
	data, err := j.Authenticator(ctx)
	if err != nil {
		return nil, err // 如果认证失败，返回 nil 和错误信息
	}

	// 如果没有指定算法，默认使用 HS256
	if j.Alg == "" {
		j.Alg = "HS256"
	}

	// 获取签名方法并创建一个新的 JWT token
	signingMethod := jwt.GetSigningMethod(j.Alg)
	token := jwt.New(signingMethod)

	// 获取 token 的声明（claims），并将认证数据加入到 claims 中
	claims := token.Claims.(jwt.MapClaims)
	if data != nil {
		for key, value := range data {
			claims[key] = value // 将认证数据加入到 claims 中
		}
	}

	// 如果没有指定时间函数，默认使用当前时间
	if j.TimeFuc == nil {
		j.TimeFuc = func() time.Time {
			return time.Now()
		}
	}

	// 计算 token 的过期时间
	expire := j.TimeFuc().Add(j.TimeOut)
	claims["exp"] = expire.Unix()      // 设置过期时间（exp）
	claims["iat"] = j.TimeFuc().Unix() // 设置签发时间（iat）

	// 根据算法选择使用公钥或密钥进行签名，并生成 token 字符串
	var tokenString string
	var tokenErr error
	if j.usingPublicKeyAlgo() {
		tokenString, tokenErr = token.SignedString(j.PrivateKey) // 使用私钥进行签名
	} else {
		tokenString, tokenErr = token.SignedString(j.Key) // 使用密钥进行签名
	}
	if tokenErr != nil {
		return nil, tokenErr // 如果签名失败，返回 nil 和错误信息
	}

	// 创建 JwtResponse 结构体实例
	jr := &JwtResponse{
		Token: tokenString, // 设置主 JWT
	}

	// 生成刷新令牌
	refreshToken, err := j.refreshToken(token)
	if err != nil {
		return nil, err // 如果生成刷新令牌失败，返回 nil 和错误信息
	}
	jr.RefreshToken = refreshToken // 设置刷新令牌

	// 如果配置了发送 Cookie 的选项，将 token 设置到 Cookie 中
	if j.SendCookie {
		if j.CookieName == "" {
			j.CookieName = JWTToken // 如果未指定 Cookie 名称，使用默认值
		}
		if j.CookieMaxAge == 0 {
			j.CookieMaxAge = expire.Unix() - j.TimeFuc().Unix() // 设置 Cookie 的最大存活时间
		}
		// 设置 Cookie
		ctx.SetCookie(j.CookieName, tokenString, int(j.CookieMaxAge), "/", j.CookieDomain, j.SecureCookie, j.CookieHTTPOnly)
	}

	return jr, nil // 返回生成的 JwtResponse 结构体实例
}

// 判断是否使用公钥算法
func (j *JwtHandler) usingPublicKeyAlgo() bool {
	switch j.Alg {
	case "RS256", "RS512", "RS384":
		return true // 使用公钥算法
	}
	return false // 不使用公钥算法
}

// refreshToken 方法用于生成新的刷新令牌
func (j *JwtHandler) refreshToken(token *jwt.Token) (string, error) {
	// 获取 token 的声明（claims）
	claims := token.Claims.(jwt.MapClaims)
	// 设置新的过期时间为当前时间加上刷新过期时间
	claims["exp"] = j.TimeFuc().Add(j.RefreshTimeOut).Unix()

	// 根据算法选择使用公钥或密钥进行签名，并生成 token 字符串
	var tokenString string
	var tokenErr error
	if j.usingPublicKeyAlgo() {
		tokenString, tokenErr = token.SignedString(j.PrivateKey) // 使用私钥进行签名
	} else {
		tokenString, tokenErr = token.SignedString(j.Key) // 使用密钥进行签名
	}
	if tokenErr != nil {
		return "", tokenErr // 如果签名失败，返回空字符串和错误信息
	}
	return tokenString, nil // 返回生成的刷新令牌
}

// LogoutHandler 退出登录
func (j *JwtHandler) LogoutHandler(ctx *web.Context) error {
	// 如果配置了发送 Cookie 的选项
	if j.SendCookie {
		if j.CookieName == "" {
			j.CookieName = JWTToken // 如果未指定 Cookie 名称，使用默认值
		}
		// 设置 Cookie，值为空，过期时间为负数表示删除该 Cookie
		ctx.SetCookie(j.CookieName, "", -1, "/", j.CookieDomain, j.SecureCookie, j.CookieHTTPOnly)
		return nil // 返回 nil 表示成功
	}
	return nil // 返回 nil 表示成功
}

// RefreshHandler 刷新 token
func (j *JwtHandler) RefreshHandler(ctx *web.Context) (*JwtResponse, error) {
	// 获取刷新令牌
	rToken, ok := ctx.Get(j.RefreshKey)
	if !ok {
		return nil, errors.New("refresh token is null") // 如果没有刷新令牌，返回错误
	}
	// 如果没有指定算法，默认使用 HS256
	if j.Alg == "" {
		j.Alg = "HS256"
	}
	// 解析 token
	t, err := jwt.Parse(rToken.(string), func(token *jwt.Token) (interface{}, error) {
		if j.usingPublicKeyAlgo() {
			return j.PrivateKey, nil // 使用私钥进行验证
		} else {
			return j.Key, nil // 使用密钥进行验证
		}
	})
	if err != nil {
		return nil, err // 如果解析失败，返回错误
	}
	// 获取 token 的声明（claims）
	claims := t.Claims.(jwt.MapClaims)

	// 如果没有指定时间函数，默认使用当前时间
	if j.TimeFuc == nil {
		j.TimeFuc = func() time.Time {
			return time.Now()
		}
	}
	// 计算新的过期时间并设置声明中的 "exp" 和 "iat"
	expire := j.TimeFuc().Add(j.TimeOut)
	claims["exp"] = expire.Unix()      // 设置过期时间（exp）
	claims["iat"] = j.TimeFuc().Unix() // 设置签发时间（iat）

	// 根据算法选择使用公钥或密钥进行签名，并生成新的 token 字符串
	var tokenString string
	var tokenErr error
	if j.usingPublicKeyAlgo() {
		tokenString, tokenErr = t.SignedString(j.PrivateKey) // 使用私钥进行签名
	} else {
		tokenString, tokenErr = t.SignedString(j.Key) // 使用密钥进行签名
	}
	if tokenErr != nil {
		return nil, tokenErr // 如果签名失败，返回错误
	}

	// 创建 JwtResponse 结构体实例，并设置新的主 JWT
	jr := &JwtResponse{
		Token: tokenString,
	}

	// 生成新的刷新令牌
	refreshToken, err := j.refreshToken(t)
	if err != nil {
		return nil, err // 如果生成刷新令牌失败，返回错误
	}
	jr.RefreshToken = refreshToken // 设置刷新令牌

	// 如果配置了发送 Cookie 的选项，将新的 token 设置到 Cookie 中
	if j.SendCookie {
		if j.CookieName == "" {
			j.CookieName = JWTToken // 如果未指定 Cookie 名称，使用默认值
		}
		if j.CookieMaxAge == 0 {
			j.CookieMaxAge = expire.Unix() - j.TimeFuc().Unix() // 设置 Cookie 的最大存活时间
		}
		// 设置 Cookie
		ctx.SetCookie(j.CookieName, tokenString, int(j.CookieMaxAge), "/", j.CookieDomain, j.SecureCookie, j.CookieHTTPOnly)
	}

	return jr, nil // 返回生成的 JwtResponse 结构体实例
}

// AuthInterceptor jwt 登录中间件，检查请求头或 Cookie 中是否有有效的 token
func (j *JwtHandler) AuthInterceptor(next web.HandlerFunc) web.HandlerFunc {
	return func(ctx *web.Context) {
		if j.Header == "" {
			j.Header = "Authorization" // 如果未指定头部字段名称，使用默认值
		}
		// 从请求头中获取 token
		token := ctx.R.Header.Get(j.Header)
		if token == "" {
			if j.SendCookie {
				cookie, err := ctx.R.Cookie(j.CookieName)
				if err != nil {
					j.AuthErrorHandler(ctx, err) // 如果获取 Cookie 失败，调用错误处理函数
					return
				}
				token = cookie.String()
			}
		}
		if token == "" {
			j.AuthErrorHandler(ctx, errors.New("token is null")) // 如果没有 token，调用错误处理函数
			return
		}

		// 解析 token
		t, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
			if j.usingPublicKeyAlgo() {
				return j.PrivateKey, nil // 使用私钥进行验证
			} else {
				return j.Key, nil // 使用密钥进行验证
			}
		})
		if err != nil {
			j.AuthErrorHandler(ctx, err) // 如果解析失败，调用错误处理函数
			return
		}
		// 获取 token 的声明（claims）
		claims := t.Claims.(jwt.MapClaims)
		ctx.Set("jwt_claims", claims) // 将 claims 设置到上下文中
		next(ctx)                     // 调用下一个处理函数
	}
}

// AuthErrorHandler 认证错误处理函数
func (j *JwtHandler) AuthErrorHandler(ctx *web.Context, err error) {
	if j.AuthHandler == nil {
		ctx.W.WriteHeader(http.StatusUnauthorized) // 如果未指定错误处理函数，返回 401 状态码
	} else {
		j.AuthHandler(ctx, err) // 调用自定义错误处理函数
	}
}
