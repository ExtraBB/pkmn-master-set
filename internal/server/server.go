package server

import (
	"html/template"
	"log"
	"net/http"

	"github.com/ExtraBB/pkmn-master-set/web"
)

// Server holds everything the HTTP handlers need.
type Server struct {
	tmpl *template.Template
}

func New() *Server {
	return &Server{tmpl: template.Must(template.ParseFS(web.Templates, "templates/*.html"))}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /static/", http.FileServerFS(web.Static))
	mux.HandleFunc("GET /{$}", s.handleIndex)
	mux.HandleFunc("POST /ping", s.handlePing)

	return mux
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	s.render(w, "index.html", nil)
}

// handlePing returns an HTML fragment that htmx swaps into the page.
func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	s.render(w, "ping.html", map[string]any{"Message": "pong"})
}

func (s *Server) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("render %s: %v", name, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}
