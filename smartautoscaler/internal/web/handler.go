package web

import (
	"context"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
)

type DataProvider interface {
	GetGraph(ctx context.Context, namespace string) []ServiceNode
	GetHistograms(ctx context.Context, namespace string) []ServiceHistogram
}

type Server struct {
	provider  DataProvider
	namespace string
	tmpl      *template.Template
}

func NewServer(provider DataProvider, namespace string) (*Server, error) {
	tmpl, err := template.New("dashboard.html").
		Funcs(template.FuncMap{
			"toJSON": func(v any) (template.JS, error) {
				b, err := json.Marshal(v)
				if err != nil {
					return "", err
				}
				return template.JS(b), nil
			},
		}).
		ParseFiles("/app/web/templates/dashboard.html")
	if err != nil {
		return nil, err
	}
	return &Server{provider: provider, namespace: namespace, tmpl: tmpl}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.dashboard)
	return mux
}

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	data := PageData{
		Graph:      s.provider.GetGraph(ctx, s.namespace),
		Histograms: s.provider.GetHistograms(ctx, s.namespace),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "dashboard.html", data); err != nil {
		log.Printf("template error: %v", err)
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}
