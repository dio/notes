package main

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/yuin/goldmark"
)

//go:embed index.html
var indexHTML []byte

//go:embed notes.html
var notesTemplateSource string

//go:embed notes
var notesFS embed.FS

var notesTemplate = template.Must(template.New("notes").Parse(notesTemplateSource))

var notesIndexTemplate = template.Must(template.New("notes-index").Parse(`<h1>Notes</h1>
<p>Technical things worth keeping.</p>
<h2>Envoy</h2>
<ul>
{{range .}}  <li><a href="{{.URL}}">{{.Title}}</a>{{if .Repository}} — <a href="{{.Repository}}">source repository</a>{{end}}</li>
{{end}}</ul>`))

type noteDefinition struct {
	Title      string
	URL        string
	Source     string
	Repository string
}

var publishedNotes = []noteDefinition{
	{
		Title:  "ADS DiscoveryRequest.version_info across stream reconnects",
		URL:    "/notes/envoy/ads-discovery-request-version-info",
		Source: "notes/envoy/ads-discovery-request-version-info.md",
	},
	{
		Title:      "One Route, One Cluster, Many Providers",
		URL:        "/notes/envoy/one-route-one-cluster-many-providers",
		Source:     "notes/envoy/one-route-one-cluster-many-providers.md",
		Repository: "https://github.com/dio/envoy-one-cluster-many-providers",
	},
	{
		Title:  "Current Envoy prototypes and issue reproductions",
		URL:    "/notes/envoy-prototypes-and-issues",
		Source: "notes/envoy-prototypes-and-issues.md",
	},
	{
		Title:  "Envoy project archive",
		URL:    "/notes/envoy/projects",
		Source: "notes/envoy/projects.md",
	},
}

type notesPage struct {
	Title   string
	Content template.HTML
}

func newHandler() http.Handler {
	renderedNotes := loadNotes()
	notesIndex := renderNotesIndex()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		requestPath := strings.TrimSuffix(r.URL.Path, "/")
		if requestPath == "" {
			requestPath = "/"
		}

		switch requestPath {
		case "/":
			writeHTML(w, indexHTML)
		case "/notes":
			writeNotesPage(w, notesPage{
				Title:   "Notes",
				Content: notesIndex,
			})
		default:
			page, ok := renderedNotes[requestPath]
			if !ok {
				http.NotFound(w, r)
				return
			}
			writeNotesPage(w, page)
		}
	})
}

func loadNotes() map[string]notesPage {
	renderedNotes := make(map[string]notesPage, len(publishedNotes))
	for _, note := range publishedNotes {
		source, err := notesFS.ReadFile(note.Source)
		if err != nil {
			panic(fmt.Sprintf("read %s: %v", note.Source, err))
		}
		renderedNotes[note.URL] = notesPage{
			Title:   note.Title,
			Content: renderMarkdown(source),
		}
	}
	return renderedNotes
}

func renderNotesIndex() template.HTML {
	var rendered bytes.Buffer
	if err := notesIndexTemplate.Execute(&rendered, publishedNotes); err != nil {
		panic(fmt.Sprintf("render notes index: %v", err))
	}
	return template.HTML(rendered.String())
}

func renderMarkdown(source []byte) template.HTML {
	var rendered bytes.Buffer
	if err := goldmark.Convert(source, &rendered); err != nil {
		panic(fmt.Sprintf("render Markdown: %v", err))
	}
	return template.HTML(rendered.String())
}

func writeHTML(w http.ResponseWriter, content []byte) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(content)
}

func writeNotesPage(w http.ResponseWriter, page notesPage) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := notesTemplate.Execute(w, page); err != nil {
		log.Printf("render notes page: %v", err)
	}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	address := fmt.Sprintf(":%s", port)
	log.Printf("listening on %s", address)
	log.Fatal(http.ListenAndServe(address, newHandler()))
}
