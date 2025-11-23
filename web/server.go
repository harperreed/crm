// ABOUTME: Web UI server with embedded templates
// ABOUTME: Provides read-only dashboard at localhost:8080
package web

import (
	"database/sql"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/harperreed/pagen/viz"
)

//go:embed templates/*
var templatesFS embed.FS

type Server struct {
	db        *sql.DB
	templates *template.Template
	generator *viz.GraphGenerator
}

func NewServer(database *sql.DB) (*Server, error) {
	// Helper functions for templates
	funcMap := template.FuncMap{
		"divide": func(a, b int64) int64 {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"multiply": func(a, b int) int {
			return a * b
		},
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	return &Server{
		db:        database,
		templates: tmpl,
		generator: viz.NewGraphGenerator(database),
	}, nil
}

func (s *Server) Start(port int) error {
	// Routes
	http.HandleFunc("/", s.handleDashboard)
	http.HandleFunc("/contacts", s.handleContacts)
	http.HandleFunc("/companies", s.handleCompanies)
	http.HandleFunc("/deals", s.handleDeals)
	http.HandleFunc("/graphs", s.handleGraphs)

	// Partials for HTMX
	http.HandleFunc("/partials/contact-detail", s.handleContactDetail)
	http.HandleFunc("/partials/company-detail", s.handleCompanyDetail)
	http.HandleFunc("/partials/deal-detail", s.handleDealDetail)
	http.HandleFunc("/partials/graph", s.handleGraphPartial)

	addr := fmt.Sprintf(":%d", port)
	log.Printf("Starting web server at http://localhost%s", addr)
	return http.ListenAndServe(addr, nil)
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	stats, err := viz.GenerateDashboardStats(s.db)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Stats": stats,
		"Title": "Dashboard",
	}

	s.renderTemplate(w, "dashboard.html", data)
}

func (s *Server) renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	err := s.templates.ExecuteTemplate(w, name, data)
	if err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Stub handlers - to be implemented in later tasks
func (s *Server) handleContacts(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

func (s *Server) handleCompanies(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

func (s *Server) handleDeals(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

func (s *Server) handleGraphs(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

func (s *Server) handleContactDetail(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

func (s *Server) handleCompanyDetail(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

func (s *Server) handleDealDetail(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

func (s *Server) handleGraphPartial(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}
