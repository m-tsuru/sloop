package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/m-tsuru/sloop/internal/sloop"
)

func setupProject(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	repo := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "xdg"))
	var out bytes.Buffer
	if err := Execute([]string{"init", "demo", repo}, &out, &out); err != nil {
		t.Fatal(err)
	}
	t.Chdir(repo)
	editor := filepath.Join(t.TempDir(), "editor")
	content := "#!/bin/sh\nprintf '%s' \"$SLOOP_TEST_DOCUMENT\" > \"$1\"\n"
	if err := os.WriteFile(editor, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SLOOP_EDITOR", editor)
	return repo
}

func executeForTest(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	err := Execute(args, &stdout, &stderr)
	return stdout.String(), err
}

func TestNewEditRecordAndLog(t *testing.T) {
	setupProject(t)
	t.Setenv("SLOOP_TEST_DOCUMENT", `---
id: "demo-1"
title: "Parse Markdown"
status: DRAFT
parents: []
---

## Goal {#goal}

Parse it.
`)
	if output, err := executeForTest(t, "new", "--author.name", "human"); err != nil {
		t.Fatal(err)
	} else if output != "Specification demo-1 is created.\n" {
		t.Fatalf("unexpected output %q", output)
	}

	project, err := openProject()
	if err != nil {
		t.Fatal(err)
	}
	spec, err := project.Store.SpecificationByID(context.Background(), "demo-1")
	project.Store.Close()
	if err != nil || !spec.Dirty || spec.HeadHash != "" {
		t.Fatalf("unexpected new spec: %#v, %v", spec, err)
	}

	t.Setenv("SLOOP_TEST_DOCUMENT", strings.Replace(sloop.RenderEditable(spec), "Parse it.", "Parse all Markdown.", 1))
	if _, err := executeForTest(t, "edit", "1", "--record", "--author.name", "human"); err != nil {
		t.Fatal(err)
	}
	log, err := executeForTest(t, "log", "demo-1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(log, "DRAFT") || strings.Count(log, "human") != 2 {
		t.Fatalf("expected two revisions, got:\n%s", log)
	}
}

func TestMeaningfulEditReturnsReadySpecificationToDraft(t *testing.T) {
	setupProject(t)
	t.Setenv("SLOOP_TEST_DOCUMENT", `---
id: demo-1
title: First
status: READY
parents: []
---

## Goal {#goal}
Before
`)
	if _, err := executeForTest(t, "new", "--author.name", "human"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SLOOP_TEST_DOCUMENT", `---
id: demo-1
title: Changed
status: READY
parents: []
---

## Goal {#goal}
After
`)
	if _, err := executeForTest(t, "edit", "1", "--author.name", "human"); err != nil {
		t.Fatal(err)
	}
	project, _ := openProject()
	defer project.Store.Close()
	spec, err := project.Store.SpecificationByID(context.Background(), "demo-1")
	if err != nil {
		t.Fatal(err)
	}
	if spec.Status != sloop.StatusDraft || spec.HeadHash == "" || spec.Dirty {
		t.Fatalf("meaningful edit should record DRAFT revision: %#v", spec)
	}
}

func TestAgentCannotChangeStatusThroughEditor(t *testing.T) {
	setupProject(t)
	t.Setenv("SLOOP_TEST_DOCUMENT", `---
id: demo-1
title: Agent
status: READY
parents: []
---
body
`)
	_, err := executeForTest(t, "new", "--author.name", "bot", "--author.agent")
	if err == nil || !strings.Contains(err.Error(), "agents cannot change") {
		t.Fatalf("unexpected error: %v", err)
	}
}
