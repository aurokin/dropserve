package control

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"
	"sync"
	"time"
)

const defaultOpenMinutes = 15

type Portal struct {
	ID                   string
	DestAbs              string
	OpenUntil            time.Time
	CreatedAt            time.Time
	Reusable             bool
	DefaultPolicy        string
	AutorenameOnConflict bool
}

type CreatePortalInput struct {
	DestAbs              string
	OpenMinutes          int
	Reusable             bool
	DefaultPolicy        string
	AutorenameOnConflict bool
}

type Store struct {
	mu      sync.Mutex
	portals map[string]Portal
}

func NewStore() *Store {
	return &Store{portals: make(map[string]Portal)}
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
	}

	s.mu.Lock()
	s.portals[id] = portal
	s.mu.Unlock()

	return portal, nil
}

func newPortalID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate portal id: %w", err)
	}

	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf)
	return "p_" + strings.ToLower(encoded), nil
}
