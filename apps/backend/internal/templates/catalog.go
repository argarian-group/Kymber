package templates

func All() []Template {
	templates := make([]Template, 0)
	templates = append(templates, databasePresets()...)
	templates = append(templates, servicePresets()...)
	templates = append(templates, applicationPresets()...)
	templates = append(templates, Template{Engine: "docker", DefaultVersion: "latest", Custom: true, Fields: []Field{}})
	for i := range templates {
		if templates[i].Fields == nil {
			templates[i].Fields = []Field{}
		}
	}
	return templates
}

func Get(engine string) (Template, bool) {
	for _, template := range All() {
		if template.Engine == engine {
			return template, true
		}
	}
	return Template{}, false
}

func DockerArgs(engine string, _ Config) ([]string, bool) {
	if _, ok := Get(engine); !ok {
		return nil, false
	}
	switch engine {
	case "redis":
		return []string{"sh", "-c", `exec redis-server --appendonly yes --requirepass "$REDIS_PASSWORD"`}, true
	case "valkey":
		return []string{"sh", "-c", `exec valkey-server --appendonly yes --requirepass "$VALKEY_PASSWORD"`}, true
	case "keydb":
		return []string{"sh", "-c", `exec keydb-server /etc/keydb/keydb.conf --appendonly yes --requirepass "$KEYDB_PASSWORD"`}, true
	case "dragonfly":
		return []string{"sh", "-c", `exec dragonfly --logtostderr --requirepass="$DRAGONFLY_PASSWORD"`}, true
	case "nats":
		return []string{"sh", "-c", `exec nats-server --auth "$NATS_AUTH_TOKEN"`}, true
	case "typesense":
		return []string{"sh", "-c", `exec typesense-server --data-dir=/data --api-key="$TYPESENSE_API_KEY"`}, true
	case "victoriametrics":
		return []string{"-storageDataPath=/victoria-metrics-data"}, true
	case "minio":
		return []string{"server", "/data", "--console-address", ":9001"}, true
	case "pulsar":
		return []string{"bin/pulsar", "standalone"}, true
	case "scylladb":
		return []string{"--smp", "1", "--memory", "1G", "--developer-mode", "1"}, true
	case "yugabyte":
		return []string{"bin/yugabyted", "start", "--daemon=false", "--base_dir=/home/yugabyte/yb_data"}, true
	case "stalwart":
		// The image keeps its config in /etc/stalwart, a second volume Kymber does not mount.
		return []string{"--config", "/var/lib/stalwart/config.json"}, true
	case "cloudflared":
		return []string{"sh", "-c", `exec cloudflared tunnel --no-autoupdate run --token "$TUNNEL_TOKEN"`}, true
	default:
		return nil, true
	}
}

func passwordField(name, label string) Field {
	return Field{Name: name, Label: label, Type: "password", Required: true, Secret: true, Generate: true, Min: 16, Max: 128}
}

func textField(name, label, value string) Field {
	return Field{Name: name, Label: label, Type: "text", Required: true, Default: value, Max: 64}
}
