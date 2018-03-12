package main

import (
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gopkg.in/mgo.v2"
	// "gopkg.in/mgo.v2/bson"
)

const (
	pagenum    = 10
	timeLayout = "2006-01-02 15:04:05"
)

type Weibo struct {
	Author  string   `json:"author"`
	URL     string   `json:"url"`
	Text    string   `json:"text"`
	Addtime int64    `json:"addtime"`
	Pics    []string `json:"pics"`
	At_By   string   `json:"at_by"`
	At_Time int64    `json:"at_time"`
}


func (wb *Weibo) FormatTime(i int64) string {
	tm := time.Unix(i, 0)
	return tm.Format(timeLayout)
}

func (wb *Weibo) Sync(session *mgo.Session) {
	
}

type HTML struct {
	Weibos   *[]Weibo
	Page     int
	LastPage int
	NextPage int
	AllPage  int
	PageNum  int
}

func (html *HTML) PageNew(session *mgo.Session, page int, pagenum int) {
	weibos := make([]Weibo, pagenum)
	c := session.DB("sinaweibo").C("weibos")
	count, err := c.Find(nil).Count()
	if err != nil {
		panic(err)
	}
	allpage := int(math.Ceil(float64(count) / float64(pagenum)))
	if page > allpage {
		page = allpage
	}
	err = c.Find(nil).Sort("-at_time").Skip((page - 1) * pagenum).Limit(pagenum).All(&weibos)
	if err != nil {
		panic(err)
	}
	lastpage := -1
	if page > 1 {
		lastpage = page - 1
	}
	nextpage := -1
	if page*pagenum < count {
		nextpage = page + 1
	}
	html.Weibos = &weibos
	html.Page = page
	html.PageNum = pagenum
	html.LastPage = lastpage
	html.NextPage = nextpage
	html.AllPage = allpage	
}



func Homepage(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/page/1", http.StatusFound)
}


func WeiboList(w http.ResponseWriter, r *http.Request) {
	path := strings.Split(r.URL.Path, "/")
	page, err := strconv.Atoi(path[len(path)-1])
	if err != nil || page <= 0 {
		http.Redirect(w, r, "/page/1", http.StatusFound)
		return
	}
	html := new(HTML)
	session, err := mgo.Dial("115.28.137.182:27017")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	html.PageNew(session, page, pagenum)

	fmt.Println(path[len(path)-1])
	fmt.Printf("%+v\n", html)

	t, err := template.New("weibo.html").ParseFiles("templates/weibo.html")
	err = t.Execute(w, html)
	if err != nil {
		fmt.Println(err)
	}
}

func main() {
	http.HandleFunc("/", Homepage)
	http.HandleFunc("/page/", WeiboList)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	err := http.ListenAndServe(":9090", nil) //设置监听的端口
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
