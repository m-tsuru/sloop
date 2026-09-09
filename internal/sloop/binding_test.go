package sloop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeBindingFixture(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestGoSymbolResolution(t *testing.T) {
	root := t.TempDir()
	writeBindingFixture(t, root, "internal/lint/lint.go", `package lint

func LintMarkdown() {}
func TestHelper() {}

type MarkdownLinter struct{}
func (MarkdownLinter) Lint() {}
func (*MarkdownLinter) PointerLint() {}
`)
	for _, locator := range []string{
		"internal/lint/lint.go:LintMarkdown",
		"internal/lint/lint.go:TestHelper",
		"internal/lint/lint.go:MarkdownLinter:Lint",
		"internal/lint/lint.go:MarkdownLinter:PointerLint",
	} {
		resolution, err := ResolveSymbol(root, locator)
		if err != nil {
			t.Fatal(err)
		}
		if resolution.Status != BindingResolved {
			t.Fatalf("%s: %#v", locator, resolution)
		}
	}
	resolution, err := ResolveSymbol(root, "internal/lint/lint.go:Missing")
	if err != nil || resolution.Status != BindingMissing {
		t.Fatalf("unexpected missing resolution: %#v, %v", resolution, err)
	}
}

func TestPythonSymbolResolutionAndAmbiguity(t *testing.T) {
	root := t.TempDir()
	writeBindingFixture(t, root, "sloop/lint.py", `"""def ignored_docstring(): pass"""

def lint_markdown():
    def nested():
        pass

async def lint_async():
    pass

class MarkdownLinter:
    def lint(self):
        pass

    @staticmethod
    def lint_static():
        pass

class Duplicate:
    def run(self):
        pass
    def run(self):
        pass
`)
	for _, locator := range []string{
		"sloop/lint.py:lint_markdown",
		"sloop/lint.py:lint_async",
		"sloop/lint.py:MarkdownLinter:lint",
		"sloop/lint.py:MarkdownLinter:lint_static",
	} {
		resolution, err := ResolveSymbol(root, locator)
		if err != nil {
			t.Fatal(err)
		}
		if resolution.Status != BindingResolved {
			t.Fatalf("%s: %#v", locator, resolution)
		}
	}
	nested, err := ResolveSymbol(root, "sloop/lint.py:nested")
	if err != nil || nested.Status != BindingMissing {
		t.Fatalf("nested function must not resolve as a module function: %#v, %v", nested, err)
	}
	ambiguous, err := ResolveSymbol(root, "sloop/lint.py:Duplicate:run")
	if err != nil || ambiguous.Status != BindingAmbiguous {
		t.Fatalf("duplicate method must be ambiguous: %#v, %v", ambiguous, err)
	}
}

func TestResolutionDistinguishesMissingAndUnsupported(t *testing.T) {
	root := t.TempDir()
	writeBindingFixture(t, root, "feature.txt", "feature")
	missing, err := ResolveSymbol(root, "missing.go:Missing")
	if err != nil || missing.Status != BindingMissing {
		t.Fatalf("unexpected missing result: %#v, %v", missing, err)
	}
	unsupported, err := ResolveSymbol(root, "feature.txt:Feature")
	if err != nil || unsupported.Status != BindingUnsupported {
		t.Fatalf("unexpected unsupported result: %#v, %v", unsupported, err)
	}
}

func TestFeatureBindingNormalizationAndSectionValidation(t *testing.T) {
	document := `---
id: demo-1
title: Feature
status: DRAFT
parents: []
func:
  empty: {}
  nulls:
    impls:
    tests:
  arrays:
    impls: []
    tests: []
---

## Empty {#func:empty}
`
	doc, err := ParseEditable([]byte(document))
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"empty", "nulls", "arrays"} {
		binding := doc.Features[id]
		if binding.Impls == nil || binding.Tests == nil || len(binding.Impls) != 0 || len(binding.Tests) != 0 {
			t.Fatalf("%s was not normalized: %#v", id, binding)
		}
	}
	rendered := RenderEditable(Specification{
		ID: "demo-1", Title: "Feature", Status: StatusDraft, Parents: []string{},
		Features: doc.Features, Body: doc.Body,
	})
	if !strings.Contains(rendered, "impls: []") || !strings.Contains(rendered, "tests: []") {
		t.Fatalf("empty bindings were not rendered canonically:\n%s", rendered)
	}

	missingFrontMatter := strings.Replace(document, "  empty: {}\n", "", 1)
	if _, err := ParseEditable([]byte(missingFrontMatter)); err == nil || !strings.Contains(err.Error(), "no corresponding") {
		t.Fatalf("unexpected missing front matter error: %v", err)
	}
	invalidID := strings.Replace(document, "empty: {}", "Bad: {}", 1)
	if _, err := ParseEditable([]byte(invalidID)); err == nil || !strings.Contains(err.Error(), "invalid feature ID") {
		t.Fatalf("unexpected feature ID error: %v", err)
	}
	duplicateSection := strings.Replace(document, "## Empty {#func:empty}", "## Empty {#func:empty}\n\n## Again {#func:empty}", 1)
	if _, err := ParseEditable([]byte(duplicateSection)); err == nil || !strings.Contains(err.Error(), "more than one") {
		t.Fatalf("unexpected duplicate section error: %v", err)
	}
}

func TestFeatureBindingsParticipateInFormatTwoRevisionHash(t *testing.T) {
	base := Revision{
		FormatVersion: 2, ProjectID: "project", SpecificationUUID: "uuid",
		SpecificationID: "demo-1", Author: Author{Name: "human"}, Content: "body\n",
		Status: StatusDraft, Parents: []string{}, References: []Reference{},
		Features: FeatureBindings{},
	}
	first, err := revisionHash(base)
	if err != nil {
		t.Fatal(err)
	}
	base.Features = FeatureBindings{
		"feature": {Impls: []string{"main.go:Run"}, Tests: []string{}},
	}
	second, err := revisionHash(base)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("feature bindings did not change the revision hash")
	}
}
