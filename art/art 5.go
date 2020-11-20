package main

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

	"golang.org/x/net/html"
)

const domain = "https://life.ru"

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

func getElementsByType(node *html.Node, name string) []*html.Node {
	var nodes []*html.Node
	if node.Data == name {
		nodes = append(nodes, node)
	}
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		cn := getElementsByType(c, name)
		nodes = append(nodes, cn...)
	}
	return nodes
}

type item struct {
	Title string
	Preview *preview
	HasPreview bool
	Date string
	Link string
}

type article struct {
	Title string
	Content string
}

type preview struct {
	Url string
	IsVideo bool
}

func parsePreview(node *html.Node) *preview {
	imgs := getElementsByType(node, "img")
	videos := getElementsByType(node, "video")
	if len(imgs) > 0 {
		src, _ := getAttribute(imgs[0], "src")
		return &preview{
			Url: src,
			IsVideo: false,
		}
	}
	src, _ := getAttribute(videos[0], "src")
	return &preview{
		Url: src,
		IsVideo: true,
	}
}

func parseMain(document *html.Node) []item {
	blocks := getElementsByClassName(document, "styles_root__2aHN8")
	var res []item
	for _, b := range blocks {
		previewBlock := getElementsByClassName(b, "styles_imgWrapper__3XFTR")
		fmt.Println(previewBlock)
		var prev *preview
		if len(previewBlock) > 0 {
			prev = parsePreview(previewBlock[0])
		}
		headerBlock := getElementsByClassName(b, "styles_title__VjSwt")[0]
		dateBlock := getElementsByClassName(b, "styles_date__1zS9H")[0]
		link, _ := getAttribute(b, "href")
		i := item{
			Title: headerBlock.FirstChild.FirstChild.Data,
			Date: dateBlock.FirstChild.Data,
			Preview: prev,
			HasPreview: prev != nil,
			Link: link,
		}
		res = append(res, i)
	}
	return res
}

func parseBlock(n *html.Node) string {
	if n.FirstChild == nil {
		return n.Data
	}
	res := ""
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		res += parseBlock(c)
	}
	return res
}

func parseArticle(document *html.Node) article {
	blocks := getElementsByClassName(document, "styles_text__fxCxY") // этот класс только внутри p, а в них нет картинок, поэтому firChild == nil только у текста
	cur := ""
	for _, b := range blocks {
		cur += parseBlock(b)
	}
	titleBlock := getElementsByClassName(document, "styles_title__2F4Y1")[0]
	return article{
		Content: cur,
		Title: titleBlock.FirstChild.Data,
	}
}

const mainTemplate = `<!DOCTYPE html>
<html>
	<head>
		<title>Life.ru</title>
		<meta charset="utf-8" />
	</head>
	<body>
		<h1>Life.ru</h1>
		{{ range .items }}
			{{ if .HasPreview }}
				{{ if .Preview.IsVideo }}
					<h2> <a href="{{ .Link }}?video={{ .Preview.Url }}"> {{ .Title }} </a> </h2>
					<video src="{{ .Preview.Url}}" autoplay loop></video>
				{{ else }}
					<h2> <a href="{{ .Link }}?img={{ .Preview.Url }}"> {{ .Title }} </a> </h2>
					<img src="{{ .Preview.Url}}" />
				{{ end }}
			{{ else }}
					<h2> <a href="{{ .Link }}"> {{ .Title }} </a> </h2>
			{{ end }}
			<br />
			{{ .Date }}
		{{ end }}
	</body>
</html>
`

const articleTemplate = `<!DOCTYPE html>
<html>
	<head>
		<title>{{ .article.Title }}</title>
		<meta charset="utf-8" />
	</head>
	<body>
		<h1>{{ .article.Title }}</h1>
		{{ if .hasImage }}
			<img src="{{ .image }}" /> <br />
		{{ else if .hasVideo }}
			<video src="{{ .video }}" autoplay loop /></video> <br />
		{{ end }}
		{{ .article.Content | html }}
	</body>
</html>
`

func main() {
	http.HandleFunc("/p/", func(rw http.ResponseWriter, r *http.Request) {
		p, _ := http.Get(domain + r.URL.Path)
		document, _ := html.Parse(p.Body)
		article := parseArticle(document)
		t, _ := template.New("").Parse(articleTemplate)
		rw.WriteHeader(200)
		b := bytes.NewBufferString("")
		imgUrl := r.URL.Query().Get("img")
		videoUrl := r.URL.Query().Get("video")
		t.Execute(b, map[string] interface{} {
			"article": article,
			"hasImage": imgUrl != "",
			"image": imgUrl,
			"hasVideo": videoUrl != "",
			"video": videoUrl,
		})
		_, err := rw.Write(b.Bytes())
		if err != nil {
			log.Println(err.Error())
		}
	})

	http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		p, _ := http.Get(domain)
		document, _ := html.Parse(p.Body)
		items := parseMain(document)
		/*for n, i := range items {
			fmt.Print(n, ")\t")
			fmt.Println(i.Date)
		}*/
		t, _ := template.New("").Parse(mainTemplate)
		rw.WriteHeader(200)
		b := bytes.NewBufferString("")
		t.Execute(b, map[string] interface{} {
			"items": items,
		})
		_, err := rw.Write(b.Bytes())
		if err != nil {
			log.Println(err.Error())
		}
		fmt.Println()
	})

	http.ListenAndServe(":7001", nil)
}