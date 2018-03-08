package main

import (
	"fmt"
	"net/http"
	"html/template"
	// "strconv"
	// "fmt"
	// "strings"
	"log"
)

type Weibo struct {
	Author string
	Url string
	Text string
	Addtime string
	Pics []string
	At_by string
	At_time string
}

type HomePage struct {
	Weibos []*Weibo
	Last int
	Next int
}
var weibo = Weibo{
	"Libraco", 
	"http://weibo.com/1618051664/G56vivVKx", 
	"今天三八妇女节",
	"2018-02-28 10:39:49",
	[]string{},
	"Liu Han",
	"2018-02-28 10:40:06",	
}
var homepage = HomePage{
	[]*Weibo {&weibo, &weibo, &weibo}, 1, 10,
}
func sayhelloName(w http.ResponseWriter, r *http.Request) {
	// pagenum := 20
	// paths := strings.Split(r.URL.Path, "/")
	// rawpage := paths[len(paths) - 1]
	// page, err := strconv.Atoi(rawpage)
	// if err != nil {
	// 	fmt.Fprintf(w, "invalid page: %v", rawpage)
	// 	return
	// }
	t, err := template.New("weibo.html").ParseFiles("templates/weibo.html")
	err = t.Execute(w, homepage)
	if err != nil {
		fmt.Println(err)
	}
	// r.ParseForm()  //解析参数，默认是不会解析的
	// fmt.Println(r.Form)  //这些信息是输出到服务器端的打印信息
	// fmt.Println("path", r.URL.Path)
	// fmt.Println("scheme", r.URL.Scheme)
	// fmt.Println(r.Form["url_long"])
	// for k, v := range r.Form {
	// 	fmt.Println("key:", k)
	// 	fmt.Println("val:", strings.Join(v, ""))
	// }
	// fmt.Fprintf(w, "Hello astaxie!") //这个写入到w的是输出到客户端的
}

func main() {
	http.HandleFunc("/", sayhelloName) //设置访问的路由
	err := http.ListenAndServe(":9090", nil) //设置监听的端口
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}