package web

import (
	"html/template"
	"net/http"
)

type Context struct {
	W http.ResponseWriter
	R *http.Request
}

func (c *Context) HTML(status int, html string) error {

	c.W.Header().Set("Content-Type", "text/html; charset=utf-8")
	//设置返回状态是200 ，默认不设置的话，如果调用了write这个方法，实际上默认返回状态200
	c.W.WriteHeader(status)
	_, err := c.W.Write([]byte(html))
	return err
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
