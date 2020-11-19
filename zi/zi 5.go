package main

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"golang.org/x/net/html"
)

const domain = "https://www.artlebedev.ru"
const mainUrl = domain + "/news/2020"

func getAttribute(node *html.Node, name string) (string, error) {
	for _, attr := range node.Attr {
		if attr.Key == name {
			return attr.Val, nil
		}
	}
	return "", errors.New("No such attribute")
}

func getClassNames(node *html.Node) []string {
	val, err := getAttribute(node, "class")
	if err != nil {
		return nil
	}
	return strings.Split(val, " ")
}

func hasClassName(node *html.Node, name string) bool {
	names := getClassNames(node)
	for _, n := range names {
		if n == name {
			return true
		}
	}
	return false
}

func getElementsByClassName(node *html.Node, name string) []*html.Node {
	var nodes []*html.Node
	if hasClassName(node, name) {
		nodes = append(nodes, node)
	}
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		cn := getElementsByClassName(c, name)
		nodes = append(nodes, cn...)
	}
	return nodes
}

type item struct {
	Title string
	Date string
	Link string
}

type article struct {
	Title string
	Content string
}

func parseMain(document *html.Node) []item {
	blocks := getElementsByClassName(document, "item")
	var res []item
	for _, b := range blocks {
		// дата
		date := "no_date"
		ctx := b.FirstChild.FirstChild
		if len(getElementsByClassName(b, "date")) > 0 {
			ctx = getElementsByClassName(b, "date")[0]
			date = ctx.FirstChild.Data
			ctx = ctx.NextSibling
		}
		//fmt.Println("\t", date)

		// откр. ковычка, название
		title := ctx.Data // начало (или все) названия
		ctx = ctx.NextSibling

		// дальше название, ссылка
		link := "no_link"
		if ctx != nil && ctx.Data == "a" {
			title += ctx.FirstChild.Data // середина названия
			link, _ = getAttribute(ctx, "href")

			// закр. ковычка
			title += ctx.NextSibling.Data // конец названия
		}
		//fmt.Println(link)
		//title = "\b" + title + "\b" // убираем лишние отступы
		//fmt.Println(title)

		i := item{
			Title: title,
			Date: date,
			Link: link,
		}
		res = append(res, i)
	}
	return res
}

func parseBlock(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	res := ""
	for c := n.FirstChild; c != nil; c = c.NextSibling { // с - текцщий ребенок n
		res += parseBlock(c)
	}
	return res
}

func parseArticle(document *html.Node) article {
	// парсинг названия
	title := getElementsByClassName(document, "als-text-title")[0].FirstChild.FirstChild.Data

	mainBlock := getElementsByClassName(document, "without-cover")[0]

	return article{
		Title: title,
		Content: parseBlock(mainBlock),
	}
}

var mainTemplate, articleTemplate string

func main() {
	t, _ := ioutil.ReadFile("zi/mainTemplate.html")
	mainTemplate = string(t)
	t, _ = ioutil.ReadFile("zi/articleTemplate.html")
	articleTemplate = string(t)

	http.HandleFunc("/favicon.ico", func(rw http.ResponseWriter, r *http.Request) {
		fav, _ := ioutil.ReadFile("zi/favicon.ico")
		rw.Write(fav)
	})

	http.HandleFunc("/p/", func(rw http.ResponseWriter, r *http.Request) {
		// r.URL.Path - все после домена (начиная с первого слеша)
		url := r.URL.Path[2:] // - начиная с 3 символа (удаляем /p/)
		fmt.Println("move to", url)
		p, _ := http.Get(domain + url)
		document, _ := html.Parse(p.Body)
		article := parseArticle(document)
		t, _ := template.New("").Parse(articleTemplate)
		rw.WriteHeader(200)
		b := bytes.NewBufferString("")
		t.Execute(b, map[string] interface{} {
			"article": article,
		})
		_, err := rw.Write(b.Bytes())
		if err != nil {
			log.Println(err.Error())
		}
	})

	http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		p, _ := http.Get(mainUrl)
		document, _ := html.Parse(p.Body)
		items := parseMain(document)
		t, _ := template.New("").Parse(mainTemplate)
		rw.WriteHeader(200)
		b := bytes.NewBufferString("")
		err := t.Execute(b, map[string] interface{} {
			"items": items,
		})
		if err != nil {
			log.Println(err.Error())
		}
		_, err = rw.Write(b.Bytes())
		if err != nil {
			log.Println(err.Error())
		}
		fmt.Println("Finish\n\n")
	})


	fmt.Println("Server is listening...")
	http.ListenAndServe(":7777", nil)
}