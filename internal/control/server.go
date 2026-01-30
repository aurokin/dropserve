package control

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"
)

type Server struct {
	store  *Store
	logger *log.Logger
}

type errorResponse struct {
	Error string `json:"error"`
}

type requestIDKey struct{}

func NewServer(store *Store, logger *log.Logger) *Server {
	return &Server{store: store, logger: logger}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/control/health", s.handleHealth)
	mux.HandleFunc("/api/control/portals", s.handleCreatePortal)
	return s.withRequestID(mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleCreatePortal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	var req CreatePortalRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json"})
		return
	}

	if strings.TrimSpace(req.DestAbs) == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "dest_abs required"})
		return
	}

	policy, err := normalizePolicy(req.DefaultPolicy)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	portal, err := s.store.CreatePortal(CreatePortalInput{
		DestAbs:              req.DestAbs,
		OpenMinutes:          req.OpenMinutes,
		Reusable:             req.Reusable,
		DefaultPolicy:        policy,
		AutorenameOnConflict: req.AutorenameOnConflict,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to create portal"})
		return
	}

	resp := CreatePortalResponse{
		PortalID:  portal.ID,
		ExpiresAt: portal.OpenUntil.Format(time.RFC3339),
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-Id")
		if requestID == "" {
			requestID = newRequestID()
		}

		ctx := context.WithValue(r.Context(), requestIDKey{}, requestID)
		w.Header().Set("X-Request-Id", requestID)
		s.logger.Printf("request_id=%s method=%s path=%s", requestID, r.Method, r.URL.Path)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func newRequestID() string {
	return "r_" + time.Now().UTC().Format("20060102T150405.000000000")
}

func normalizePolicy(policy string) (string, error) {
	trimmed := strings.TrimSpace(strings.ToLower(policy))
	if trimmed == "" {
		return "overwrite", nil
	}

	switch trimmed {
	case "overwrite", "autorename":
		return trimmed, nil
	default:
		return "", errors.New("policy must be overwrite or autorename")
	}
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
