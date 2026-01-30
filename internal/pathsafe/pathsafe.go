package pathsafe

import (
	"errors"
	"path"
	"path/filepath"
	"strings"
	"unicode"
)

var (
	ErrRelpathEmpty   = errors.New("relpath required")
	ErrRelpathInvalid = errors.New("relpath invalid")
	ErrDestAbsInvalid = errors.New("dest_abs must be absolute")
	ErrRelpathEscapes = errors.New("relpath escapes destination")
)

func SanitizeRelpath(input string) (string, error) {
	if input == "" {
		return "", ErrRelpathEmpty
	}

	normalized := strings.ReplaceAll(input, "\\", "/")
	if strings.ContainsRune(normalized, '\x00') {
		return "", ErrRelpathInvalid
	}
	if strings.HasPrefix(normalized, "/") || strings.HasPrefix(normalized, "~/") {
		return "", ErrRelpathInvalid
	}

	segments := strings.Split(normalized, "/")
	for _, segment := range segments {
		if segment == ".." {
			return "", ErrRelpathInvalid
		}
		if hasWindowsDrivePrefix(segment) {
			return "", ErrRelpathInvalid
		}
	}

	cleaned := path.Clean(normalized)
	if cleaned == "." || cleaned == "" {
		return "", ErrRelpathEmpty
	}
	if path.IsAbs(cleaned) {
		return "", ErrRelpathInvalid
	}

	return cleaned, nil
}

func JoinAndVerify(destAbs, cleanedRelpath string) (string, error) {
	if cleanedRelpath == "" {
		return "", ErrRelpathEmpty
	}
	if path.IsAbs(cleanedRelpath) {
		return "", ErrRelpathInvalid
	}

	destAbs = filepath.Clean(destAbs)
	if destAbs == "" || destAbs == "." || !filepath.IsAbs(destAbs) {
		return "", ErrDestAbsInvalid
	}

	finalAbs := filepath.Clean(filepath.Join(destAbs, filepath.FromSlash(cleanedRelpath)))
	rel, err := filepath.Rel(destAbs, finalAbs)
	if err != nil {
		return "", ErrRelpathEscapes
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", ErrRelpathEscapes
	}

	return finalAbs, nil
}

func hasWindowsDrivePrefix(segment string) bool {
	if len(segment) < 2 {
		return false
	}
	if segment[1] != ':' {
		return false
	}
	return unicode.IsLetter(rune(segment[0]))
}
