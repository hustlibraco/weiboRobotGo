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
	"io/ioutil"
	"os"

	"base62"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"github.com/json-iterator/go"
	"gopkg.in/ini.v1"
)

const (
	pagenum    = 10
	timeLayout = "2006-01-02 15:04:05"
	timeParseFmt = "Mon Jan 02 15:04:05 -0700 2006"
	weiboApi   = "https://api.weibo.com"
	N = 10000000	// 约等于64^4 ？？	
)

// secret
var (
	weiboApiKey string
	username string
	password string
	mongoURI string
	webURI string
)

type Weibo struct {
	Id		int64	 `json:"id"`
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

func Strptime(s string) int64 {
	t, _ := time.Parse(timeParseFmt, s)
	return t.Unix()
}

func MakeURL(uid, mid int64) string {
	s := ""
	for mid > 0 {
		s = base62.Encode(int(mid % N)) + s
		mid /= N
	}
    return fmt.Sprintf("https://weibo.com/%v/%v", uid, s)
}

func MakePicURL(pid string) string {
	return fmt.Sprintf("http://ww1.sinaimg.cn/thumbnail/%v.jpg", pid)
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

func SyncWeibo(interval int) {
	session, err := mgo.Dial(mongoURI)
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()
	session.SetMode(mgo.Monotonic, true)
	c := session.DB("sinaweibo").C("weibos")
	client := &http.Client{}
	url := fmt.Sprintf("%v/2/comments/mentions.json?source=%v", weiboApi, weiboApiKey)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.SetBasicAuth(username, password)
	for {
		resp, err := client.Do(req)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		size := jsoniter.Get(body, "comments").Size()
		for i := 0; i < size; i++ {
			comment := jsoniter.Get(body, "comments", i)
			at_by := comment.Get("user", "name").ToString()
			status := comment.Get("status").MustBeValid()
			t := status.Get("retweeted_status")
			if t.Size() == 0 {
				t = status
			}
			if t.Get("deleted").ToInt() == 1 {
				continue
			}
			id := t.Get("id").MustBeValid().ToInt64()
			count, err := c.Find(bson.M{"id": id}).Count()
			if err != nil {
				log.Fatal(err)
				continue
			}
			if count > 0 {
				continue
			}
			log.Printf("new weibo: %v", id)
			uid := t.Get("user", "id").ToInt64()
			mid := t.Get("mid").ToInt64()
			text := t.Get("text").ToString()
			author := t.Get("user", "name").ToString()
			addtime := t.Get("created_at").ToString()
			pic_size := t.Get("pic_ids").Size()
			pics := make([]string, 0)
			for i := 0; i < pic_size; i++ {
				pic_id := t.Get("pic_ids", i).ToString()
				pics = append(pics, MakePicURL(pic_id))
			}
			at_time := comment.Get("created_at").ToString()
			weibo := Weibo{}
			weibo.Id = id
			weibo.Author = author
			weibo.URL = MakeURL(uid, mid)
			weibo.Text = text
			weibo.Addtime = Strptime(addtime)
			weibo.Pics = pics
			weibo.At_By = at_by
			weibo.At_Time = Strptime(at_time)
			err = c.Insert(&weibo)
			if err != nil {
				log.Fatal(err)
			}
		}
		log.Printf("sync weibo sleep %v seconds", interval)
		time.Sleep(time.Duration(interval) * time.Second)
	}
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
	session, err := mgo.Dial(mongoURI)
	if err != nil {
		panic(err)
	}
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	html.PageNew(session, page, pagenum)
	t, err := template.New("weibo.html").ParseFiles("templates/weibo.html")
	err = t.Execute(w, html)
	if err != nil {
		fmt.Println(err)
	}
}

func init() {
	cfg, err := ini.Load("config.ini")
    if err != nil {
        log.Printf("Fail to read file: %v", err)
        os.Exit(1)
	}
	weiboApiKey = cfg.Section("weibo").Key("api_key").String()
	username = cfg.Section("weibo").Key("username").String()
	password = cfg.Section("weibo").Key("password").String()
	mongoURI = fmt.Sprintf(
		"%v:%v", 
		cfg.Section("mongo").Key("host").String(), 
		cfg.Section("mongo").Key("port").String(),
	)
	webURI = fmt.Sprintf(
		"%v:%v", 
		cfg.Section("web").Key("host").String(), 
		cfg.Section("web").Key("port").String(),
	)
}

func main() {
	// 同步微博
	go SyncWeibo(3600)
	// web服务
	http.HandleFunc("/", Homepage)
	http.HandleFunc("/page/", WeiboList)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	err := http.ListenAndServe(webURI, nil) //设置监听的端口
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
