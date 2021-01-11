package gee

import (
	"fmt"
	"net/http"
)

type HandlerFunc func(http.ResponseWriter, *http.Request)

type Engine struct {
	Route map[string]HandlerFunc
}

//实现一个构造器以便用户创建框架实例
func New() *Engine{
  return &Engine{Route:make(map[string]HandlerFunc)}
}

//实现一个GET方法可以处理GET请求
func (e *Engine) GET(pattern string,handler HandlerFunc){
  key := "GET" + "-" + pattern
	e.Route[key] = handler
}

//实现一个POST方法可以处理POST请求
func (e *Engine)POST(pattern string,handler HandlerFunc){
  key:= "POST" + "-" + pattern
  e.Route[key]=handler
}

func (e *Engine) Run(addr string) (err error) {
	return http.ListenAndServe(addr, e)
}

func (e *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request){
  key := req.Method + "-" + req.URL.Path
  if handler, ok := e.Route[key]; ok {
		handler(w, req)
	} else {
		fmt.Fprintf(w, "404 NOT FOUND: %s\n", req.URL)
	}
}