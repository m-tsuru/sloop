package cli

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/m-tsuru/sloop/internal/sloop"
)

func TestReferenceLifecycleMarksWorkingStateDraft(t *testing.T) {
	repo := setupProject(t)
	createDraftSpecification(t)
	if _, err := executeForTest(t, "ready", "1", "--author.name", "human"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(repo+"/parser.go", []byte("package parser\n\nfunc Parse() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	output, err := executeForTest(t, "ref", "add", "1", "--kind", "code", "parser.go:1-3", "--author.name", "human")
	if err != nil {
		t.Fatal(err)
	}
	fields := strings.Fields(output)
	if len(fields) < 2 {
		t.Fatalf("unexpected output: %q", output)
	}
	referenceID := fields[1]

	project, _ := openProject()
	spec, err := project.Store.SpecificationByID(context.Background(), "demo-1")
	project.Store.Close()
	if err != nil {
		t.Fatal(err)
	}
	if spec.Status != sloop.StatusDraft || !spec.Dirty || len(spec.References) != 1 || spec.References[0].Path != "parser.go" {
		t.Fatalf("unexpected referenced specification: %#v", spec)
	}
	list, err := executeForTest(t, "ref", "list", "1")
	if err != nil || !strings.Contains(list, "parser.go:1-3") {
		t.Fatalf("unexpected list %q, %v", list, err)
	}
	if _, err := executeForTest(t, "ref", "remove", "1", referenceID[:8], "--author.name", "human"); err != nil {
		t.Fatal(err)
	}
	project, _ = openProject()
	defer project.Store.Close()
	spec, _ = project.Store.SpecificationByID(context.Background(), "demo-1")
	if len(spec.References) != 0 || !spec.Dirty {
		t.Fatalf("reference was not removed: %#v", spec)
	}
}

func TestReferenceAtCommitStoresFullCommit(t *testing.T) {
	repo := setupProject(t)
	createDraftSpecification(t)
	runGit(t, repo, "init")
	runGit(t, repo, "config", "user.name", "test")
	runGit(t, repo, "config", "user.email", "test@example.com")
	if err := os.WriteFile(repo+"/go.txt", []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", "go.txt")
	runGit(t, repo, "commit", "-m", "fixture")
	shortCommit := strings.TrimSpace(runGit(t, repo, "rev-parse", "--short", "HEAD"))
	if _, err := executeForTest(t, "ref", "add", "1", "--kind", "test", "go.txt", "--at", shortCommit); err != nil {
		t.Fatal(err)
	}
	project, _ := openProject()
	defer project.Store.Close()
	spec, _ := project.Store.SpecificationByID(context.Background(), "demo-1")
	if len(spec.References) != 1 || len(spec.References[0].GitCommit) != 40 {
		t.Fatalf("expected full commit hash: %#v", spec.References)
	}
}
