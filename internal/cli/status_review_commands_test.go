package cli

import (
	"context"
	"strings"
	"testing"

	"github.com/m-tsuru/sloop/internal/sloop"
)

func createDraftSpecification(t *testing.T) {
	t.Helper()
	t.Setenv("SLOOP_TEST_DOCUMENT", `---
id: demo-1
title: Status test
status: DRAFT
parents: []
---

## Goal {#goal}
Test transitions.
`)
	if _, err := executeForTest(t, "new", "--author.name", "human"); err != nil {
		t.Fatal(err)
	}
}

func TestHumanAndAgentStatusTransitionsCreateRevisions(t *testing.T) {
	setupProject(t)
	createDraftSpecification(t)
	if _, err := executeForTest(t, "ready", "1", "--author.name", "human"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeForTest(t, "implemented", "demo-1", "--reason", "Implemented parser and its tests.", "--author.name", "bot", "--author.agent"); err != nil {
		t.Fatal(err)
	}
	project, _ := openProject()
	defer project.Store.Close()
	spec, err := project.Store.SpecificationByID(context.Background(), "demo-1")
	if err != nil {
		t.Fatal(err)
	}
	revisions, err := project.Store.Revisions(context.Background(), spec.UUID)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Status != sloop.StatusImplemented || len(revisions) != 2 || revisions[0].Status != sloop.StatusReady || revisions[1].Status != sloop.StatusImplemented {
		t.Fatalf("unexpected transition history: %#v %#v", spec, revisions)
	}
}

func TestAgentTransitionConstraints(t *testing.T) {
	setupProject(t)
	createDraftSpecification(t)
	if _, err := executeForTest(t, "ready", "1", "--author.name", "human"); err != nil {
		t.Fatal(err)
	}
	_, err := executeForTest(t, "complete", "1", "--reason", "Everything is finished.", "--author.name", "bot", "--author.agent")
	if err == nil || !strings.Contains(err.Error(), "cannot set") {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = executeForTest(t, "implemented", "1", "--author.name", "bot", "--author.agent")
	if err == nil || !strings.Contains(err.Error(), "require a reason") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReviewsAreRevisionScopedAndForceReadyCannotBeRejected(t *testing.T) {
	setupProject(t)
	createDraftSpecification(t)
	if _, err := executeForTest(t, "ready", "1", "--author.name", "human"); err != nil {
		t.Fatal(err)
	}
	if _, err := executeForTest(t, "reject", "1", "--reason", "Deletion behavior is unspecified.", "--author.name", "bot", "--author.agent"); err != nil {
		t.Fatal(err)
	}
	project, _ := openProject()
	spec, _ := project.Store.SpecificationByID(context.Background(), "demo-1")
	reviews, _ := project.Store.Reviews(context.Background(), spec.UUID)
	project.Store.Close()
	if len(reviews) != 1 || reviews[0].RevisionHash != spec.HeadHash || spec.Status != sloop.StatusReady {
		t.Fatalf("unexpected review: %#v, spec %#v", reviews, spec)
	}
	if _, err := executeForTest(t, "force-ready", "1", "--author.name", "human"); err != nil {
		t.Fatal(err)
	}
	_, err := executeForTest(t, "reject", "1", "--reason", "Ambiguous.", "--author.name", "bot", "--author.agent")
	if err == nil || !strings.Contains(err.Error(), "cannot be rejected") {
		t.Fatalf("unexpected error: %v", err)
	}
}
