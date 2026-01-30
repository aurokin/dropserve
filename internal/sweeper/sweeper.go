package sweeper

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"dropserve/internal/control"
)

type Config struct {
	TempDirName      string
	SweepInterval    time.Duration
	PartMaxAge       time.Duration
	PortalIdleMaxAge time.Duration
	Roots            []string
}

type Sweeper struct {
	cfg    Config
	store  *control.Store
	logger *log.Logger
}

func New(cfg Config, store *control.Store, logger *log.Logger) *Sweeper {
	if cfg.TempDirName == "" {
		cfg.TempDirName = ".dropserve_tmp"
	}
	if cfg.SweepInterval <= 0 {
		cfg.SweepInterval = 2 * time.Minute
	}
	if cfg.PartMaxAge <= 0 {
		cfg.PartMaxAge = 10 * time.Minute
	}
	if cfg.PortalIdleMaxAge <= 0 {
		cfg.PortalIdleMaxAge = 30 * time.Minute
	}
	if logger == nil {
		logger = log.New(os.Stdout, "sweep ", log.LstdFlags)
	}

	return &Sweeper{cfg: cfg, store: store, logger: logger}
}

func (s *Sweeper) Run(ctx context.Context) {
	ticker := time.NewTicker(s.cfg.SweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.RunOnce(ctx); err != nil {
				s.logger.Printf("sweep failed: %v", err)
			}
		}
	}
}

func (s *Sweeper) RunOnce(ctx context.Context) error {
	activeUploads := s.activeUploadIDs()
	activePortals := s.activePortalIDs()
	roots := s.sweepRoots()

	var sweepErr error
	for _, root := range roots {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if strings.TrimSpace(root) == "" {
			continue
		}
		if err := s.sweepRoot(root, activeUploads, activePortals); err != nil && sweepErr == nil {
			sweepErr = err
		}
	}

	return sweepErr
}

func (s *Sweeper) sweepRoots() []string {
	roots := make(map[string]struct{})
	for _, root := range s.cfg.Roots {
		trimmed := strings.TrimSpace(root)
		if trimmed == "" {
			continue
		}
		roots[trimmed] = struct{}{}
	}
	if s.store != nil {
		for _, portal := range s.store.ListPortals() {
			if strings.TrimSpace(portal.DestAbs) != "" {
				roots[portal.DestAbs] = struct{}{}
			}
		}
	}

	out := make([]string, 0, len(roots))
	for root := range roots {
		out = append(out, root)
	}
	return out
}

func (s *Sweeper) activePortalIDs() map[string]struct{} {
	active := make(map[string]struct{})
	if s.store == nil {
		return active
	}
	for _, portal := range s.store.ListPortals() {
		if portal.ActiveUploads > 0 {
			active[portal.ID] = struct{}{}
		}
	}
	return active
}

func (s *Sweeper) activeUploadIDs() map[string]struct{} {
	if s.store == nil {
		return make(map[string]struct{})
	}
	return s.store.ActiveUploadIDs()
}

func (s *Sweeper) sweepRoot(root string, activeUploads, activePortals map[string]struct{}) error {
	tempRoot := filepath.Join(root, s.cfg.TempDirName)
	entries, err := os.ReadDir(tempRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		portalID := entry.Name()
		portalPath := filepath.Join(tempRoot, portalID)
		if err := s.sweepPortalDir(portalID, portalPath, activeUploads, activePortals); err != nil {
			s.logger.Printf("sweeper portal=%s err=%v", portalID, err)
		}
	}

	return nil
}

func (s *Sweeper) sweepPortalDir(portalID, portalPath string, activeUploads, activePortals map[string]struct{}) error {
	lastActivity := modTime(portalPath)
	uploadsDir := filepath.Join(portalPath, "uploads")
	entries, err := os.ReadDir(uploadsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return s.maybeRemovePortal(portalID, portalPath, lastActivity, activePortals)
		}
		return err
	}

	now := time.Now()
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(lastActivity) {
			lastActivity = info.ModTime()
		}

		ext := filepath.Ext(entry.Name())
		if ext != ".part" && ext != ".json" {
			continue
		}

		uploadID := strings.TrimSuffix(entry.Name(), ext)
		if _, ok := activeUploads[uploadID]; ok {
			continue
		}
		if now.Sub(info.ModTime()) <= s.cfg.PartMaxAge {
			continue
		}

		path := filepath.Join(uploadsDir, entry.Name())
		if err := os.Remove(path); err != nil {
			if !os.IsNotExist(err) {
				s.logger.Printf("sweeper remove failed path=%s err=%v", path, err)
			}
			continue
		}
		s.logger.Printf("sweeper removed stale upload artifact path=%s", path)
	}

	return s.maybeRemovePortal(portalID, portalPath, lastActivity, activePortals)
}

func (s *Sweeper) maybeRemovePortal(portalID, portalPath string, lastActivity time.Time, activePortals map[string]struct{}) error {
	if _, ok := activePortals[portalID]; ok {
		return nil
	}
	if lastActivity.IsZero() {
		return nil
	}
	if time.Since(lastActivity) <= s.cfg.PortalIdleMaxAge {
		return nil
	}
	if err := os.RemoveAll(portalPath); err != nil {
		return err
	}
	s.logger.Printf("sweeper removed stale portal dir path=%s", portalPath)
	return nil
}

func modTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}
