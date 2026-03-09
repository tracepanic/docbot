package web

import (
	"html/template"
	"log"
	"net/http"
)

type PageRow struct {
	ID         int64
	Title      string
	URL        string
	LastReview string
	Status     string
}

type PagesData struct {
	Rows []PageRow
}

type ReviewerRow struct {
	Username     string
	LastAssigned string
	Status       string
}

type ReviewersData struct {
	Rows []ReviewerRow
}

var templates map[string]*template.Template

func init() {
	templates = make(map[string]*template.Template)
	templates["pages"] = template.Must(
		template.ParseFS(templateFS, "templates/layout.html", "templates/pages.html"),
	)
	templates["reviewers"] = template.Must(
		template.ParseFS(templateFS, "templates/layout.html", "templates/reviewers.html"),
	)
}

func (s *Server) handlePages(w http.ResponseWriter, r *http.Request) {
	docs, err := s.queries.ListAllDocuments(r.Context())
	if err != nil {
		log.Printf("Failed to list documents: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	rows := make([]PageRow, len(docs))
	for i, d := range docs {
		status := "active"
		if !d.Active {
			status = "inactive"
		}
		lastReview := ""
		if d.LastReviewed.Valid {
			lastReview = d.LastReviewed.Time.Format("2006-01-02")
		}
		rows[i] = PageRow{
			ID:         d.ID,
			Title:      d.Title,
			URL:        d.Url,
			LastReview: lastReview,
			Status:     status,
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates["pages"].ExecuteTemplate(w, "layout", PagesData{Rows: rows}); err != nil {
		log.Printf("Failed to render pages template: %v", err)
	}
}

func (s *Server) handleReviewers(w http.ResponseWriter, r *http.Request) {
	reviewers, err := s.queries.ListAllReviewers(r.Context())
	if err != nil {
		log.Printf("Failed to list reviewers: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	rows := make([]ReviewerRow, len(reviewers))
	for i, rv := range reviewers {
		status := "active"
		if !rv.Active {
			status = "inactive"
		}
		lastAssigned := ""
		if rv.LastAssigned.Valid {
			lastAssigned = rv.LastAssigned.Time.Format("2006-01-02")
		}
		rows[i] = ReviewerRow{
			Username:     rv.Username,
			LastAssigned: lastAssigned,
			Status:       status,
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates["reviewers"].ExecuteTemplate(w, "layout", ReviewersData{Rows: rows}); err != nil {
		log.Printf("Failed to render reviewers template: %v", err)
	}
}
