package main

import (
	"net/http"
	"strconv"
	"text/template"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

const (
	master = `
	<html><head><meta http-equiv="refresh" content="300" />
	<title>{{.Name}}</title></head><body>
	<h1>{{.Name}}</h1>
	{{$name := .Name}}
	{{range $i, $w := .Body.Widgets}}
		{{ if $w.HasMarkdown }}
			{{$w.Markdown}}
		{{else}}
			<img src="/widget/{{$name}}/{{$i}}" />
		{{end}}
	{{end}}
	</body></html>
	`
	index = `
	<html><head><title>Dashboards</title></head><body><ul>
	<h1>Dashboards</h1>
	{{range .}}
		<li><a href="/dashboard/{{.}}">{{.}}</a></li>
	{{end}}</ul>
	</body></html>
	`
)

func main() {

	masterTmpl, err := template.New("master").Parse(master)
	indexTmpl, err := template.New("index").Parse(index)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	imgGen, err := newImageGenerator()
	if err != nil {
		panic("can't create image generator: " + err.Error())
	}
	err = imgGen.refreshDashboardList()
	if err != nil {
		panic("can't refresh dashboard list: " + err.Error())
	}

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		if err := indexTmpl.Execute(w, imgGen.dashboardList); err != nil {
			panic(err)
		}
	})

	r.Get("/widget/{name:[A-Za-z0-9\\-]+}/{number:\\d+}", func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		number := chi.URLParam(r, "number")
		index, _ := strconv.Atoi(number)
		imgBody, err := imgGen.renderWidget(name, index)
		if err != nil {
			panic("unable to render image." + err.Error())
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "max-age=300") // 30 days
		w.Header().Set("Content-Length", strconv.Itoa(len(imgBody)))
		if _, err := w.Write(imgBody); err != nil {
			panic("unable to write image." + err.Error())
		}
		w.Write([]byte(number))
	})

	r.Get("/dashboard/{name:[A-Za-z0-9\\-]+}", func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		if imgGen.dashboards[name] == nil {
			if err := imgGen.refreshBody(name); err != nil {
				http.Error(w, http.StatusText(404), 404)
				return
			}
		}

		w.Header().Set("Content-Type", "text/html")
		if err := masterTmpl.Execute(w, struct {
			Name string
			Body *dashboardBody
		}{name, imgGen.dashboards[name]}); err != nil {
			panic(err)
		}
	})

	http.ListenAndServe(":8000", r)
}
