# 制作一个简单的Golang Web框架

> 本文参考[7天用Go从零实现Web框架Gee教程](https://geektutu.com/post/gee.html)

## 前言

`Golang`本身的标准库`net/http`已经十分强大了，基本上可以利用其提供的库函数就可以直接进行Web应用开发了，本文基于`net/http`实现一个简单的Web框架，一方面为了体会一般Web框架的设计思想与解决的问题，也为了我们更熟悉`golang`与`net/http`的用法

## 接收一个HTTP请求

首先在`net/http`标准库中，我们需要先引入以下几个函数

### 开始监听

#### `http.ListenAndServe()`

```go
func ListenAndServe(addr string, handler Handler) error {
	server := &Server{Addr: addr, Handler: handler}
	return server.ListenAndServe()
}
```

该函数接收两个参数，`addr`为监听套接字，即`ip:port`,`handler`是一个Handler接口的实例。

该函数本质上是利用传入的参数构建一个`http.Server{}`的实例，再调用该实例的`ListenAndServer`方法。

```golang
func main(){
  http.ListenAndServe("127.0.0.1:8000", nil)
}
```



#### `http.Server{}.ListenAndServe()`

```go
func (srv *Server) ListenAndServe() error {
	if srv.shuttingDown() {
		return ErrServerClosed
	}
	addr := srv.Addr
	if addr == "" {
		addr = ":http"
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return srv.Serve(ln)
}
```

该函数无参数，使用Server对象中的属性进行环境配置。

```golang
func main(){
  server:=http.Server{
    Addr:"127.0.0.1"
  }
  server.ListenAndServe()
}
```

#### 比较

可以看出,最终创建一个监听服务是通过`http.Server{}.ListenAndServe()`实现的，当我们需要对多项规则进行配置时，可以直接生成一个`http.Server{}`实例，配置其属性，然后创建监听服务。

当我们仅仅对`Addr`和`Handler`两项参数进行配置时，可以直接使用`http.ListenAndServe()`方法，该方法会自动创建`http.Server{}`实例。

### 处理路由

#### `http.Handle()`

```go
func Handle(pattern string, handler Handler) { 
  DefaultServeMux.Handle(pattern, handler) 
}

```

该函数接收两个参数，`pattern`为url地址，`handler`为`Handler`接口类型的实例

#### `http.HandleFunc()`

```go
func HandleFunc(pattern string, handler func(ResponseWriter, *Request)) {
	DefaultServeMux.HandleFunc(pattern, handler)
}
```

该函数接收两个参数，`pattern`为url地址，`handler`为`func(ResponseWriter, *Request)`类型的一个函数，随后调用`http.ServeMux{}.HandleFunc()`方法

```go
func (mux *ServeMux) HandleFunc(pattern string, handler func(ResponseWriter, *Request)) {
	if handler == nil {
		panic("http: nil handler")
	}
	mux.Handle(pattern, HandlerFunc(handler))
}
```

可见该方法是将`func(ResponseWriter, *Request))`通过`HandlerFunc`函数转换成了一个`Handler`接口类型的实例，再调用`http.ServeMux{}.Handle()`函数注册到`Server`中

#### 比较

一般而言，当我们需要处理多个不同的url请求时，则需要多个`Handler`类型的实例，而由于对于每个`Handler类型`都要声明`strcut`且实现`ServeHTTP()`方法，因此我们大多直接使用`http.HandleFunc()`函数来自动转换`Handler`接口



## 监听多个HTTP请求

当我们需要对多个url请求进行监听时，需要使用多路复用器。

当我们主动的创建一个多路复用器时，则需要通过该多路复用器来绑定`Handler`，并且将该多路复用器作为`Server`的`Handler`

```go
func main(){
  mux:=http.NewServeMux()
  mux.Handle("/static",http.StripPrefix("/static/",files))
  mux.HandleFunc("/",index)
  server:=&http.Server{
    Addr:"127.0.0.1:8080",
    Handler:mux,
  }
  server.ListenAndServe()
}
```

当我们不想主动的创建多路复用器，则需要将`Server`的`Handler`置为`nil`，则`Server`的`Handler`会自动变成`DefaultServeMux`。

## 创建一个Handler

由`net/http`的源码可知一个实现了`ServeHTTP()`的类型即为一个`Handler`

```go
type Engine struct{}

func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request){
  fmt.Fprintf(w, "URL.Path = %q\n", req.URL.Path)
}

func main(){
  e:=new(Engine)
  http.ListenAndServe(":8080",e)
}
```

如上所示，我们定义了一个结构体`Engine`并且实现了`ServeHTTP()`，因此该结构体就是一个`Handle`。但是请注意，由于这里我们直接将`Server`的`Handler`	置为了`Engine`的一个实例，而不是一个多路复用器，因此我们不能再使用`http.Handler()`等方法来为不同的url请求分配不同的`Handler`,而是只能在`Engine`实例的`ServeHTTP()`方法中处理所有的路由请求。

## 实现多Route监听

由于只能在`ServeHTTP()`中对请求信息进行处理，因此我们需要主动的维护一张路由表，在`ServeHTTP()`中对比请求地址和路由表来进行相应处理

```go
type HandlerFunc func(http.ResponseWriter, *http.Request)
Route:=make(map[string]HandlerFunc)
Route["/"]=func(http.ResponseWriter, *http.Request){
  fmt.Fprintf(w, "您正在访问跟路径")
}
Route["/hello"]=func(http.ResponseWriter, *http.Request){
  fmt.Fprintf(w, "hello world")
}

type Engine struct{}

func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request){
  key := req.URL.Path
  if handler, ok := engine.router[key]; ok {
		handler(w, req)
	} else {
		fmt.Fprintf(w, "404 NOT FOUND: %s\n", req.URL)
	}
}

func main(){
  e:=new(Engine)
  http.ListenAndServe(":8080",e)
}
```

那么作为一个提供给用户使用的web框架，让用户自行在`Route`中添加键值对未免太不优雅，因此我们需要对其进行封装，使得用户在使用时与使用`net/http`或其他web框架无异，只需要设定路由与对应的处理函数即可。同时我们对HTTP请求方法也加以辨别，是其作为路由选择的一部分。

```go
//首先我们将路由表作为Engine的一部分
type HandlerFunc func(http.ResponseWriter, *http.Request)
type Engine struct{
  Route map[string]HandlerFunc
}

//实现一个构造器以便用户创建框架实例
func New() *Engine{
  return &Engine{Route:make(map[string]HandlerFunc)}
}

//实现一个GET方法可以处理GET请求
func (e *Engine) GET(pattern string,handler HandlerFunc){
  key := "GET" + "-" + pattern
	e.router[key] = handler
}

//实现一个POST方法可以处理POST请求
func (e *Engine)POST(pattern string,handler HandlerFunc){
  key:= "POST" + "-" + pattern
  e.router[key]=handler
}
```

最后我们将Web服务的启动函数也进行封装，以便达到一个统一的用户体验

```go
func (e *Engine) Run(addr string) (err error) {
	return http.ListenAndServe(addr, e)
}
```

## Web框架1.0

至此，我们已经基于`net/http`封装了一个简单的可以接收请求的Web框架，整体代码如下

`gee.go`

```go
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
```



接下来可以编写简单的`main.go`来使用该框架

```go
package main

import (
	"fmt"
	"net/http"
	"./gee"
)

func main() {
	r := gee.New()
	r.GET("/", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "URL.Path = %q\n", req.URL.Path)
	})

	r.GET("/hello", func(w http.ResponseWriter, req *http.Request) {
		for k, v := range req.Header {
			fmt.Fprintf(w, "Header[%q] = %q\n", k, v)
		}
	})

	r.Run(":8080")
}
```




