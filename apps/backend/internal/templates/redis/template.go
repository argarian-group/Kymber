package redis

import "kymber/backend/internal/templates"

func Definition() templates.Template {
	return templates.Template{
		Engine: "redis", Category: "database", Image: "docker.io/library/redis", DefaultVersion: "latest", Port: 6379, DataPath: "/data",
		HealthCmd:  `redis-cli -a "$REDIS_PASSWORD" ping | grep PONG`,
		Fields:     []templates.Field{{Name: "REDIS_PASSWORD", Label: "Password", Type: "password", Required: true, Secret: true, Generate: true, Min: 16, Max: 128}},
		Connection: templates.Connection{Scheme: "redis", PasswordField: "REDIS_PASSWORD"},
	}
}

func DockerArgs(config templates.Config) []string {
	return []string{"sh", "-c", `exec redis-server --appendonly yes --requirepass "$REDIS_PASSWORD"`}
}
