package docker

import "kymber/backend/internal/templates"

func Definition() templates.Template {
	return templates.Template{Engine: "docker", DefaultVersion: "latest", Custom: true, Fields: []templates.Field{}}
}

func DockerArgs(config templates.Config) []string {
	return nil
}
