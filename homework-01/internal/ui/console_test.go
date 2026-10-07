package ui

import (
	"reflect"
	"strings"
	"testing"
)

func TestSplitArguments(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"", nil},
		{"add task 2026-10-20", []string{"add", "task", "2026-10-20"}},
		{`add "task with spaces" 2026-10-20`, []string{"add", "task with spaces", "2026-10-20"}},
		{`edit 1 --title 'quoted title'`, []string{"edit", "1", "--title", "quoted title"}},
		{`add "escaped \"quote\"" 2026-10-20`, []string{"add", `escaped "quote"`, "2026-10-20"}},
	}
	for _, test := range tests {
		got, err := splitArguments(test.input)
		if err != nil {
			t.Fatalf("split %q: %v", test.input, err)
		}
		if !reflect.DeepEqual(got, test.want) {
			t.Fatalf("split %q = %#v, want %#v", test.input, got, test.want)
		}
	}
}

func TestSplitArgumentsRejectsMalformedInput(t *testing.T) {
	for _, input := range []string{`add "unfinished`, `add trailing\`} {
		if _, err := splitArguments(input); err == nil {
			t.Fatalf("expected error for %q", input)
		}
	}
}

func TestParseDeadline(t *testing.T) {
	dateOnly, err := parseDeadline("2026-10-20")
	if err != nil {
		t.Fatal(err)
	}
	if got := dateOnly.Format("2006-01-02T15:04:05Z07:00"); got != "2026-10-20T00:00:00Z" {
		t.Fatalf("date-only deadline = %s", got)
	}
	if _, err := parseDeadline("tomorrow"); err == nil || !strings.Contains(err.Error(), "YYYY-MM-DD") {
		t.Fatalf("unexpected invalid-date error: %v", err)
	}
}
