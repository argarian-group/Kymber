package templates

import "testing"

func TestParseRepository(t *testing.T) {
	cases := map[string]string{
		"redis":              "library/redis",
		"library/redis":      "library/redis",
		"bitnami/redis":      "bitnami/redis",
		"my-org/my_app.v2":   "my-org/my_app.v2",
		"":                   "",
		"Redis":              "",
		"-redis":             "",
		"a/b/c":              "",
		"redis:latest":       "",
		"../redis":           "",
		"ghcr.io/org":        "",
		"redis --privileged": "",
	}
	for input, want := range cases {
		got, err := ParseRepository(input)
		if got != want || (err == nil) != (want != "") {
			t.Errorf("ParseRepository(%q) = %q, %v; want %q", input, got, err, want)
		}
	}
}

func TestImageRefRejectsImagesOutsideDockerHub(t *testing.T) {
	cases := map[string]bool{
		"docker.io/library/redis": true,
		"docker.io/bitnami/redis": true,
		"ghcr.io/org/app":         false,
		"--privileged":            false,
		"":                        false,
	}
	for image, valid := range cases {
		if _, err := (Template{Image: image}).ImageRef("latest"); (err == nil) != valid {
			t.Errorf("ImageRef with image %q = %v, want valid=%v", image, err, valid)
		}
	}
}

func TestValidDataPath(t *testing.T) {
	cases := map[string]bool{
		"":               true,
		"/data":          true,
		"/var/lib/app_1": true,
		"/":              false,
		"data":           false,
		"/data:/etc":     false,
		"/../etc":        false,
		"/data dir":      false,
	}
	for path, valid := range cases {
		if got := ValidDataPath(path); got != valid {
			t.Errorf("ValidDataPath(%q) = %v, want %v", path, got, valid)
		}
	}
}

func TestForServiceOnlyChangesCustomTemplates(t *testing.T) {
	builtin := Template{Image: "docker.io/library/mysql", Port: 3306, DataPath: "/var/lib/mysql"}
	if got := builtin.ForService("docker.io/evil/app", 1, "/x"); got.Image != builtin.Image || got.Port != 3306 || got.DataPath != builtin.DataPath {
		t.Errorf("built-in template changed: %+v", got)
	}
	got := Template{Custom: true}.ForService("docker.io/bitnami/redis", 6379, "/data")
	if got.Image != "docker.io/bitnami/redis" || got.Port != 6379 || got.DataPath != "/data" {
		t.Errorf("custom template not resolved: %+v", got)
	}
}

func TestNormalizeCustomTemplateKeepsFreeFormVariables(t *testing.T) {
	custom := Template{Custom: true}
	got, err := Normalize(custom, Config{"APP_MODE": "prod", "EMPTY": ""})
	if err != nil || got["APP_MODE"] != "prod" {
		t.Fatalf("Normalize = %v, %v", got, err)
	}
	if _, kept := got["EMPTY"]; kept {
		t.Error("empty variable should be dropped")
	}
	for name, config := range map[string]Config{
		"bad name":   {"BAD-NAME": "x"},
		"line break": {"OK": "a\nINJECTED=1"},
	} {
		if _, err := Normalize(custom, config); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
	if _, err := Normalize(Template{}, Config{"APP_MODE": "prod"}); err == nil {
		t.Error("built-in templates must still reject unknown variables")
	}
}
