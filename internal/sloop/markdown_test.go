package sloop

import "testing"

func TestParseEditableAndSections(t *testing.T) {
	doc, err := ParseEditable([]byte("---\r\nid: x-1\r\ntitle: A\r\nstatus: draft\r\nparents: []\r\n---\r\n\r\n## Goal {#goal}\r\n\r\nDo it.\r\n\r\n## Extra {#custom}\r\nValue\r\n"))
	if err != nil {
		t.Fatal(err)
	}
	if doc.Status != "DRAFT" || doc.Body == "" {
		t.Fatalf("unexpected document: %#v", doc)
	}
	sections := Sections(doc.Body)
	if sections["goal"] != "Do it." || sections["custom"] != "Value" {
		t.Fatalf("unexpected sections: %#v", sections)
	}
}

func TestRevisionHashIgnoresTimestampAndParentOrder(t *testing.T) {
	base := Revision{ProjectID: "p", SpecificationUUID: "u", SpecificationID: "p-1", ParentRevisionHashes: []string{"b", "a"}, Author: Author{Name: "a"}, Content: "x", Status: StatusDraft, Parents: []string{}, References: []Reference{}}
	other := base
	other.ParentRevisionHashes = []string{"a", "b"}
	hash1, err := revisionHash(base)
	if err != nil {
		t.Fatal(err)
	}
	hash2, err := revisionHash(other)
	if err != nil {
		t.Fatal(err)
	}
	if hash1 != hash2 || len(hash1) != 64 {
		t.Fatalf("hashes differ: %s %s", hash1, hash2)
	}
}

func TestSectionsHandlesHeadingWithoutTrailingNewline(t *testing.T) {
	sections := Sections("## Empty {#empty}")
	if value, ok := sections["empty"]; !ok || value != "" {
		t.Fatalf("unexpected sections: %#v", sections)
	}
}
