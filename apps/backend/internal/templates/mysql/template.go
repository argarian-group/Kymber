package mysql

import "kymber/backend/internal/templates"

func Definition() templates.Template {
	return templates.Template{
		Engine: "mysql", Image: "docker.io/library/mysql", DefaultVersion: "latest", Port: 3306, DataPath: "/var/lib/mysql",
		HealthCmd: `mysqladmin ping -h 127.0.0.1 -uroot -p"$MYSQL_ROOT_PASSWORD" --silent`,
		Fields: []templates.Field{
			{Name: "MYSQL_DATABASE", Label: "Database", Type: "text", Required: true, Default: "kymber", Max: 64, Pattern: `^[A-Za-z0-9_]+$`},
			{Name: "MYSQL_USER", Label: "Username", Type: "text", Required: true, Default: "kymber", Max: 32, Pattern: `^[A-Za-z0-9_]+$`},
			{Name: "MYSQL_PASSWORD", Label: "Password", Type: "password", Required: true, Secret: true, Generate: true, Min: 16, Max: 128},
			{Name: "MYSQL_ROOT_PASSWORD", Label: "Root password", Type: "password", Required: true, Secret: true, Generate: true, Min: 16, Max: 128},
		},
		Connection: templates.Connection{Scheme: "mysql", DatabaseField: "MYSQL_DATABASE", UserField: "MYSQL_USER", PasswordField: "MYSQL_PASSWORD"},
	}
}

func DockerArgs(config templates.Config) []string {
	return nil
}
