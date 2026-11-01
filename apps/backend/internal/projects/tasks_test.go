package projects

import (
	"strings"
	"testing"
)

func TestNormalizeTask(t *testing.T) {
	date := func(value string) *string { return &value }
	valid := func(change func(*taskPayload)) taskPayload {
		input := taskPayload{Title: "Ship it", Status: "todo", Priority: "normal"}
		change(&input)
		return input
	}
	tests := map[string]struct {
		input taskPayload
		ok    bool
	}{
		"minimal":           {valid(func(*taskPayload) {}), true},
		"all statuses":      {valid(func(p *taskPayload) { p.Status = "in_review" }), true},
		"due date":          {valid(func(p *taskPayload) { p.DueDate = date("2028-02-29") }), true},
		"200 characters":    {valid(func(p *taskPayload) { p.Title = strings.Repeat("é", 200) }), true},
		"blank title":       {valid(func(p *taskPayload) { p.Title = "   " }), false},
		"201 characters":    {valid(func(p *taskPayload) { p.Title = strings.Repeat("é", 201) }), false},
		"unknown status":    {valid(func(p *taskPayload) { p.Status = "blocked" }), false},
		"unknown priority":  {valid(func(p *taskPayload) { p.Priority = "low" }), false},
		"impossible date":   {valid(func(p *taskPayload) { p.DueDate = date("2027-02-29") }), false},
		"date with time":    {valid(func(p *taskPayload) { p.DueDate = date("2027-01-01T00:00:00Z") }), false},
		"empty date string": {valid(func(p *taskPayload) { p.DueDate = date("") }), false},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			message := normalizeTask(&test.input)
			if (message == "") != test.ok {
				t.Fatalf("normalizeTask() = %q, want ok = %v", message, test.ok)
			}
		})
	}

	input := taskPayload{Title: "  Trim me  ", Status: "done", Priority: "high"}
	if message := normalizeTask(&input); message != "" || input.Title != "Trim me" {
		t.Fatalf("normalizeTask() = %q, title = %q", message, input.Title)
	}
}
