package cli

import (
	"reflect"
	"testing"
)

func TestNormalizeExplicitAuthorAgentBoolean(t *testing.T) {
	input := []string{"status", "1", "implemented", "--author.agent", "true", "--reason", "tested"}
	want := []string{"status", "1", "implemented", "--author.agent=true", "--reason", "tested"}
	if got := normalizeExplicitBoolArgs(input); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	bare := []string{"implemented", "1", "--author.agent"}
	if got := normalizeExplicitBoolArgs(bare); !reflect.DeepEqual(got, bare) {
		t.Fatalf("bare flag changed: %#v", got)
	}
}
