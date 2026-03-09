package web

import (
	"net/http"
	"time"

	"github.com/PlakarKorp/docbot/bot/db"
)

type Server struct {
	queries *db.Queries
	http    *http.Server
}

func New(addr string, queries *db.Queries) *Server {
	s := &Server{queries: queries}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /pages", s.handlePages)
	mux.HandleFunc("GET /reviewers", s.handleReviewers)

	s.http = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return s
}

func (s *Server) Start() error {
	return s.http.ListenAndServe()
}
