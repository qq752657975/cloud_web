package web

import (
	"fmt"
	"github.com/ygb616/web/config"
	myLog "github.com/ygb616/web/log"
	"github.com/ygb616/web/render"
	"github.com/ygb616/web/util"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"sync"
)

const ANY = "ANY"

type HandlerFunc func(ctx *Context) // 定义函数类型

type MiddlewareFunc func(handler HandlerFunc) HandlerFunc //定义中间件函数类型

type router struct {
	groups []*routerGroup
	engine *Engine
}

func (r *router) Group(name string) *routerGroup {
	g := &routerGroup{
		groupName:          name,
		handlerMap:         make(map[string]map[string]HandlerFunc),
		middlewaresFuncMap: make(map[string]map[string][]MiddlewareFunc),
		handlerMethodMap:   make(map[string][]string),
		treeNode:           &treeNode{name: "/", children: make([]*treeNode, 0)},
	}
	g.Use(r.engine.Middles...)
	r.groups = append(r.groups, g)
	return g
}

func (r *routerGroup) handle(name string, method string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	// 检查 handlerMap 中是否已存在指定名称的路由
	_, ok := r.handlerMap[name]
	if !ok {
		// 如果不存在，初始化一个新的 map
		r.handlerMap[name] = make(map[string]HandlerFunc)
		r.middlewaresFuncMap[name] = make(map[string][]MiddlewareFunc)
	}
	_, ok = r.handlerMap[name][method]
	if ok {
		panic("有重复路由")
	}
	// 将处理函数存储在 handlerMap 中
	r.handlerMap[name][method] = handlerFunc
	// 将路由名称添加到 handlerMethodMap 中
	r.middlewaresFuncMap[name][method] = append(r.middlewaresFuncMap[name][method], middlewareFunc...)
	// 将路由名称插入到 treeNode 中，以便进行路由匹配
	r.treeNode.Put(name)
}

func (r *routerGroup) Use(middlewares ...MiddlewareFunc) {
	r.middlewares = append(r.middlewares, middlewares...)
}

func (r *routerGroup) Any(name string, handlerFunc HandlerFunc) {
	r.handle(name, "ANY", handlerFunc)
}

func (r *routerGroup) Handle(name string, method string, handlerFunc HandlerFunc) {
	//method有效性做校验
	r.handle(name, method, handlerFunc)
}

func (r *routerGroup) Get(name string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodGet, handlerFunc, middlewareFunc...)
}
func (r *routerGroup) Post(name string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodPost, handlerFunc, middlewareFunc...)
}
func (r *routerGroup) Delete(name string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodDelete, handlerFunc, middlewareFunc...)
}
func (r *routerGroup) Put(name string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodPut, handlerFunc, middlewareFunc...)
}
func (r *routerGroup) Patch(name string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodPatch, handlerFunc, middlewareFunc...)
}
func (r *routerGroup) Options(name string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodOptions, handlerFunc, middlewareFunc...)
}
func (r *routerGroup) Head(name string, handlerFunc HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.handle(name, http.MethodHead, handlerFunc, middlewareFunc...)
}

// methodHandle 处理中间件逻辑
func (r *routerGroup) methodHandle(name string, method string, h HandlerFunc, ctx *Context) {
	//通用中间件
	if r.middlewares != nil {
		for _, middlewareFunc := range r.middlewares {
			h = middlewareFunc(h)
		}
	}
	//组路由级别
	funcMidis := r.middlewaresFuncMap[name][method]
	if funcMidis != nil {
		for _, middlewareFunc := range funcMidis {
			h = middlewareFunc(h)
		}
	}
	h(ctx)
}

// routerGroup 表示一组路由及其处理函数
type routerGroup struct {
	// groupName 是路由组的名称或前缀，用于组织和管理路由
	groupName string
	// handlerMap 是一个多级映射，保存每个路由和 HTTP 方法对应的处理函数
	// 第一层键是路由路径，第二层键是 HTTP 方法 (如 "GET", "POST")，值是相应的处理函数
	handlerMap map[string]map[string]HandlerFunc
	// middlewaresFuncMap 是一个多级映射，保存每个路由和 HTTP 方法对应的中间件函数
	middlewaresFuncMap map[string]map[string][]MiddlewareFunc
	// handlerMethodMap 保存每个路由路径支持的 HTTP 方法列表
	// 键是路由路径，值是该路径支持的 HTTP 方法的切片
	handlerMethodMap map[string][]string
	// treeNode 是该路由组的树节点，用于存储路由树结构，实现高效路由匹配
	treeNode *treeNode
	//路由中间件集合
	middlewares []MiddlewareFunc
}

type ErrorHandler func(err error) (int, any)

type Engine struct {
	*router
	funcMap      template.FuncMap
	HTMLRender   render.HTMLRender
	pool         sync.Pool
	Logger       *myLog.Logger
	Middles      []MiddlewareFunc
	errorHandler ErrorHandler
}

func New() *Engine {
	r := &router{}
	engine := &Engine{
		router:     r,
		funcMap:    nil,
		HTMLRender: render.HTMLRender{},
		Logger:     myLog.Default(),
	}
	engine.pool.New = func() any {
		return engine.allocateContext()
	}
	r.engine = engine
	return engine
}

// Default 函数创建并返回一个默认配置的 Engine 实例
func Default() *Engine {
	// 创建一个新的 Engine 实例
	engine := New()

	// 设置 Logger 为默认日志记录器
	engine.Logger = myLog.Default()

	// 从配置中获取日志路径，如果存在则设置日志路径
	logPath, ok := config.Conf.Log["path"]
	if ok {
		engine.Logger.SetLogPath(logPath.(string))
	}

	// 使用中间件 Logging 和 Recovery
	engine.Use(Logging, Recovery)

	// 设置 router 的 engine 字段为当前的 engine 实例
	engine.router.engine = engine

	// 返回配置好的 Engine 实例
	return engine
}

func (e *Engine) Use(middles ...MiddlewareFunc) {
	e.Middles = append(e.Middles, middles...)
}

func (e *Engine) allocateContext() any {
	return &Context{E: e}
}

func (e *Engine) SetFuncMap(funcMap template.FuncMap) {
	e.funcMap = funcMap
}

// LoadTemplate LoadTemplateGlob 加载所有模板
func (e *Engine) LoadTemplate(pattern string) {
	t := template.Must(template.New("").Funcs(e.funcMap).ParseGlob(pattern))
	e.SetHtmlTemplate(t)
}

func (e *Engine) SetHtmlTemplate(t *template.Template) {
	e.HTMLRender = render.HTMLRender{Template: t}
}

func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := e.pool.Get().(*Context)
	ctx.W = w
	ctx.R = r
	ctx.Logger = e.Logger
	e.httpRequestHandler(ctx, w, r)
	e.pool.Put(ctx)
}

// Run 启动 HTTP 服务器，监听指定的端口
func (e *Engine) Run(port int) {
	// 将根 URL ("/") 与当前的 Engine 实例关联，这样所有的请求都会由该实例处理
	http.Handle("/", e)

	// 使用指定的端口启动 HTTP 服务器
	// strconv.Itoa(port) 将端口号转换为字符串形式，组合成 ":port" 格式的地址
	err := http.ListenAndServe(":"+strconv.Itoa(port), nil)

	// 如果启动服务器时发生错误，记录并终止程序
	if err != nil {
		log.Fatal(err)
	}
}

func (e *Engine) httpRequestHandler(ctx *Context, w http.ResponseWriter, r *http.Request) {
	// 获取请求的方法 (GET, POST, etc.)
	method := r.Method
	// 遍历所有路由组
	for _, group := range e.groups {
		// 获取路由名，这里使用了自定义的函数 SubStringLast
		// 比如：从请求URI中提取路由组的名称
		routerName := util.SubStringLast(r.URL.Path, "/"+group.groupName)
		// 获取匹配的路由节点
		node := group.treeNode.Get(routerName)
		if node != nil && node.isEnd {
			// 尝试获取通配符(ANY)的处理函数
			handle, ok := group.handlerMap[node.routerName][ANY]
			if ok {
				// 如果找到了通配符处理函数，调用并返回
				group.methodHandle(node.routerName, ANY, handle, ctx)
				return
			}
			// 尝试获取具体方法(GET, POST等)的处理函数
			handle, ok = group.handlerMap[node.routerName][method]
			if ok {
				// 如果找到了具体方法的处理函数，调用并返回
				group.methodHandle(node.routerName, method, handle, ctx)
				return
			}
			// 如果没有找到匹配的处理函数，返回405 Method Not Allowed
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintf(w, "%s %s not allowed \n", r.RequestURI, method)
			return
		}
	}
	// 如果没有匹配的路由，返回404 Not Found
	w.WriteHeader(http.StatusNotFound)
	_, err := fmt.Fprintf(w, "%s  not found \n", r.RequestURI)
	if err != nil {
		return
	}
}

func (c *Context) ErrorHandle(err error) {
	code, data := c.E.errorHandler(err)
	_ = c.JSON(code, data)
}

func (e *Engine) RegisterErrorHandler(err ErrorHandler) {
	e.errorHandler = err
}

func (e *Engine) RunTLS(addr, certFile, keyFile string) {
	err := http.ListenAndServeTLS(addr, certFile, keyFile, e.Handler())
	// 调用 http.ListenAndServeTLS 开启一个 HTTPS 服务
	// 参数：
	// addr：服务监听的地址（如 ":443"）
	// certFile：证书文件路径
	// keyFile：私钥文件路径
	// e.Handler()：用于处理 HTTP 请求的处理器

	if err != nil {
		log.Fatal(err)
		// 如果出现错误，记录错误并终止程序
	}
}

func (e *Engine) Handler() http.Handler {
	return e
}

// LoadTemplateGlobByConf 从配置文件中加载模板文件
func (e *Engine) LoadTemplateGlobByConf() {
	// 从配置中获取模板文件的匹配模式
	pattern, ok := config.Conf.Template["pattern"]
	if !ok {
		// 如果配置中没有找到 pattern，抛出异常
		panic("config pattern not exist")
	}
	// 解析匹配模式下的所有模板文件，并将解析后的模板赋给 t
	t := template.Must(template.New("").Funcs(e.funcMap).ParseGlob(pattern.(string)))
	// 设置 HTML 模板
	e.SetHtmlTemplate(t)
}
