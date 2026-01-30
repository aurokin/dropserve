package pathsafe

import (
	"path/filepath"
	"testing"
)

func TestSanitizeRelpathRejectsUnsafePaths(t *testing.T) {
	rejects := []string{
		"../etc/passwd",
		"a/../../b",
		"/absolute/path",
		"C:\\Windows\\System32",
		"..",
		"a/..",
		"a\\..\\b",
		"a/../b",
	}

	for _, input := range rejects {
		if _, err := SanitizeRelpath(input); err == nil {
			t.Errorf("expected error for %q", input)
		}
	}
}

func TestSanitizeRelpathAcceptsAndCleans(t *testing.T) {
	accepts := map[string]string{
		"a/b/c.txt":    "a/b/c.txt",
		"a//b///c.txt": "a/b/c.txt",
		"a/./b/c.txt":  "a/b/c.txt",
	}

	for input, expected := range accepts {
		result, err := SanitizeRelpath(input)
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", input, err)
		}
		if result != expected {
			t.Fatalf("expected %q for %q, got %q", expected, input, result)
		}
	}
}

func TestJoinAndVerifyKeepsPathsContained(t *testing.T) {
	destAbs := t.TempDir()
	cleaned, err := SanitizeRelpath("a/b/c.txt")
	if err != nil {
		t.Fatalf("unexpected sanitize error: %v", err)
	}

	finalAbs, err := JoinAndVerify(destAbs, cleaned)
	if err != nil {
		t.Fatalf("unexpected join error: %v", err)
	}

	expected := filepath.Clean(filepath.Join(destAbs, filepath.FromSlash(cleaned)))
	if finalAbs != expected {
		t.Fatalf("expected %q, got %q", expected, finalAbs)
	}
}

func TestJoinAndVerifyRejectsEscapes(t *testing.T) {
	destAbs := t.TempDir()
	if _, err := JoinAndVerify(destAbs, "../escape.txt"); err == nil {
		t.Fatalf("expected escape error")
	}
}
