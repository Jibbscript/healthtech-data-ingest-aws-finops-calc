package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jibbscript/throne-backend-poc/internal/platform/localaws"
	"github.com/jibbscript/throne-backend-poc/internal/store"
)

type Server struct {
	Store  *store.Store
	Raw    *localaws.BlobStore
	Secret string
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	protected := http.NewServeMux()
	protected.HandleFunc("/v1/users/", s.listCaptures)
	protected.HandleFunc("/v1/captures/", s.captureRoutes)
	protected.HandleFunc("/v1/findings/", s.getFinding)
	mux.Handle("/v1/", AuthMiddleware(s.secret(), protected))
	return mux
}

func (s *Server) listCaptures(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 4 || parts[0] != "v1" || parts[1] != "users" || parts[3] != "captures" {
		http.NotFound(w, r)
		return
	}
	userID := parts[2]
	if UserID(r.Context()) != userID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	items, next, err := s.Store.ListCapturesByUser(userID, limit, r.URL.Query().Get("cursor"))
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, map[string]any{"captures": items, "next_cursor": next})
}

func (s *Server) captureRoutes(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		http.NotFound(w, r)
		return
	}
	id := parts[2]
	if len(parts) == 3 && r.Method == http.MethodGet {
		s.getCapture(w, r, id)
		return
	}
	if len(parts) == 4 && parts[3] == "image-url" && r.Method == http.MethodPost {
		s.imageURL(w, r, id)
		return
	}
	http.NotFound(w, r)
}

func (s *Server) getCapture(w http.ResponseWriter, r *http.Request, id string) {
	c, err := s.Store.GetCapture(id)
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if c.UserID != UserID(r.Context()) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	writeJSON(w, map[string]any{"capture": c})
}

func (s *Server) imageURL(w http.ResponseWriter, r *http.Request, id string) {
	c, err := s.Store.GetCapture(id)
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if c.UserID != UserID(r.Context()) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	writeJSON(w, map[string]any{"url": s.Raw.Presign(c.S3Key, 5*time.Minute), "expires_in_seconds": 300})
}

func (s *Server) getFinding(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 3 {
		http.NotFound(w, r)
		return
	}
	f, err := s.Store.GetFinding(parts[2])
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if f.UserID != UserID(r.Context()) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	writeJSON(w, map[string]any{"finding": f})
}

func (s *Server) secret() string {
	if s.Secret == "" {
		return "dev-secret"
	}
	return s.Secret
}
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
