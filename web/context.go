package web

import (
	"errors"
	"github.com/ygb616/web/binding"
	myLog "github.com/ygb616/web/log"
	"github.com/ygb616/web/render"
	"github.com/ygb616/web/util"
	"html/template"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
)

const defaultMultipartMemory = 30 << 20 //30M

type Context struct {
	W                     http.ResponseWriter
	R                     *http.Request
	E                     *Engine
	queryCache            url.Values
	formCache             url.Values
	DisallowUnknownFields bool
	IsValidate            bool
	StatusCode            int
	Logger                *myLog.Logger
	Keys                  map[string]any
	mu                    sync.RWMutex
	sameSize              http.SameSite
}

func (c *Context) SetSameSize(site http.SameSite) {
	c.sameSize = site
}

func (c *Context) FormFile(name string) (*multipart.FileHeader, error) {
	req := c.R
	if err := req.ParseMultipartForm(defaultMultipartMemory); err != nil {
		return nil, err
	}
	file, header, err := req.FormFile(name)
	if err != nil {
		return nil, err
	}
	err = file.Close()
	if err != nil {
		return nil, err
	}
	return header, nil
}

func (c *Context) SaveUploadedFile(file *multipart.FileHeader, dst string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, src)
	return err
}

func (c *Context) MultipartForm() (*multipart.Form, error) {
	err := c.R.ParseMultipartForm(defaultMultipartMemory)
	return c.R.MultipartForm, err
}

func (c *Context) initQueryCache() {
	if c.R != nil {
		c.queryCache = c.R.URL.Query()
	} else {
		c.queryCache = url.Values{}
	}
}

// http://xxx.com/user/add?id=1&age=20&username=张三
func (c *Context) GetQuery(key string) string {
	c.initQueryCache()
	return c.queryCache.Get(key)
}

func (c *Context) HTML(status int, html string) error {
	return c.Render(status, &render.HTML{
		Data:       html,
		IsTemplate: false,
	})
}

func (c *Context) GetQueryArray(key string) (values []string, ok bool) {
	c.initQueryCache()
	values, ok = c.queryCache[key]
	return
}

func (c *Context) QueryArray(key string) (values []string) {
	c.initQueryCache()
	values, _ = c.queryCache[key]
	return
}

func (c *Context) DefaultQuery(key, defaultValue string) string {
	array, ok := c.GetQueryArray(key)
	if !ok {
		return defaultValue
	}
	return array[0]
}

func (c *Context) QueryMap(key string) (dicts map[string]string) {
	dicts, _ = c.GetQueryMap(key)
	return
}

func (c *Context) GetQueryMap(key string) (map[string]string, bool) {
	c.initQueryCache()
	return c.get(c.queryCache, key)
}

func (c *Context) get(m map[string][]string, key string) (map[string]string, bool) {
	//user[id]=1&user[name]=张三
	dicts := make(map[string]string)
	exist := false
	for k, value := range m {
		if i := strings.IndexByte(k, '['); i >= 1 && k[0:i] == key {
			if j := strings.IndexByte(k[i+1:], ']'); j >= 1 {
				exist = true
				dicts[k[i+1:][:j]] = value[0]
			}
		}
	}
	return dicts, exist
}

func (c *Context) initFormCache() {
	if c.formCache == nil {
		c.formCache = make(url.Values)
		req := c.R
		if err := req.ParseMultipartForm(defaultMultipartMemory); err != nil {
			if !errors.Is(err, http.ErrNotMultipart) {
				log.Println(err)
			}
		}
		c.formCache = c.R.PostForm
	}
}

func (c *Context) GetPostForm(key string) (string, bool) {
	if values, ok := c.GetPostFormArray(key); ok {
		return values[0], ok
	}
	return "", false
}

func (c *Context) PostFormArray(key string) (values []string) {
	values, _ = c.GetPostFormArray(key)
	return
}

func (c *Context) GetPostFormArray(key string) (values []string, ok bool) {
	c.initFormCache()
	values, ok = c.formCache[key]
	return
}

func (c *Context) GetPostFormMap(key string) (map[string]string, bool) {
	c.initFormCache()
	return c.get(c.formCache, key)
}

func (c *Context) PostFormMap(key string) (dicts map[string]string) {
	dicts, _ = c.GetPostFormMap(key)
	return
}

func (c *Context) HTMLTemplate(name string, data any, filenames ...string) error {

	c.W.Header().Set("Content-Type", "text/html; charset=utf-8")
	t, err := template.New(name).ParseFiles(filenames...) //加载传入的模版名称
	if err != nil {
		return err
	}
	err = t.Execute(c.W, data)
	return err
}

func (c *Context) HTMLTemplateGlob(name string, data any, pattern string) error {

	c.W.Header().Set("Content-Type", "text/html; charset=utf-8")
	t, err := template.New(name).ParseGlob(pattern) //加载传入的模版通配表达式
	if err != nil {
		return err
	}
	err = t.Execute(c.W, data)
	return err
}

func (c *Context) Template(name string, data any) error {
	return c.Render(http.StatusOK, &render.HTML{
		Data:       data,
		IsTemplate: true,
		Name:       name,
		Template:   c.E.HTMLRender.Template,
	})
}

func (c *Context) JSON(status int, data any) error {
	return c.Render(status, &render.JSON{
		Data: data,
	})
}

func (c *Context) XML(status int, data any) error {
	return c.Render(status, &render.XML{Data: data})
}

func (c *Context) File(filename string) {
	http.ServeFile(c.W, c.R, filename)
}

func (c *Context) FileAttachment(filepath, filename string) {
	if util.IsASCII(filename) {
		c.W.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	} else {
		c.W.Header().Set("Content-Disposition", `attachment; filename*=UTF-8''`+url.QueryEscape(filename))
	}
	http.ServeFile(c.W, c.R, filepath)
}

// FileFromFS filepath是相对文件系统的路径
func (c *Context) FileFromFS(filepath string, fs http.FileSystem) {
	defer func(old string) {
		c.R.URL.Path = old
	}(c.R.URL.Path)

	c.R.URL.Path = filepath

	http.FileServer(fs).ServeHTTP(c.W, c.R)
}

func (c *Context) Redirect(status int, url string) error {
	//如果小于300 大于308 并且不等于 201 则抛出异常
	return c.Render(status, &render.Redirect{
		Code:     status,
		Request:  c.R,
		Location: url,
	})
}

func (c *Context) String(status int, format string, values ...any) error {
	return c.Render(status, &render.String{
		Format: format,
		Data:   values,
	})
}

func (c *Context) Render(statusCode int, r render.Render) error {
	//如果设置了statusCode，对header的修改就不生效了
	err := r.Render(c.W, statusCode)
	c.StatusCode = statusCode
	//多次调用 WriteHeader 就会产生这样的警告 superfluous response.WriteHeader
	return err
}

func (c *Context) BindJson(data any) error {
	json := binding.JSON
	json.DisallowUnknownFields = true
	json.IsValidate = true
	return c.MustBindWith(data, &json)
}

func (c *Context) MustBindWith(data any, bind binding.Binding) error {
	if err := c.ShouldBind(data, bind); err != nil {
		c.W.WriteHeader(http.StatusBadRequest)
		return err
	}
	return nil
}

func (c *Context) ShouldBind(data any, bind binding.Binding) error {
	return bind.Bind(c.R, data)
}

func (c *Context) BindXML(data any) error {
	return c.MustBindWith(data, binding.XML)
}

func (c *Context) Fail(code int, msg string) {
	c.String(code, msg)
}

func (c *Context) HandlerWithError(code int, obj any, err error) {
	if err != nil {
		statusCode, data := c.E.errorHandler(err)
		_ = c.JSON(statusCode, data)
		return
	}
	_ = c.JSON(code, obj)
}

// Set 方法将键值对存储在 Context 中
func (c *Context) Set(key string, value any) {
	c.mu.Lock() // 加写锁，防止并发写入
	if c.Keys == nil {
		c.Keys = make(map[string]any) // 如果 Keys 为空，初始化它
	}

	c.Keys[key] = value // 将键值对存储在 Keys 中
	c.mu.Unlock()       // 释放写锁
}

// Get 方法根据键获取值，并返回值和是否存在的布尔值
func (c *Context) Get(key string) (value any, exists bool) {
	c.mu.RLock()                // 加读锁，允许并发读取
	value, exists = c.Keys[key] // 从 Keys 中获取值
	c.mu.RUnlock()              // 释放读锁
	return                      // 返回值和是否存在
}

// SetCookie 在 HTTP 响应中设置一个 Cookie
func (c *Context) SetCookie(name, value string, maxAge int, path, domain string, secure, httpOnly bool) {
	// 如果未指定路径，则默认设置为 "/"
	if path == "" {
		path = "/"
	}

	// 调用 http.SetCookie 方法，在响应中添加一个新的 Cookie
	http.SetCookie(c.W, &http.Cookie{
		Name:     name,                   // Cookie 名称
		Value:    url.QueryEscape(value), // Cookie 值，进行 URL 编码
		MaxAge:   maxAge,                 // Cookie 的最大存活时间，单位为秒
		Path:     path,                   // Cookie 的路径
		Domain:   domain,                 // Cookie 的域名
		SameSite: c.sameSize,             // Cookie 的 SameSite 属性，防止 CSRF 攻击
		Secure:   secure,                 // 是否为安全 Cookie（仅通过 HTTPS 发送）
		HttpOnly: httpOnly,               // 是否将 Cookie 设置为 HTTPOnly（客户端 JavaScript 无法访问）
	})
}

func (c *Context) GetHeader(key string) string {
	return c.R.Header.Get(key)
}
