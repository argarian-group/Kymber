package catalog

import (
	"kymber/backend/internal/templates"
	template0 "kymber/backend/internal/templates/docker"
	template1 "kymber/backend/internal/templates/mysql"
	template2 "kymber/backend/internal/templates/redis"
)

func All() []templates.Template {
	return []templates.Template{template0.Definition(), template1.Definition(), template2.Definition()}
}

func Get(engine string) (templates.Template, bool) {
	switch engine {
	case template0.Definition().Engine:
		return template0.Definition(), true
	case template1.Definition().Engine:
		return template1.Definition(), true
	case template2.Definition().Engine:
		return template2.Definition(), true
	default:
		return templates.Template{}, false
	}
}

func DockerArgs(engine string, config templates.Config) ([]string, bool) {
	switch engine {
	case template0.Definition().Engine:
		return template0.DockerArgs(config), true
	case template1.Definition().Engine:
		return template1.DockerArgs(config), true
	case template2.Definition().Engine:
		return template2.DockerArgs(config), true
	default:
		return nil, false
	}
}
