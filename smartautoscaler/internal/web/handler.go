package web

import (
	"context"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
)

// DataProvider — то, что ваш контроллер должен реализовать,
// чтобы отдать данные для веб-страницы.
type DataProvider interface {
	GetGraph(ctx context.Context) []ServiceNode
	GetHistograms(ctx context.Context) []ServiceHistogram
}

type Server struct {
	provider DataProvider
	tmpl     *template.Template

	NAMESPACE string
}

func NewServer(provider DataProvider) (*Server, error) {
	tmpl, err := template.New("dashboard.html").
		Funcs(template.FuncMap{
			// toJSON — безопасная вставка JSON в <script>.
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
	return &Server{provider: provider, tmpl: tmpl}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.dashboard)
	return mux
}

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	data := PageData{
		Graph:      s.provider.GetGraph(r.Context()),
		Histograms: s.provider.GetHistograms(r.Context()),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "dashboard.html", data); err != nil {
		log.Printf("template error: %v", err)
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}
