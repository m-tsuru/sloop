package cli

import (
	"context"
	"strings"
	"testing"
)

func TestCommitRecordCreatesRevisionIndexAndGitNote(t *testing.T) {
	repo := setupProject(t)
	runGit(t, repo, "init")
	runGit(t, repo, "config", "user.name", "test")
	runGit(t, repo, "config", "user.email", "test@example.com")
	runGit(t, repo, "add", ".sloop")
	runGit(t, repo, "commit", "-m", "initialize project")
	createDraftSpecification(t)
	commit := strings.TrimSpace(runGit(t, repo, "rev-parse", "HEAD"))
	output, err := executeForTest(t, "cr", "1", commit[:8])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "Recorded demo-1@") {
		t.Fatalf("unexpected output: %q", output)
	}
	note := runGit(t, repo, "notes", "--ref=refs/notes/sloop", "show", commit)
	if !strings.Contains(note, "id: demo-1") || !strings.Contains(note, "revision-hash:") || !strings.Contains(note, "relation: implementation") {
		t.Fatalf("unexpected note:\n%s", note)
	}
	project, _ := openProject()
	defer project.Store.Close()
	spec, _ := project.Store.SpecificationByID(context.Background(), "demo-1")
	relations, err := project.Store.GitRelations(context.Background(), spec.UUID)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Dirty || len(spec.HeadHash) != 64 || len(relations) != 1 || relations[0].Commit != commit || relations[0].RevisionHash != spec.HeadHash {
		t.Fatalf("unexpected relation/spec: %#v %#v", relations, spec)
	}
}

func TestCommitRecordCreatesCartesianRelations(t *testing.T) {
	repo := setupProject(t)
	runGit(t, repo, "init")
	runGit(t, repo, "config", "user.name", "test")
	runGit(t, repo, "config", "user.email", "test@example.com")
	runGit(t, repo, "add", ".sloop")
	runGit(t, repo, "commit", "-m", "first")
	first := strings.TrimSpace(runGit(t, repo, "rev-parse", "HEAD"))
	createDraftSpecification(t)
	t.Setenv("SLOOP_TEST_DOCUMENT", `---
id: demo-2
title: Second
status: DRAFT
parents: []
---
second
`)
	if _, err := executeForTest(t, "new"); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "commit", "--allow-empty", "-m", "second")
	second := strings.TrimSpace(runGit(t, repo, "rev-parse", "HEAD"))
	if _, err := executeForTest(t, "commit-record", "1,2", first+","+second); err != nil {
		t.Fatal(err)
	}
	for _, commit := range []string{first, second} {
		note := runGit(t, repo, "notes", "--ref=refs/notes/sloop", "show", commit)
		if strings.Count(note, "relation: implementation") != 2 {
			t.Fatalf("expected two relations on %s:\n%s", commit, note)
		}
	}
}
