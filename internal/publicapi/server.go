package publicapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"dropserve/internal/control"
)

type Server struct {
	store  *control.Store
	logger *log.Logger
}

type errorResponse struct {
	Error string `json:"error"`
}

type ClaimPortalResponse struct {
	PortalID    string      `json:"portal_id"`
	ClientToken string      `json:"client_token"`
	ExpiresAt   string      `json:"expires_at"`
	Policy      ClaimPolicy `json:"policy"`
}

type ClaimPolicy struct {
	Overwrite  bool `json:"overwrite"`
	Autorename bool `json:"autorename"`
}

type requestIDKey struct{}

func NewServer(store *control.Store, logger *log.Logger) *Server {
	return &Server{store: store, logger: logger}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/portals/", s.handlePortals)
	return s.withRequestID(mux)
}

func (s *Server) handlePortals(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/portals/")
	if path == r.URL.Path {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "not found"})
		return
	}

	segments := strings.Split(strings.Trim(path, "/"), "/")
	if len(segments) != 2 {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "not found"})
		return
	}

	portalID := segments[0]
	action := segments[1]

	switch action {
	case "claim":
		s.handleClaim(w, r, portalID)
	default:
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "not found"})
	}
}

func (s *Server) handleClaim(w http.ResponseWriter, r *http.Request, portalID string) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	if err := decodeEmptyJSON(r); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json"})
		return
	}

	result, err := s.store.ClaimPortal(portalID)
	if err != nil {
		switch {
		case errors.Is(err, control.ErrPortalNotFound):
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "portal not found"})
		case errors.Is(err, control.ErrPortalAlreadyClaimed):
			writeJSON(w, http.StatusConflict, errorResponse{Error: "portal already claimed"})
		default:
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to claim portal"})
		}
		return
	}

	resp := ClaimPortalResponse{
		PortalID:    result.Portal.ID,
		ClientToken: result.ClientToken,
		ExpiresAt:   result.Portal.OpenUntil.Format(time.RFC3339),
		Policy: ClaimPolicy{
			Overwrite:  result.Portal.DefaultPolicy == "overwrite",
			Autorename: result.Portal.DefaultPolicy == "autorename",
		},
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) requireClientToken(w http.ResponseWriter, r *http.Request, portalID string) bool {
	token := strings.TrimSpace(r.Header.Get("X-Client-Token"))
	if err := s.store.RequireClientToken(portalID, token); err != nil {
		switch {
		case errors.Is(err, control.ErrPortalNotFound):
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "portal not found"})
		case errors.Is(err, control.ErrClientTokenRequired):
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "client token required"})
		case errors.Is(err, control.ErrClientTokenInvalid):
			writeJSON(w, http.StatusForbidden, errorResponse{Error: "client token invalid"})
		default:
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to validate client token"})
		}
		return false
	}

	return true
}

func decodeEmptyJSON(r *http.Request) error {
	if r.Body == nil {
		return nil
	}

	var payload struct{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&payload); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}

	if decoder.More() {
		return errors.New("invalid json")
	}

	return nil
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

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
