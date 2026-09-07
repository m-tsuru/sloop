package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIndexRebuildRecoversFromDeletedDatabase(t *testing.T) {
	setupProject(t)
	createDraftSpecification(t)
	if _, err := executeForTest(t, "ready", "1", "--author.name", "human"); err != nil {
		t.Fatal(err)
	}
	project, err := openProject()
	if err != nil {
		t.Fatal(err)
	}
	dbPath := project.Store.Paths.DB
	project.Store.Close()
	if err := os.Remove(dbPath); err != nil {
		t.Fatal(err)
	}
	output, err := executeForTest(t, "index", "rebuild")
	if err != nil {
		t.Fatal(err)
	}
	if output != "Rebuilt index from 1 revision objects.\n" {
		t.Fatalf("unexpected output: %q", output)
	}
	project, _ = openProject()
	defer project.Store.Close()
	spec, err := project.Store.SpecificationByID(context.Background(), "demo-1")
	if err != nil || spec.Dirty || len(spec.HeadHash) != 64 {
		t.Fatalf("specification not recovered: %#v, %v", spec, err)
	}
	revisions, err := project.Store.Revisions(context.Background(), spec.UUID)
	if err != nil || len(revisions) != 1 || revisions[0].Hash != spec.HeadHash {
		t.Fatalf("revisions not recovered: %#v, %v", revisions, err)
	}
}

func TestGarbageCollectionOnlyRemovesCache(t *testing.T) {
	setupProject(t)
	createDraftSpecification(t)
	if _, err := executeForTest(t, "ready", "1"); err != nil {
		t.Fatal(err)
	}
	project, _ := openProject()
	cacheFile := project.Store.Paths.Cache + "/generated/context"
	objectRoot := project.Store.Paths.Objects
	if err := os.MkdirAll(project.Store.Paths.Cache+"/generated", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cacheFile, []byte("cache"), 0o644); err != nil {
		t.Fatal(err)
	}
	project.Store.Close()
	output, err := executeForTest(t, "gc")
	if err != nil || !strings.Contains(output, "Removed 1") {
		t.Fatalf("unexpected gc: %q %v", output, err)
	}
	if _, err := os.Stat(cacheFile); !os.IsNotExist(err) {
		t.Fatalf("cache was not removed: %v", err)
	}
	entries := 0
	_ = filepathWalkFiles(objectRoot, func() { entries++ })
	if entries == 0 {
		t.Fatal("recorded revision object was removed")
	}
}

func filepathWalkFiles(root string, found func()) error {
	return filepath.WalkDir(root, func(_ string, entry os.DirEntry, err error) error {
		if err == nil && !entry.IsDir() {
			found()
		}
		return err
	})
}
