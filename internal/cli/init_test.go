package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/m-tsuru/sloop/internal/sloop"
)

func TestInitCreatesSharedAndLocalProjectData(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "xdg"))

	var stdout, stderr bytes.Buffer
	if err := Execute([]string{"init", "demo", repo}, &stdout, &stderr); err != nil {
		t.Fatalf("init: %v", err)
	}
	match := regexp.MustCompile(`Project ID: (demo-[0-9a-f-]+)`).FindStringSubmatch(stdout.String())
	if len(match) != 2 {
		t.Fatalf("unexpected output: %q", stdout.String())
	}

	cfg, err := sloop.ReadConfig(filepath.Join(repo, ".sloop", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Project.ID != match[1] || cfg.Project.SpecPrefix != "demo" || cfg.Git.NotesRef != "refs/notes/sloop" {
		t.Fatalf("unexpected config: %#v", cfg)
	}
	template, err := os.ReadFile(filepath.Join(repo, ".sloop", "templates", "default.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(template), "{#acceptance-criteria}") {
		t.Fatalf("unexpected template: %q", template)
	}
	paths, err := sloop.PathsForProject(cfg.Project.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{paths.DB, paths.Objects, paths.Cache} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected local path %s: %v", path, err)
		}
	}
}

func TestInitRefusesExistingProject(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	t.Setenv("HOME", home)
	var stdout, stderr bytes.Buffer
	if err := Execute([]string{"init", "demo", repo}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	err := Execute([]string{"init", "demo", repo}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "already initialized") {
		t.Fatalf("expected already initialized error, got %v", err)
	}
}
