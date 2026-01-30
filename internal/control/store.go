package control

import (
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

const defaultOpenMinutes = 15

var (
	ErrPortalNotFound         = errors.New("portal not found")
	ErrPortalAlreadyClaimed   = errors.New("portal already claimed")
	ErrClientTokenRequired    = errors.New("client token required")
	ErrClientTokenInvalid     = errors.New("client token invalid")
	ErrUploadNotFound         = errors.New("upload not found")
	ErrUploadAlreadyCommitted = errors.New("upload already committed")
	ErrUploadAlreadyExists    = errors.New("upload already exists")
)

type Portal struct {
	ID                   string
	DestAbs              string
	OpenUntil            time.Time
	CreatedAt            time.Time
	Reusable             bool
	DefaultPolicy        string
	AutorenameOnConflict bool
	ClientTokens         map[string]struct{}
}

type UploadStatus string

const (
	UploadWriting   UploadStatus = "writing"
	UploadCommitted UploadStatus = "committed"
	UploadFailed    UploadStatus = "failed"
)

type Upload struct {
	ID            string
	PortalID      string
	Relpath       string
	Size          int64
	ClientSHA256  string
	Policy        string
	Status        UploadStatus
	ServerSHA256  string
	BytesReceived int64
	FinalRelpath  string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type CreatePortalInput struct {
	DestAbs              string
	OpenMinutes          int
	Reusable             bool
	DefaultPolicy        string
	AutorenameOnConflict bool
}

type CreateUploadInput struct {
	PortalID     string
	UploadID     string
	Relpath      string
	Size         int64
	ClientSHA256 string
	Policy       string
}

type ClaimPortalResult struct {
	Portal      Portal
	ClientToken string
}

type Store struct {
	mu      sync.Mutex
	portals map[string]Portal
	uploads map[string]Upload
}

func NewStore() *Store {
	return &Store{portals: make(map[string]Portal), uploads: make(map[string]Upload)}
}

func (s *Store) CreatePortal(input CreatePortalInput) (Portal, error) {
	id, err := newPortalID()
	if err != nil {
		return Portal{}, err
	}

	minutes := input.OpenMinutes
	if minutes <= 0 {
		minutes = defaultOpenMinutes
	}

	now := time.Now()
	portal := Portal{
		ID:                   id,
		DestAbs:              input.DestAbs,
		OpenUntil:            now.Add(time.Minute * time.Duration(minutes)),
		CreatedAt:            now,
		Reusable:             input.Reusable,
		DefaultPolicy:        input.DefaultPolicy,
		AutorenameOnConflict: input.AutorenameOnConflict,
		ClientTokens:         make(map[string]struct{}),
	}

	s.mu.Lock()
	s.portals[id] = portal
	s.mu.Unlock()

	return portal, nil
}

func (s *Store) ClaimPortal(id string) (ClaimPortalResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	portal, ok := s.portals[id]
	if !ok {
		return ClaimPortalResult{}, ErrPortalNotFound
	}

	if !portal.Reusable && len(portal.ClientTokens) > 0 {
		return ClaimPortalResult{}, ErrPortalAlreadyClaimed
	}

	clientToken, err := newClientToken()
	if err != nil {
		return ClaimPortalResult{}, err
	}

	if portal.ClientTokens == nil {
		portal.ClientTokens = make(map[string]struct{})
	}
	portal.ClientTokens[clientToken] = struct{}{}
	s.portals[id] = portal

	return ClaimPortalResult{Portal: portal, ClientToken: clientToken}, nil
}

func (s *Store) RequireClientToken(id, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	portal, ok := s.portals[id]
	if !ok {
		return ErrPortalNotFound
	}

	if !portal.Reusable && len(portal.ClientTokens) == 0 {
		return ErrClientTokenRequired
	}

	if len(portal.ClientTokens) == 0 {
		return nil
	}

	if token == "" {
		return ErrClientTokenRequired
	}

	if _, ok := portal.ClientTokens[token]; !ok {
		return ErrClientTokenInvalid
	}

	return nil
}

func (s *Store) PortalByID(id string) (Portal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	portal, ok := s.portals[id]
	if !ok {
		return Portal{}, ErrPortalNotFound
	}

	return portal, nil
}

func (s *Store) CreateUpload(input CreateUploadInput) (Upload, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.portals[input.PortalID]; !ok {
		return Upload{}, ErrPortalNotFound
	}

	if existing, ok := s.uploads[input.UploadID]; ok {
		if existing.Status == UploadCommitted {
			return Upload{}, ErrUploadAlreadyCommitted
		}
		return Upload{}, ErrUploadAlreadyExists
	}

	now := time.Now()
	upload := Upload{
		ID:            input.UploadID,
		PortalID:      input.PortalID,
		Relpath:       input.Relpath,
		Size:          input.Size,
		ClientSHA256:  input.ClientSHA256,
		Policy:        input.Policy,
		Status:        UploadWriting,
		BytesReceived: 0,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	s.uploads[input.UploadID] = upload
	return upload, nil
}

func (s *Store) GetUpload(id string) (Upload, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	upload, ok := s.uploads[id]
	if !ok {
		return Upload{}, ErrUploadNotFound
	}

	return upload, nil
}

func (s *Store) DeleteUpload(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.uploads, id)
}

func (s *Store) MarkUploadCommitted(id, serverSHA256, finalRelpath string, bytesReceived int64) (Upload, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	upload, ok := s.uploads[id]
	if !ok {
		return Upload{}, ErrUploadNotFound
	}

	upload.Status = UploadCommitted
	upload.ServerSHA256 = serverSHA256
	upload.BytesReceived = bytesReceived
	upload.FinalRelpath = finalRelpath
	upload.UpdatedAt = time.Now()
	s.uploads[id] = upload

	return upload, nil
}

func (s *Store) MarkUploadFailed(id string) (Upload, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	upload, ok := s.uploads[id]
	if !ok {
		return Upload{}, ErrUploadNotFound
	}

	upload.Status = UploadFailed
	upload.UpdatedAt = time.Now()
	s.uploads[id] = upload

	return upload, nil
}

func newPortalID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate portal id: %w", err)
	}

	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf)
	return "p_" + strings.ToLower(encoded), nil
}

func newClientToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate client token: %w", err)
	}

	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf)
	return "ct_" + strings.ToLower(encoded), nil
}
