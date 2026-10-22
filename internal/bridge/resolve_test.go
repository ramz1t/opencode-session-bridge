package bridge

import (
	"os"
	"path/filepath"
	"testing"

	"opencode-session-bridge/internal/opencode"
)

func TestResolveDirectory(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "my-app"))
	mkdir(t, filepath.Join(root, "other-app"))
	known := []opencode.Session{
		{ID: "ses_1", Directory: "/nonexistent/elsewhere/my-app"},
	}

	t.Run("absolute path that exists", func(t *testing.T) {
		got, err := resolveDirectory(filepath.Join(root, "other-app"), nil, nil)
		assertDir(t, got, err, filepath.Join(root, "other-app"))
	})

	t.Run("absolute path that does not exist", func(t *testing.T) {
		if _, err := resolveDirectory(filepath.Join(root, "missing"), nil, nil); err == nil {
			t.Fatal("expected an error")
		}
	})

	t.Run("name found in a configured root", func(t *testing.T) {
		got, err := resolveDirectory("other-app", nil, []string{root})
		assertDir(t, got, err, filepath.Join(root, "other-app"))
	})

	t.Run("name matches a known session directory case-insensitively", func(t *testing.T) {
		dir := mkdirTemp(t, "Known-App")
		sessions := []opencode.Session{{ID: "ses_2", Directory: dir}}
		got, err := resolveDirectory("known-app", sessions, nil)
		assertDir(t, got, err, dir)
	})

	t.Run("known session directory wins over a root", func(t *testing.T) {
		dir := mkdirTemp(t, "my-app")
		got, err := resolveDirectory("my-app", []opencode.Session{{ID: "ses_3", Directory: dir}}, []string{root})
		assertDir(t, got, err, dir)
	})

	t.Run("unknown name", func(t *testing.T) {
		if _, err := resolveDirectory("nope", known, []string{root}); err == nil {
			t.Fatal("expected an error")
		}
	})
}

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mkdirTemp(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	mkdir(t, path)
	return path
}

func assertDir(t *testing.T, got string, err error, want string) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}
