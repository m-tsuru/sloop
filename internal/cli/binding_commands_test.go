package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/m-tsuru/sloop/internal/sloop"
)

func executeWithStderrForTest(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	err := Execute(args, &stdout, &stderr)
	return stdout.String(), stderr.String(), err
}

func TestMarkdownBindingsAreRecordedAndExposedInContext(t *testing.T) {
	repo := setupProject(t)
	if err := os.WriteFile(repo+"/feature.go", []byte("package feature\n\nfunc Run() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(repo+"/feature_test.go", []byte("package feature\n\nfunc TestRun() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SLOOP_TEST_DOCUMENT", `---
id: demo-1
title: Bound feature
status: READY
parents: []
func:
  run:
    impls:
      - feature.go:Run
    tests:
      - feature_test.go:TestRun
---

## Run {#func:run}

Run the feature.
`)
	if _, err := executeForTest(t, "new", "--author.name", "human"); err != nil {
		t.Fatal(err)
	}
	output, err := executeForTest(t, "context", "1", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		SpecificationUUID string                    `json:"specification_uuid"`
		Features          []sloop.FeatureResolution `json:"features"`
	}
	if err := json.Unmarshal([]byte(output), &document); err != nil {
		t.Fatal(err)
	}
	if document.SpecificationUUID == "" || len(document.Features) != 1 || document.Features[0].ID != "run" ||
		document.Features[0].Impls[0].Status != sloop.BindingResolved ||
		document.Features[0].Tests[0].Status != sloop.BindingResolved {
		t.Fatalf("unexpected context features: %#v", document.Features)
	}
	view, err := executeForTest(t, "view", "1")
	if err != nil || !strings.Contains(view, "func:") || !strings.Contains(view, "feature.go:Run") {
		t.Fatalf("view did not contain bindings: %q, %v", view, err)
	}

	project, err := openProject()
	if err != nil {
		t.Fatal(err)
	}
	spec, err := project.Store.SpecificationByID(context.Background(), "demo-1")
	if err != nil {
		t.Fatal(err)
	}
	baseHash := spec.HeadHash
	revision, err := project.Store.RevisionByHash(context.Background(), baseHash)
	project.Store.Close()
	if err != nil || revision.Features["run"].Impls[0] != "feature.go:Run" {
		t.Fatalf("revision did not snapshot features: %#v, %v", revision, err)
	}

	if err := os.Remove(repo + "/feature.go"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeForTest(t, "context", "1", "--json"); err == nil || !strings.Contains(err.Error(), "MISSING") {
		t.Fatalf("READY context should reject a missing binding: %v", err)
	}
	if _, err := executeForTest(t, "context", "1", "--working", "--json"); err == nil || !strings.Contains(err.Error(), "MISSING") {
		t.Fatalf("READY working context should reject a missing binding: %v", err)
	}
	if _, err := executeForTest(t, "force-ready", "1", "--author.name", "human"); err == nil || !strings.Contains(err.Error(), "MISSING") {
		t.Fatalf("FORCEREADY should reject a missing binding: %v", err)
	}
	if _, err := executeForTest(t, "implemented", "1",
		"--reason", "Implementation exists but its registered symbol was removed.",
		"--author.name", "codex", "--author.agent", "true"); err == nil || !strings.Contains(err.Error(), "MISSING") {
		t.Fatalf("IMPLEMENTED should reject a missing binding: %v", err)
	}
	project, _ = openProject()
	defer project.Store.Close()
	historical, err := project.Store.RevisionByHash(context.Background(), baseHash)
	if err != nil || historical.Features["run"].Impls[0] != "feature.go:Run" {
		t.Fatalf("historical binding changed after file deletion: %#v, %v", historical, err)
	}
}

func TestDraftMarkdownAllowsUnresolvedBindingsWithWarning(t *testing.T) {
	setupProject(t)
	t.Setenv("SLOOP_TEST_DOCUMENT", `---
id: demo-1
title: Future implementation
status: DRAFT
parents: []
func:
  future:
    impls:
      - future.go:Run
    tests:
---

## Future {#func:future}
`)
	_, stderr, err := executeWithStderrForTest(t, "new", "--author.name", "human")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr, "Warning: unresolved feature binding") || !strings.Contains(stderr, "[MISSING]") {
		t.Fatalf("missing unresolved warning: %q", stderr)
	}
	if _, err := executeForTest(t, "ready", "1", "--author.name", "human"); err == nil || !strings.Contains(err.Error(), "MISSING") {
		t.Fatalf("READY should fail for unresolved bindings: %v", err)
	}
	if _, err := executeForTest(t, "force-ready", "1", "--author.name", "human"); err == nil || !strings.Contains(err.Error(), "MISSING") {
		t.Fatalf("FORCEREADY should fail for unresolved bindings: %v", err)
	}
}

func TestMarkdownEditAddsAndRemovesFeatureBindings(t *testing.T) {
	repo := setupProject(t)
	if err := os.WriteFile(repo+"/feature.go", []byte("package feature\n\nfunc Run() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SLOOP_TEST_DOCUMENT", `---
id: demo-1
title: Markdown editing
status: DRAFT
parents: []
---
`)
	if _, err := executeForTest(t, "new", "--author.name", "human"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SLOOP_TEST_DOCUMENT", `---
id: demo-1
title: Markdown editing
status: DRAFT
parents: []
func:
  run:
    impls:
      - feature.go:Run
    tests: []
---

## Run {#func:run}
`)
	if _, err := executeForTest(t, "edit", "1", "--author.name", "human"); err != nil {
		t.Fatal(err)
	}
	project, _ := openProject()
	spec, err := project.Store.SpecificationByID(context.Background(), "demo-1")
	project.Store.Close()
	if err != nil || len(spec.Features) != 1 || spec.Features["run"].Impls[0] != "feature.go:Run" {
		t.Fatalf("Markdown edit did not add the binding: %#v, %v", spec, err)
	}

	t.Setenv("SLOOP_TEST_DOCUMENT", `---
id: demo-1
title: Markdown editing
status: DRAFT
parents: []
---
`)
	if _, err := executeForTest(t, "edit", "1", "--author.name", "human"); err != nil {
		t.Fatal(err)
	}
	project, _ = openProject()
	defer project.Store.Close()
	spec, err = project.Store.SpecificationByID(context.Background(), "demo-1")
	if err != nil || len(spec.Features) != 0 {
		t.Fatalf("Markdown edit did not remove the feature: %#v, %v", spec, err)
	}
}

func TestBindingCLIUpdatesSharedStateAndSupportsAgentWorkflow(t *testing.T) {
	repo := setupProject(t)
	t.Setenv("SLOOP_TEST_DOCUMENT", `---
id: demo-1
title: CLI binding
status: DRAFT
parents: []
func:
  execute: {}
---

## Execute {#func:execute}
`)
	if _, err := executeForTest(t, "new", "--author.name", "human"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeForTest(t, "ready", "1", "--author.name", "human"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(repo+"/execute.go", []byte("package execute\n\ntype Runner struct{}\nfunc Run() {}\nfunc (Runner) Execute() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(repo+"/test_execute.py", []byte("def test_execute():\n    pass\n\nclass TestExecute:\n    def test_method(self):\n        pass\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := executeForTest(t, "bind", "add", "1", "execute",
		"--impl", "execute.go:Run", "--impl", "execute.go:Runner:Execute",
		"--author.name", "codex", "--author.agent", "true"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeForTest(t, "bind", "add", "1", "execute",
		"--test", "test_execute.py:test_execute", "--test", "test_execute.py:TestExecute:test_method",
		"--author.name", "codex", "--author.agent", "true"); err != nil {
		t.Fatal(err)
	}
	project, _ := openProject()
	spec, err := project.Store.SpecificationByID(context.Background(), "demo-1")
	project.Store.Close()
	if err != nil || spec.Status != sloop.StatusDraft || spec.Dirty || len(spec.Features["execute"].Impls) != 2 {
		t.Fatalf("binding change did not create a recorded DRAFT boundary: %#v, %v", spec, err)
	}
	if _, err := executeForTest(t, "implemented", "1",
		"--reason", "Implemented Run and Runner.Execute and registered their implementation and test symbols.",
		"--author.name", "codex", "--author.agent", "true"); err != nil {
		t.Fatal(err)
	}

	output, err := executeForTest(t, "bind", "list", "1", "execute", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var listed struct {
		Feature string                    `json:"feature"`
		Impls   []sloop.BindingResolution `json:"impls"`
		Tests   []sloop.BindingResolution `json:"tests"`
	}
	if err := json.Unmarshal([]byte(output), &listed); err != nil {
		t.Fatal(err)
	}
	if listed.Feature != "execute" || len(listed.Impls) != 2 || len(listed.Tests) != 2 {
		t.Fatalf("unexpected binding list: %#v", listed)
	}

	if _, err := executeForTest(t, "bind", "remove", "1", "execute",
		"--impl", "execute.go:Run", "--author.name", "human"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeForTest(t, "bind", "remove", "1", "execute",
		"--impl", "execute.go:Run", "--author.name", "human"); err != nil {
		t.Fatalf("idempotent removal failed: %v", err)
	}
	view, err := executeForTest(t, "view", "1")
	if err != nil || strings.Contains(view, "- execute.go:Run\n") || !strings.Contains(view, "execute.go:Runner:Execute") {
		t.Fatalf("Markdown and CLI did not share binding state: %q, %v", view, err)
	}
	if _, err := executeForTest(t, "bind", "remove", "1", "execute", "--impl", "malformed"); err == nil {
		t.Fatal("malformed locator removal must fail")
	}
}

func TestBindingAddRejectsUnresolvedSymbols(t *testing.T) {
	repo := setupProject(t)
	t.Setenv("SLOOP_TEST_DOCUMENT", `---
id: demo-1
title: Reject unresolved
status: DRAFT
parents: []
---
`)
	if _, err := executeForTest(t, "new", "--author.name", "human"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeForTest(t, "bind", "add", "1", "feature", "--impl", "missing.go:Run"); err == nil || !strings.Contains(err.Error(), "could not be resolved") {
		t.Fatalf("missing symbol was accepted: %v", err)
	}
	if err := os.WriteFile(repo+"/feature.txt", []byte("Run"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := executeForTest(t, "bind", "add", "1", "feature", "--impl", "feature.txt:Run"); err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("unsupported symbol was accepted: %v", err)
	}
	if err := os.WriteFile(repo+"/duplicate.py", []byte("def run(): pass\ndef run(): pass\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := executeForTest(t, "bind", "add", "1", "feature", "--impl", "duplicate.py:run"); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("ambiguous symbol was accepted: %v", err)
	}
}

func TestHistoricalRevisionWithMissingBindingRestoresAsUnrecordedDraft(t *testing.T) {
	repo := setupProject(t)
	if err := os.WriteFile(repo+"/old.go", []byte("package old\n\nfunc Run() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	original := `---
id: demo-1
title: Historical binding
status: READY
parents: []
func:
  old:
    impls:
      - old.go:Run
    tests: []
---

## Old {#func:old}

Original behavior.
`
	t.Setenv("SLOOP_TEST_DOCUMENT", original)
	if _, err := executeForTest(t, "new", "--author.name", "human"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SLOOP_TEST_DOCUMENT", strings.Replace(original, "Original behavior.", "Changed behavior.", 1))
	if _, err := executeForTest(t, "edit", "1", "--author.name", "human"); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(repo + "/old.go"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SLOOP_TEST_DOCUMENT", strings.Replace(original, "status: READY", "status: DRAFT", 1))
	if _, err := executeWithInputForTest(t, "y\n", "edit", "1#1", "--author.name", "human"); err != nil {
		t.Fatal(err)
	}

	project, err := openProject()
	if err != nil {
		t.Fatal(err)
	}
	spec, err := project.Store.SpecificationByID(context.Background(), "demo-1")
	if err != nil {
		project.Store.Close()
		t.Fatal(err)
	}
	revisions, err := project.Store.Revisions(context.Background(), spec.UUID)
	project.Store.Close()
	if err != nil {
		t.Fatal(err)
	}
	if spec.Status != sloop.StatusDraft || !spec.Dirty || len(revisions) != 2 || spec.Features["old"].Impls[0] != "old.go:Run" {
		t.Fatalf("historical binding was not restored as an unrecorded DRAFT: spec=%#v revisions=%#v", spec, revisions)
	}
	if _, err := executeForTest(t, "ready", "1", "--author.name", "human"); err == nil || !strings.Contains(err.Error(), "MISSING") {
		t.Fatalf("unresolved restored binding should block READY: %v", err)
	}
}
