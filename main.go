package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/yuin/goldmark"
)

//go:embed index.html
var indexHTML []byte

//go:embed notes.html
var notesTemplateSource string

//go:embed notes/envoy/ads-discovery-request-version-info.md
var adsVersionInfoMarkdown []byte

var notesTemplate = template.Must(template.New("notes").Parse(notesTemplateSource))

type notesPage struct {
	Title   string
	Content template.HTML
}

func newHandler() http.Handler {
	adsVersionInfoHTML := renderMarkdown(adsVersionInfoMarkdown)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		switch r.URL.Path {
		case "/":
			writeHTML(w, indexHTML)
		case "/notes", "/notes/":
			writeNotesPage(w, notesPage{
				Title: "Notes",
				Content: template.HTML(`<h1>Notes</h1>
<p>Technical things worth keeping.</p>
<h2>Envoy</h2>
<ul>
  <li><a href="/notes/envoy/ads-discovery-request-version-info">ADS DiscoveryRequest.version_info across stream reconnects</a></li>
</ul>`),
			})
		case "/notes/envoy/ads-discovery-request-version-info",
			"/notes/envoy/ads-discovery-request-version-info/":
			writeNotesPage(w, notesPage{
				Title:   "ADS DiscoveryRequest.version_info across stream reconnects",
				Content: adsVersionInfoHTML,
			})
		default:
			http.NotFound(w, r)
		}
	})
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
