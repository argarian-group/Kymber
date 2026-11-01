package projects

import (
	"reflect"
	"strings"
	"testing"
)

func TestNormalizeTask(t *testing.T) {
	date := func(value string) *string { return &value }
	valid := func(change func(*taskPayload)) taskPayload {
		input := taskPayload{Title: "Ship it", Priority: "normal"}
		change(&input)
		return input
	}
	tests := map[string]struct {
		input taskPayload
		ok    bool
	}{
		"minimal":              {valid(func(*taskPayload) {}), true},
		"due date":             {valid(func(p *taskPayload) { p.DueDate = date("2028-02-29") }), true},
		"200 characters":       {valid(func(p *taskPayload) { p.Title = strings.Repeat("é", 200) }), true},
		"long description":     {valid(func(p *taskPayload) { p.Description = strings.Repeat("é", 20000) }), true},
		"blank title":          {valid(func(p *taskPayload) { p.Title = "   " }), false},
		"201 characters":       {valid(func(p *taskPayload) { p.Title = strings.Repeat("é", 201) }), false},
		"too long description": {valid(func(p *taskPayload) { p.Description = strings.Repeat("é", 20001) }), false},
		"unknown priority":     {valid(func(p *taskPayload) { p.Priority = "low" }), false},
		"impossible date":      {valid(func(p *taskPayload) { p.DueDate = date("2027-02-29") }), false},
		"date with time":       {valid(func(p *taskPayload) { p.DueDate = date("2027-01-01T00:00:00Z") }), false},
		"empty date string":    {valid(func(p *taskPayload) { p.DueDate = date("") }), false},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			message := normalizeTask(&test.input)
			if (message == "") != test.ok {
				t.Fatalf("normalizeTask() = %q, want ok = %v", message, test.ok)
			}
		})
	}

	input := taskPayload{Title: "  Trim me  ", Priority: "high"}
	if message := normalizeTask(&input); message != "" || input.Title != "Trim me" {
		t.Fatalf("normalizeTask() = %q, title = %q", message, input.Title)
	}
}

func TestInsertAt(t *testing.T) {
	tests := []struct {
		position int
		want     []string
	}{
		{-1, []string{"x", "a", "b"}},
		{0, []string{"x", "a", "b"}},
		{1, []string{"a", "x", "b"}},
		{2, []string{"a", "b", "x"}},
		{99, []string{"a", "b", "x"}},
	}
	for _, test := range tests {
		if got := insertAt([]string{"a", "b"}, "x", test.position); !reflect.DeepEqual(got, test.want) {
			t.Errorf("insertAt(%d) = %v, want %v", test.position, got, test.want)
		}
	}
}

func TestSameIDs(t *testing.T) {
	tests := []struct {
		a, b []string
		want bool
	}{
		{[]string{"a", "b"}, []string{"b", "a"}, true},
		{nil, nil, true},
		{[]string{"a"}, []string{"a", "b"}, false},
		{[]string{"a", "a"}, []string{"a", "b"}, false},
		{[]string{"a", "b"}, []string{"a", "a"}, false},
	}
	for _, test := range tests {
		if got := sameIDs(test.a, test.b); got != test.want {
			t.Errorf("sameIDs(%v, %v) = %v, want %v", test.a, test.b, got, test.want)
		}
	}
}

func TestValidLabelColor(t *testing.T) {
	if !validLabelColor("teal") || validLabelColor("#fff") || validLabelColor("") {
		t.Fatal("validLabelColor accepted or rejected the wrong value")
	}
}
