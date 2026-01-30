package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	defaultTempDirName          = ".dropserve_tmp"
	defaultSweepIntervalSeconds = 120
	defaultPartMaxAgeSeconds    = 600
	defaultPortalIdleMaxSeconds = 1800
)

func TempDirName() string {
	value := strings.TrimSpace(os.Getenv("DROPSERVE_TMP_DIR_NAME"))
	if value == "" {
		return defaultTempDirName
	}
	return value
}

func SweepInterval() time.Duration {
	return durationSecondsFromEnv("DROPSERVE_SWEEP_INTERVAL_SECONDS", defaultSweepIntervalSeconds)
}

func PartMaxAge() time.Duration {
	return durationSecondsFromEnv("DROPSERVE_PART_MAX_AGE_SECONDS", defaultPartMaxAgeSeconds)
}

func PortalIdleMaxAge() time.Duration {
	return durationSecondsFromEnv("DROPSERVE_PORTAL_IDLE_MAX_SECONDS", defaultPortalIdleMaxSeconds)
}

func SweepRoots() []string {
	raw := strings.TrimSpace(os.Getenv("DROPSERVE_SWEEP_ROOTS"))
	if raw == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil
		}
		return []string{cwd}
	}

	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == os.PathListSeparator
	})

	roots := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		abs, err := filepath.Abs(trimmed)
		if err != nil {
			roots = append(roots, filepath.Clean(trimmed))
			continue
		}
		roots = append(roots, abs)
	}

	return roots
}

func durationSecondsFromEnv(name string, defaultSeconds int) time.Duration {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return time.Duration(defaultSeconds) * time.Second
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return time.Duration(defaultSeconds) * time.Second
	}
	return time.Duration(value) * time.Second
}
