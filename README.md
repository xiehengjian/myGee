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



## 为什么需要Context

在使用`net/http`进行Web编程时，构造完整的响应需要写大量繁琐重复的代码，为了提高效率，我们可以将HTTP传输的内容封装到一个`Context`中，并为其实现一些常用的方法。

首先将请求与响应封装到`Context`中

```golang
type Context struct{
	Writer http.ResponseWriter
	Req    *http.Request
}
```

至此我们可以将handler函数写成如下形式了

```go
type HandlerFunc func(c *Context)
```

当最终`ServeHTTP()`处理请求时，将其接受的`	http.ResponseWriter  `与`*http.Request`构造成一个`Context`,然后再调用`Route`中保存的用户编写的handler函数即可

```go
func (e *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request){
  key := req.Method + "-" + req.URL.Path
  c := newContext(w, req)
  if handler, ok := e.Route[key]; ok {
		handler(c)
	} else {
		fmt.Fprintf(w, "404 NOT FOUND: %s\n", req.URL)
	}
}
```

## 封装一些常用的功能

为了方便处理http传输的信息，我们先对`Context`进行扩充

```go
type Context struct {
	Writer http.ResponseWriter
	Req    *http.Request
	Path   string
	Method string
	StatusCode int
}
```

为`Context`实现构造器

```go
func newContext(w http.ResponseWriter, req *http.Request) *Context {
	return &Context{
		Writer: w,
		Req:    req,
		Path:   req.URL.Path,
		Method: req.Method,
	}
}
```

封装一个设置HEAD的方法

```go
func (c *Context)SetHead(key string,value string){
  c.Writer.Head().Set(key,value)
}
```

封装一个返回字符串的方法

```go
func (c *Context) String(code int, format string, values ...interface{}) {
	c.SetHeader("Content-Type", "text/plain")
	c.Status(code)
	c.Writer.Write([]byte(fmt.Sprintf(format, values...)))
}
```

封装一个返回JSON数据的方法

```go
func (c *Context) JSON(code int, obj interface{}) {
	c.SetHeader("Content-Type", "application/json")
	c.Status(code)
	encoder := json.NewEncoder(c.Writer)
	if err := encoder.Encode(obj); err != nil {
		http.Error(c.Writer, err.Error(), 500)
	}
}
```

封装一个返回HTML数据的方法

```go
func (c *Context) HTML(code int, html string) {
	c.SetHeader("Content-Type", "text/html")
	c.Status(code)
	c.Writer.Write([]byte(html))
}
```

至此，我们已经实现了`Context`的定义和相关方法的实现，我们将其单独放到`context.go`中。

## Web框架1.1

`context.go`

```go
package gee 

import (
	"net/http"
	"fmt"
	"encoding/json"

)
type Context struct {
	Writer http.ResponseWriter
	Req    *http.Request
	Path   string
	Method string
	StatusCode int
}

func newContext(w http.ResponseWriter, req *http.Request) *Context {
	return &Context{
		Writer: w,
		Req:    req,
		Path:   req.URL.Path,
		Method: req.Method,
	}
}

func (c *Context) Status(code int) {
	c.StatusCode = code
	c.Writer.WriteHeader(code)
}
func (c *Context)SetHeader(key string,value string){
	c.Writer.Header().Set(key,value)
  }

  func (c *Context) String(code int, format string, values ...interface{}) {
	c.SetHeader("Content-Type", "text/plain")
	c.Status(code)
	c.Writer.Write([]byte(fmt.Sprintf(format, values...)))
}

func (c *Context) JSON(code int, obj interface{}) {
	c.SetHeader("Content-Type", "application/json")
	c.Status(code)
	encoder := json.NewEncoder(c.Writer)
	if err := encoder.Encode(obj); err != nil {
		http.Error(c.Writer, err.Error(), 500)
	}
}

func (c *Context) HTML(code int, html string) {
	c.SetHeader("Content-Type", "text/html")
	c.Status(code)
	c.Writer.Write([]byte(html))
}
```

`gee.go`

```go
package gee

import (
	"fmt"
	"net/http"
)

type HandlerFunc func(* Context)

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
  c := newContext(w, req)
  if handler, ok := e.Route[key]; ok {
		handler(c)
	} else {
		fmt.Fprintf(w, "404 NOT FOUND: %s\n", req.URL)
	}
}
```

`main.go`

```go
package main

import (
	"fmt"
	"./gee"
)

func main() {
	r := gee.New()
	r.GET("/", func(c *gee.Context) {
		fmt.Fprintf(c.Writer, "URL.Path = %q\n", c.Path)
	})

	r.GET("/hello", func(c *gee.Context) {
		for k, v := range c.Req.Header {
			fmt.Fprintf(c.Writer, "Header[%q] = %q\n", k, v)
		}
	})

	r.Run(":8080")
}
```

## 更高级的路由

对于一个Web框架来说，支持动态路由是一个基本的功能，因此我们的框架也需要实现动态路由的功能。在此之前我们的路由表设计成了哈希表，可以非常高效的进行路由匹配，但是在匹配动态路由时需要进行逐步的查找与搜索，那么哈希表则不太适用了。

目前我们计划设计支持两种模式的动态路由，`:name`和`*filepath`

因此我们采用一种叫做`Trie`树的数据结构，也称为前缀树或字典树。

> 在计算机科学中，trie，又称前缀树或字典树，是一种有序树，用于保存关联数组，其中的键通常是字符串。与二叉查找树不同，键不是直接保存在节点中，而是由节点在树中的位置决定。一个节点的所有子孙都有相同的前缀，也就是这个节点对应的字符串，而根节点对应空字符串。一般情况下，不是所有的节点都有对应的值，只有叶子节点和部分内部节点所对应的键才有相关的值。
>
> trie中的键通常是字符串，但也可以是其它的结构。trie的算法可以很容易地修改为处理其它结构的有序序列，比如一串数字或者形状的排列。比如，bitwise trie中的键是一串位元，可以用于表示整数或者内存地址

首先我们设计的Trie树的结构如下

```go
type node struct{
  pattern string  //待匹配的路由
  part string 	//匹配部分
  children []*node //子节点
  isWild bool  //是否是通配符，为true时说明当前结点为通配结点
}
```

由于`Route`不再使用简单的哈希表作为底层数据结构，因此我们需要重新设计一下`Route`

```go
type Route struct{
  roots    map[string]*node//为每一种http方法构建一颗trie树
	handlers map[string]HandlerFunc
}
// roots key eg, roots['GET'] roots['POST']
// handlers key eg, handlers['GET-/p/:lang/doc'], handlers['POST-/p/book']
```



接下来我们需要解决的是路由规则的注册，对于用户提供的动态路由模板，如何将其插入到路由表中。

首先我们抽象出一个独立的添加路由的方法`addRoute()`

```go
func (r *router) addRoute(method string, pattern string, handler HandlerFunc) {
  parts := parsePattern(pattern)//将路由分解，如/index/:id拆分成["index",":id"]

	key := method + "-" + pattern
  //构建不同HTTP方法的trie树
	_, ok := r.roots[method]
	if !ok {
		r.roots[method] = &node{}
	}
  //将该路由插入trie树中
	r.roots[method].insert(pattern, parts, 0)
  //绑定处理函数
	r.handlers[key] = handler
}
```

那么我们就需要实现一下插入的算法

```go
func (n *node) insert(pattern string, parts []string, height int) {
  //pattern为完整路由
  //parts为路径数组
  //height为路径数组索引
  
  //插入终止条件，即遍历完所有路径
	if len(parts) == height {
		n.pattern = pattern
		return
	}
	
  //取出当前需要插入的字符
	part := parts[height]
  //在当前节点的子结点中寻找该part
	child := n.matchChild(part)
  //如果找不到，就新建一个
	if child == nil {
		child = &node{part: part, isWild: part[0] == ':' || part[0] == '*'}
		n.children = append(n.children, child)
	}
  //递归的在子结点中插入剩下的部分
	child.insert(pattern, parts, height+1)
}

```

然后我们再来实现匹配的算法

```go
// 第一个匹配成功的节点，用于插入
func (n *node) matchChild(part string) *node {
  //遍历所有子结点
	for _, child := range n.children {
    //如果该子结点的part与匹配的相同
    //如果该子结点是通配结点
		if child.part == part || child.isWild {
			return child
		}
	}
	return nil
}
```

那么至此用户注册路由的方法就实现了.

接下来我们还需要实现对于请求路由的动态匹配问题了。

```go
func (r *router) getRoute(method string, path string) (*node, map[string]string) {
  //将路由分解，如/index/:id拆分成["index",":id"]
	searchParts := parsePattern(path)
  //构建路由变量与实际请求的映射
	params := make(map[string]string)
  //如果当前HTTP方法不存在路由树，说明不存在该路由，直接返回nil
	root, ok := r.roots[method]

	if !ok {
		return nil, nil
	}
  
	//寻找到匹配的路由对应的trie结点，则n.pattern则为用户注册的路由
	n := root.search(searchParts, 0)

  //n不为空，即找到了对应的trie结点
	if n != nil {
    //对用户注册的路由进行拆分
		parts := parsePattern(n.pattern)
    //遍历路由的各个路径项
		for index, part := range parts {
      //如果该路径为通配路径
			if part[0] == ':' {
        //params["name"]="jack"
				params[part[1:]] = searchParts[index]
			}
			if part[0] == '*' && len(part) > 1 {
        //params["filepath"]=""
				params[part[1:]] = strings.Join(searchParts[index:], "/")
				break
			}
		}
		return n, params
	}

	return nil, nil
}
```

然后实现`search()`

```go
func (n *node) search(parts []string, height int) *node {
  //当搜索到最后一个part
  //当前trie结点是*通配结点
	if len(parts) == height || strings.HasPrefix(n.part, "*") {
		if n.pattern == "" {
			return nil
		}
		return n
	}

  //获取当前需要匹配的part
	part := parts[height]
  //搜索包含该part的所有子结点
	children := n.matchChildren(part)
	//遍历子结点集合，继续往下搜索
	for _, child := range children {
		result := child.search(parts, height+1)
		if result != nil {
			return result
		}
	}

	return nil
}
```

```go
// 所有匹配成功的节点，用于查找
func (n *node) matchChildren(part string) []*node {
	nodes := make([]*node, 0)
	for _, child := range n.children {
		if child.part == part || child.isWild {
			nodes = append(nodes, child)
		}
	}
	return nodes
}
```

至此我们就完整的实现了动态路由的功能了。

## Web框架1.2

`trie.go`

```go
package gee

import (
	"fmt"
	"strings"
)

type node struct {
	pattern  string
	part     string
	children []*node
	isWild   bool
}

func (n *node) String() string {
	return fmt.Sprintf("node{pattern=%s, part=%s, isWild=%t}", n.pattern, n.part, n.isWild)
}

func (n *node) insert(pattern string, parts []string, height int) {
	if len(parts) == height {
		n.pattern = pattern
		return
	}

	part := parts[height]
	child := n.matchChild(part)
	if child == nil {
		child = &node{part: part, isWild: part[0] == ':' || part[0] == '*'}
		n.children = append(n.children, child)
	}
	child.insert(pattern, parts, height+1)
}

func (n *node) search(parts []string, height int) *node {
	if len(parts) == height || strings.HasPrefix(n.part, "*") {
		if n.pattern == "" {
			return nil
		}
		return n
	}

	part := parts[height]
	children := n.matchChildren(part)

	for _, child := range children {
		result := child.search(parts, height+1)
		if result != nil {
			return result
		}
	}

	return nil
}

func (n *node) travel(list *([]*node)) {
	if n.pattern != "" {
		*list = append(*list, n)
	}
	for _, child := range n.children {
		child.travel(list)
	}
}

func (n *node) matchChild(part string) *node {
	for _, child := range n.children {
		if child.part == part || child.isWild {
			return child
		}
	}
	return nil
}

func (n *node) matchChildren(part string) []*node {
	nodes := make([]*node, 0)
	for _, child := range n.children {
		if child.part == part || child.isWild {
			nodes = append(nodes, child)
		}
	}
	return nodes
}

```

`route.go`

```go
package gee

import (
	"net/http"
	"strings"
)

type router struct {
	roots    map[string]*node
	handlers map[string]HandlerFunc
}

func newRouter() *router {
	return &router{
		roots:    make(map[string]*node),
		handlers: make(map[string]HandlerFunc),
	}
}

// Only one * is allowed
func parsePattern(pattern string) []string {
	vs := strings.Split(pattern, "/")

	parts := make([]string, 0)
	for _, item := range vs {
		if item != "" {
			parts = append(parts, item)
			if item[0] == '*' {
				break
			}
		}
	}
	return parts
}

func (r *router) addRoute(method string, pattern string, handler HandlerFunc) {
	parts := parsePattern(pattern)

	key := method + "-" + pattern
	_, ok := r.roots[method]
	if !ok {
		r.roots[method] = &node{}
	}
	r.roots[method].insert(pattern, parts, 0)
	r.handlers[key] = handler
}

func (r *router) getRoute(method string, path string) (*node, map[string]string) {
	searchParts := parsePattern(path)
	params := make(map[string]string)
	root, ok := r.roots[method]

	if !ok {
		return nil, nil
	}

	n := root.search(searchParts, 0)

	if n != nil {
		parts := parsePattern(n.pattern)
		for index, part := range parts {
			if part[0] == ':' {
				params[part[1:]] = searchParts[index]
			}
			if part[0] == '*' && len(part) > 1 {
				params[part[1:]] = strings.Join(searchParts[index:], "/")
				break
			}
		}
		return n, params
	}

	return nil, nil
}

func (r *router) getRoutes(method string) []*node {
	root, ok := r.roots[method]
	if !ok {
		return nil
	}
	nodes := make([]*node, 0)
	root.travel(&nodes)
	return nodes
}

func (r *router) handle(c *Context) {
	n, params := r.getRoute(c.Method, c.Path)
	if n != nil {
		c.Params = params
		key := c.Method + "-" + n.pattern
		r.handlers[key](c)
	} else {
		c.String(http.StatusNotFound, "404 NOT FOUND: %s\n", c.Path)
	}
}

```

`context.go`

```go
package gee

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type H map[string]interface{}

type Context struct {
	// origin objects
	Writer http.ResponseWriter
	Req    *http.Request
	// request info
	Path   string
	Method string
	Params map[string]string
	// response info
	StatusCode int
}

func newContext(w http.ResponseWriter, req *http.Request) *Context {
	return &Context{
		Writer: w,
		Req:    req,
		Path:   req.URL.Path,
		Method: req.Method,
	}
}

func (c *Context) Param(key string) string {
	value, _ := c.Params[key]
	return value
}

func (c *Context) PostForm(key string) string {
	return c.Req.FormValue(key)
}

func (c *Context) Query(key string) string {
	return c.Req.URL.Query().Get(key)
}

func (c *Context) Status(code int) {
	c.StatusCode = code
	c.Writer.WriteHeader(code)
}

func (c *Context) SetHeader(key string, value string) {
	c.Writer.Header().Set(key, value)
}

func (c *Context) String(code int, format string, values ...interface{}) {
	c.SetHeader("Content-Type", "text/plain")
	c.Status(code)
	c.Writer.Write([]byte(fmt.Sprintf(format, values...)))
}

func (c *Context) JSON(code int, obj interface{}) {
	c.SetHeader("Content-Type", "application/json")
	c.Status(code)
	encoder := json.NewEncoder(c.Writer)
	if err := encoder.Encode(obj); err != nil {
		http.Error(c.Writer, err.Error(), 500)
	}
}

func (c *Context) Data(code int, data []byte) {
	c.Status(code)
	c.Writer.Write(data)
}

func (c *Context) HTML(code int, html string) {
	c.SetHeader("Content-Type", "text/html")
	c.Status(code)
	c.Writer.Write([]byte(html))
}

```

`gee.go`

```go
package gee

import (
	"log"
	"net/http"
)

// HandlerFunc defines the request handler used by gee
type HandlerFunc func(*Context)

// Engine implement the interface of ServeHTTP
type Engine struct {
	router *router
}

// New is the constructor of gee.Engine
func New() *Engine {
	return &Engine{router: newRouter()}
}

func (engine *Engine) addRoute(method string, pattern string, handler HandlerFunc) {
	log.Printf("Route %4s - %s", method, pattern)
	engine.router.addRoute(method, pattern, handler)
}

// GET defines the method to add GET request
func (engine *Engine) GET(pattern string, handler HandlerFunc) {
	engine.addRoute("GET", pattern, handler)
}

// POST defines the method to add POST request
func (engine *Engine) POST(pattern string, handler HandlerFunc) {
	engine.addRoute("POST", pattern, handler)
}

// Run defines the method to start a http server
func (engine *Engine) Run(addr string) (err error) {
	return http.ListenAndServe(addr, engine)
}

func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	c := newContext(w, req)
	engine.router.handle(c)
}

```

`main.go`

```go
package main

/*
(1)
$ curl -i http://localhost:9999/
HTTP/1.1 200 OK
Date: Mon, 12 Aug 2019 16:52:52 GMT
Content-Length: 18
Content-Type: text/html; charset=utf-8
<h1>Hello Gee</h1>

(2)
$ curl "http://localhost:9999/hello?name=geektutu"
hello geektutu, you're at /hello

(3)
$ curl "http://localhost:9999/hello/geektutu"
hello geektutu, you're at /hello/geektutu

(4)
$ curl "http://localhost:9999/assets/css/geektutu.css"
{"filepath":"css/geektutu.css"}

(5)
$ curl "http://localhost:9999/xxx"
404 NOT FOUND: /xxx
*/

import (
	"net/http"

	"./gee"
)

func main() {
	r := gee.New()
	r.GET("/", func(c *gee.Context) {
		c.HTML(http.StatusOK, "<h1>Hello Gee</h1>")
	})

	r.GET("/hello", func(c *gee.Context) {
		// expect /hello?name=geektutu
		c.String(http.StatusOK, "hello %s, you're at %s\n", c.Query("name"), c.Path)
	})

	r.GET("/hello/:name", func(c *gee.Context) {
		// expect /hello/geektutu
		c.String(http.StatusOK, "hello %s, you're at %s\n", c.Param("name"), c.Path)
	})

	r.GET("/assets/*filepath", func(c *gee.Context) {
		c.JSON(http.StatusOK, gee.H{"filepath": c.Param("filepath")})
	})

	r.Run(":9999")
}

```

