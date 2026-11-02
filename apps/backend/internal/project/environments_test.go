package project

import (
	"strings"
	"testing"
)

func TestEnvironmentName(t *testing.T) {
	cases := map[string]string{
		"production":            "production",
		"  Staging EU  ":        "Staging EU",
		"Produção":              "Produção",
		"":                      "",
		"   ":                   "",
		"a\nb":                  "",
		strings.Repeat("é", 64): strings.Repeat("é", 64),
		strings.Repeat("a", 65): "",
	}
	for input, want := range cases {
		if got := environmentName(input); got != want {
			t.Errorf("environmentName(%q) = %q, want %q", input, got, want)
		}
	}
}
