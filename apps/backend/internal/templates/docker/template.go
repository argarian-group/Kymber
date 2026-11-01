package docker

import "kymber/backend/internal/templates"

// Definition is the template for any public Docker Hub image; each service supplies the image details.
func Definition() templates.Template {
	return templates.Template{Engine: "docker", DefaultVersion: "latest", Custom: true, Fields: []templates.Field{}}
}

func DockerArgs(config templates.Config) []string {
	return nil
}
