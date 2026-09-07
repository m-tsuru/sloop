package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestListStructuredOutputAndFilters(t *testing.T) {
	setupProject(t)
	createDraftSpecification(t)
	t.Setenv("SLOOP_TEST_DOCUMENT", `---
id: demo-2
title: Other feature
status: DRAFT
parents: []
---
## Goal {#goal}
Other.
`)
	if _, err := executeForTest(t, "new", "--author.name", "other"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeForTest(t, "ready", "1", "--author.name", "human"); err != nil {
		t.Fatal(err)
	}
	output, err := executeForTest(t, "list", "--json", "--filter", "status:ready", "--filter", "title:Status")
	if err != nil {
		t.Fatal(err)
	}
	var records []map[string]any
	if err := json.Unmarshal([]byte(output), &records); err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0]["id"] != "demo-1" {
		t.Fatalf("unexpected records: %#v", records)
	}
	if _, err := executeForTest(t, "list", "--json", "--csv"); err == nil {
		t.Fatal("expected mutually exclusive output error")
	}
}

func TestViewQueryAndContext(t *testing.T) {
	setupProject(t)
	t.Setenv("SLOOP_TEST_DOCUMENT", `---
id: demo-1
title: Context feature
status: DRAFT
parents: []
---

## Goal {#goal}

Keep context local.

## Specification {#specification}

Emit JSON.

## Done {#acceptance-criteria}

JSON parses.

## UI {#user-interface}

Plain text.
`)
	if _, err := executeForTest(t, "new", "--author.name", "human"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeForTest(t, "context", "1"); err == nil || !strings.Contains(err.Error(), "unrecorded working changes") {
		t.Fatalf("unexpected context error: %v", err)
	}
	working, err := executeForTest(t, "context", "1", "--working", "--json")
	if err != nil || !strings.Contains(working, `"recorded": false`) {
		t.Fatalf("unexpected working context: %q, %v", working, err)
	}
	if _, err := executeForTest(t, "ready", "1", "--author.name", "human"); err != nil {
		t.Fatal(err)
	}
	view, err := executeForTest(t, "view", "1")
	if err != nil || !strings.Contains(view, "revision-hash:") || !strings.Contains(view, "Keep context local.") {
		t.Fatalf("unexpected view: %q, %v", view, err)
	}
	section, err := executeForTest(t, "view", "1", "--section", "user-interface")
	if err != nil || section != "Plain text.\n" {
		t.Fatalf("unexpected section: %q, %v", section, err)
	}
	query, err := executeForTest(t, "query", "--section", "goal", "--filter", "status:READY")
	if err != nil || !strings.Contains(query, "Keep context local.") {
		t.Fatalf("unexpected query: %q, %v", query, err)
	}
	contextOutput, err := executeForTest(t, "context", "1", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal([]byte(contextOutput), &document); err != nil {
		t.Fatal(err)
	}
	if document["recorded"] != true || len(document["revision_hash"].(string)) != 64 || document["goal"] != "Keep context local." {
		t.Fatalf("unexpected context: %#v", document)
	}
}
