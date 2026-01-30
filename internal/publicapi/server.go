package publicapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"dropserve/internal/config"
	"dropserve/internal/control"
	"dropserve/internal/pathsafe"
)

type Server struct {
	store       *control.Store
	logger      *log.Logger
	tempDirName string
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

type InitUploadRequest struct {
	UploadID     string  `json:"upload_id"`
	Relpath      string  `json:"relpath"`
	Size         int64   `json:"size"`
	ClientSHA256 *string `json:"client_sha256"`
	Policy       string  `json:"policy"`
}

type InitUploadResponse struct {
	UploadID string `json:"upload_id"`
	PutURL   string `json:"put_url"`
}

type UploadCommitResponse struct {
	Status        string `json:"status"`
	Relpath       string `json:"relpath"`
	ServerSHA256  string `json:"server_sha256"`
	BytesReceived int64  `json:"bytes_received"`
	FinalRelpath  string `json:"final_relpath"`
}

type UploadStatusResponse struct {
	UploadID      string  `json:"upload_id"`
	Status        string  `json:"status"`
	ServerSHA256  *string `json:"server_sha256"`
	FinalRelpath  *string `json:"final_relpath"`
	BytesReceived int64   `json:"bytes_received"`
}

type requestIDKey struct{}

func NewServer(store *control.Store, logger *log.Logger) *Server {
	return &Server{
		store:       store,
		logger:      logger,
		tempDirName: config.TempDirName(),
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/portals/", s.handlePortals)
	mux.HandleFunc("/api/uploads/", s.handleUploads)
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
	case "uploads":
		s.handleInitUpload(w, r, portalID)
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

func (s *Server) handleInitUpload(w http.ResponseWriter, r *http.Request, portalID string) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	if !s.requireClientToken(w, r, portalID) {
		return
	}

	var req InitUploadRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json"})
		return
	}

	if strings.TrimSpace(req.UploadID) == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "upload_id required"})
		return
	}
	if req.Size < 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "size must be non-negative"})
		return
	}

	portal, err := s.store.PortalByID(portalID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "portal not found"})
		return
	}

	cleanedRelpath, err := pathsafe.SanitizeRelpath(req.Relpath)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid relpath"})
		return
	}
	if _, err := pathsafe.JoinAndVerify(portal.DestAbs, cleanedRelpath); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid relpath"})
		return
	}

	policy := strings.TrimSpace(req.Policy)
	if policy == "" {
		policy = portal.DefaultPolicy
	}
	policy, err = control.NormalizePolicy(policy)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	clientSHA := ""
	if req.ClientSHA256 != nil {
		clientSHA = strings.TrimSpace(*req.ClientSHA256)
	}

	if _, err := s.store.CreateUpload(control.CreateUploadInput{
		PortalID:     portal.ID,
		UploadID:     req.UploadID,
		Relpath:      cleanedRelpath,
		Size:         req.Size,
		ClientSHA256: clientSHA,
		Policy:       policy,
	}); err != nil {
		switch {
		case errors.Is(err, control.ErrPortalNotFound):
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "portal not found"})
		case errors.Is(err, control.ErrUploadAlreadyCommitted):
			writeJSON(w, http.StatusConflict, errorResponse{Error: "upload already committed"})
		case errors.Is(err, control.ErrUploadAlreadyExists):
			writeJSON(w, http.StatusConflict, errorResponse{Error: "upload already exists"})
		default:
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to initialize upload"})
		}
		return
	}

	tempDir := s.uploadTempDir(portal.DestAbs, portal.ID)
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		s.store.DeleteUpload(req.UploadID)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to prepare upload"})
		return
	}

	_, metaPath := uploadTempPaths(tempDir, req.UploadID)
	meta := uploadMetadata{
		PortalID:     portal.ID,
		UploadID:     req.UploadID,
		Relpath:      cleanedRelpath,
		Size:         req.Size,
		Policy:       policy,
		ClientSHA256: clientSHA,
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
	}
	if err := writeUploadMetadata(metaPath, meta); err != nil {
		cleanupUploadArtifacts("", metaPath)
		s.store.DeleteUpload(req.UploadID)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to prepare upload"})
		return
	}

	writeJSON(w, http.StatusOK, InitUploadResponse{
		UploadID: req.UploadID,
		PutURL:   "/api/uploads/" + req.UploadID,
	})
}

func (s *Server) handleUploads(w http.ResponseWriter, r *http.Request) {
	pathValue := strings.TrimPrefix(r.URL.Path, "/api/uploads/")
	if pathValue == r.URL.Path {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "not found"})
		return
	}

	segments := strings.Split(strings.Trim(pathValue, "/"), "/")
	if len(segments) == 1 {
		s.handleUploadStream(w, r, segments[0])
		return
	}
	if len(segments) == 2 && segments[1] == "status" {
		s.handleUploadStatus(w, r, segments[0])
		return
	}

	writeJSON(w, http.StatusNotFound, errorResponse{Error: "not found"})
}

func (s *Server) handleUploadStream(w http.ResponseWriter, r *http.Request, uploadID string) {
	if r.Method != http.MethodPut {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	upload, err := s.store.GetUpload(uploadID)
	if err != nil {
		switch {
		case errors.Is(err, control.ErrUploadNotFound):
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "upload not found"})
		default:
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to load upload"})
		}
		return
	}

	if upload.Status == control.UploadCommitted {
		writeJSON(w, http.StatusConflict, errorResponse{Error: "upload already committed"})
		return
	}

	portal, err := s.store.PortalByID(upload.PortalID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "portal not found"})
		return
	}

	if !s.requireClientToken(w, r, portal.ID) {
		return
	}

	tempDir := s.uploadTempDir(portal.DestAbs, portal.ID)
	partPath, metaPath := uploadTempPaths(tempDir, uploadID)

	if _, err := s.store.StartUpload(uploadID); err != nil {
		switch {
		case errors.Is(err, control.ErrPortalNotFound):
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "portal not found"})
		case errors.Is(err, control.ErrUploadNotFound):
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "upload not found"})
		default:
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to start upload"})
		}
		return
	}

	if r.ContentLength < 0 || r.ContentLength != upload.Size {
		s.failUpload(uploadID, partPath, metaPath)
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "size mismatch"})
		return
	}

	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		s.failUpload(uploadID, partPath, metaPath)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to prepare upload"})
		return
	}

	file, err := os.OpenFile(partPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		s.failUpload(uploadID, partPath, metaPath)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to write upload"})
		return
	}
	defer func() {
		_ = file.Close()
	}()
	defer func() {
		_ = r.Body.Close()
	}()

	hasher := sha256.New()
	bytesWritten, err := io.Copy(io.MultiWriter(file, hasher), r.Body)
	if err != nil {
		s.failUpload(uploadID, partPath, metaPath)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to stream upload"})
		return
	}
	if bytesWritten != upload.Size {
		s.failUpload(uploadID, partPath, metaPath)
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "size mismatch"})
		return
	}

	serverSHA := hex.EncodeToString(hasher.Sum(nil))
	if upload.ClientSHA256 != "" && !strings.EqualFold(serverSHA, upload.ClientSHA256) {
		s.failUpload(uploadID, partPath, metaPath)
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "sha256 mismatch"})
		return
	}

	finalRelpath, finalAbs, err := resolveFinalRelpath(portal.DestAbs, upload.Relpath, upload.Policy)
	if err != nil {
		s.failUpload(uploadID, partPath, metaPath)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to finalize upload"})
		return
	}
	if err := os.MkdirAll(filepath.Dir(finalAbs), 0o755); err != nil {
		s.failUpload(uploadID, partPath, metaPath)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to finalize upload"})
		return
	}

	if err := os.Rename(partPath, finalAbs); err != nil {
		s.failUpload(uploadID, partPath, metaPath)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to commit upload"})
		return
	}

	if err := os.Remove(metaPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		s.logger.Printf("failed to remove metadata: %v", err)
	}

	committed, err := s.store.MarkUploadCommitted(uploadID, serverSHA, finalRelpath, bytesWritten)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to commit upload"})
		return
	}

	writeJSON(w, http.StatusOK, UploadCommitResponse{
		Status:        string(committed.Status),
		Relpath:       committed.Relpath,
		ServerSHA256:  committed.ServerSHA256,
		BytesReceived: committed.BytesReceived,
		FinalRelpath:  committed.FinalRelpath,
	})
}

func (s *Server) handleUploadStatus(w http.ResponseWriter, r *http.Request, uploadID string) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	upload, err := s.store.GetUpload(uploadID)
	if err != nil {
		if errors.Is(err, control.ErrUploadNotFound) {
			writeJSON(w, http.StatusOK, UploadStatusResponse{
				UploadID:      uploadID,
				Status:        "not_found",
				ServerSHA256:  nil,
				FinalRelpath:  nil,
				BytesReceived: 0,
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to load upload"})
		return
	}

	var serverSHA *string
	if upload.ServerSHA256 != "" {
		serverSHA = &upload.ServerSHA256
	}
	var finalRelpath *string
	if upload.FinalRelpath != "" {
		finalRelpath = &upload.FinalRelpath
	}

	writeJSON(w, http.StatusOK, UploadStatusResponse{
		UploadID:      upload.ID,
		Status:        string(upload.Status),
		ServerSHA256:  serverSHA,
		FinalRelpath:  finalRelpath,
		BytesReceived: upload.BytesReceived,
	})
}

func (s *Server) failUpload(uploadID, partPath, metaPath string) {
	_, _ = s.store.MarkUploadFailed(uploadID)
	cleanupUploadArtifacts(partPath, metaPath)
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

type uploadMetadata struct {
	PortalID     string `json:"portal_id"`
	UploadID     string `json:"upload_id"`
	Relpath      string `json:"relpath"`
	Size         int64  `json:"size"`
	Policy       string `json:"policy"`
	ClientSHA256 string `json:"client_sha256,omitempty"`
	CreatedAt    string `json:"created_at"`
}

func (s *Server) uploadTempDir(destAbs, portalID string) string {
	return filepath.Join(destAbs, s.tempDirName, portalID, "uploads")
}

func uploadTempPaths(tempDir, uploadID string) (string, string) {
	partPath := filepath.Join(tempDir, uploadID+".part")
	metaPath := filepath.Join(tempDir, uploadID+".json")
	return partPath, metaPath
}

func writeUploadMetadata(path string, meta uploadMetadata) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	encoder := json.NewEncoder(file)
	return encoder.Encode(meta)
}

func cleanupUploadArtifacts(partPath, metaPath string) {
	if partPath != "" {
		_ = os.Remove(partPath)
	}
	if metaPath != "" {
		_ = os.Remove(metaPath)
	}
}

func resolveFinalRelpath(destAbs, relpath, policy string) (string, string, error) {
	finalRelpath := relpath
	finalAbs, err := pathsafe.JoinAndVerify(destAbs, relpath)
	if err != nil {
		return "", "", err
	}

	if policy != "autorename" {
		return finalRelpath, finalAbs, nil
	}

	if _, err := os.Stat(finalAbs); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return finalRelpath, finalAbs, nil
		}
		return "", "", err
	}

	dir, base := path.Split(relpath)
	ext := path.Ext(base)
	name := strings.TrimSuffix(base, ext)
	timestamp := time.Now().Format("2006-01-02_150405")

	for i := 0; ; i++ {
		suffix := ""
		if i > 0 {
			suffix = fmt.Sprintf("_%d", i+1)
		}
		candidate := fmt.Sprintf("%s_%s%s%s", name, timestamp, suffix, ext)
		candidateRelpath := path.Join(dir, candidate)
		candidateAbs, err := pathsafe.JoinAndVerify(destAbs, candidateRelpath)
		if err != nil {
			return "", "", err
		}
		if _, err := os.Stat(candidateAbs); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return candidateRelpath, candidateAbs, nil
			}
			return "", "", err
		}
	}
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
