package templates

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

type Config map[string]string

type Field struct {
	Name     string `json:"name"`
	Label    string `json:"label"`
	Type     string `json:"type"`
	Required bool   `json:"required,omitempty"`
	Secret   bool   `json:"secret,omitempty"`
	Default  string `json:"default,omitempty"`
	Generate bool   `json:"generate,omitempty"`
	Min      int    `json:"min,omitempty"`
	Max      int    `json:"max,omitempty"`
	Pattern  string `json:"pattern,omitempty"`
}

type Connection struct {
	Scheme        string `json:"scheme"`
	DatabaseField string `json:"database_field,omitempty"`
	UserField     string `json:"user_field,omitempty"`
	PasswordField string `json:"password_field"`
}

type Template struct {
	Engine         string     `json:"engine"`
	Image          string     `json:"image"`
	DefaultVersion string     `json:"default_version"`
	Port           int        `json:"port"`
	DataPath       string     `json:"data_path"`
	HealthCmd      string     `json:"-"`
	Fields         []Field    `json:"fields"`
	Connection     Connection `json:"connection"`
}

var environmentNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
var imageVersionPattern = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$`)

func (t Template) ImageRef(version string) (string, error) {
	if !imageVersionPattern.MatchString(version) {
		return "", errors.New("invalid image version")
	}
	return t.Image + ":" + version, nil
}

func (t Template) ContainerName(name, serviceID string) string {
	return fmt.Sprintf("%s-service-%s", name, serviceID)
}

func (t Template) VolumeName(serviceID string) string {
	return fmt.Sprintf("kymber-service-%s-data", serviceID)
}

func (t Template) InternalHost(serviceID string) string {
	return "kymber-service-" + serviceID
}

func (t Template) ConnectionURL(serviceID string, config Config) string {
	if t.Connection.Scheme == "" || config[t.Connection.PasswordField] == "" {
		return ""
	}
	user := ""
	if t.Connection.UserField != "" {
		user = config[t.Connection.UserField]
	}
	path := ""
	if t.Connection.DatabaseField != "" && config[t.Connection.DatabaseField] != "" {
		path = "/" + config[t.Connection.DatabaseField]
	}
	return (&url.URL{Scheme: t.Connection.Scheme, User: url.UserPassword(user, config[t.Connection.PasswordField]), Host: fmt.Sprintf("%s:%d", t.InternalHost(serviceID), t.Port), Path: path}).String()
}

func Normalize(t Template, config Config) (Config, error) {
	values := make(Config, len(t.Fields))
	known := make(map[string]Field, len(t.Fields))
	for _, field := range t.Fields {
		if !environmentNamePattern.MatchString(field.Name) {
			return nil, fmt.Errorf("invalid environment field name %q", field.Name)
		}
		known[field.Name] = field
	}
	for key := range config {
		if _, ok := known[key]; !ok {
			return nil, fmt.Errorf("unsupported configuration field %q", key)
		}
	}
	for _, field := range t.Fields {
		value := config[field.Name]
		if value == "" {
			value = field.Default
		}
		if value == "" && field.Generate {
			var err error
			value, err = randomValue()
			if err != nil {
				return nil, err
			}
		}
		if field.Required && value == "" {
			return nil, fmt.Errorf("%s is required", field.Name)
		}
		if strings.ContainsAny(value, "\r\n") {
			return nil, fmt.Errorf("%s cannot contain a line break", field.Name)
		}
		if field.Min > 0 && len(value) < field.Min || field.Max > 0 && len(value) > field.Max {
			return nil, fmt.Errorf("%s has invalid length", field.Name)
		}
		if value != "" && field.Pattern != "" {
			pattern, err := regexp.Compile(field.Pattern)
			if err != nil {
				return nil, fmt.Errorf("invalid template validation for %s: %w", field.Name, err)
			}
			if !pattern.MatchString(value) {
				return nil, fmt.Errorf("%s has invalid format", field.Name)
			}
		}
		if value != "" {
			values[field.Name] = value
		}
	}
	return values, nil
}

func randomValue() (string, error) {
	data := make([]byte, 32)
	if _, err := rand.Read(data); err != nil {
		return "", errors.New("could not generate template secret")
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}
