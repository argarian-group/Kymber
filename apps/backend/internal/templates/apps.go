package templates

func applicationPresets() []Template {
	return []Template{
		{Engine: "nextcloud", Category: "application", Image: "docker.io/library/nextcloud", DefaultVersion: "latest", Port: 80, DataPath: "/var/www/html", Fields: []Field{textField("NEXTCLOUD_ADMIN_USER", "Admin username", "kymber"), passwordField("NEXTCLOUD_ADMIN_PASSWORD", "Admin password"), textField("SQLITE_DATABASE", "SQLite database file", "nextcloud")}},
		{Engine: "gitea", Category: "application", Image: "docker.io/gitea/gitea", DefaultVersion: "latest", Port: 3000, DataPath: "/data"},
		{Engine: "n8n", Category: "application", Image: "docker.io/n8nio/n8n", DefaultVersion: "latest", Port: 5678, DataPath: "/home/node/.n8n", Fields: []Field{passwordField("N8N_ENCRYPTION_KEY", "Encryption key")}},
		{Engine: "vikunja", Category: "application", Image: "docker.io/vikunja/vikunja", DefaultVersion: "latest", Port: 3456, DataPath: "/db", Fields: []Field{passwordField("VIKUNJA_SERVICE_SECRET", "Service secret"), textField("VIKUNJA_SERVICE_PUBLICURL", "Public URL", "http://localhost:3456/")}},
		{Engine: "passbolt", Category: "application", Image: "docker.io/passbolt/passbolt", DefaultVersion: "latest", Port: 80, DataPath: "/etc/passbolt/gpg"},
		{Engine: "vaultwarden", Category: "application", Image: "docker.io/vaultwarden/server", DefaultVersion: "latest", Port: 80, DataPath: "/data"},
		{Engine: "jellyfin", Category: "application", Image: "docker.io/jellyfin/jellyfin", DefaultVersion: "latest", Port: 8096, DataPath: "/config"},
		{Engine: "freshrss", Category: "application", Image: "docker.io/linuxserver/freshrss", DefaultVersion: "latest", Port: 80, DataPath: "/config"},
		{Engine: "syncthing", Category: "application", Image: "docker.io/syncthing/syncthing", DefaultVersion: "latest", Port: 8384, DataPath: "/var/syncthing"},
		{Engine: "home-assistant", Category: "application", Image: "docker.io/homeassistant/home-assistant", DefaultVersion: "stable", Port: 8123, DataPath: "/config"},
		{Engine: "filebrowser", Category: "application", Image: "docker.io/filebrowser/filebrowser", DefaultVersion: "latest", Port: 80, DataPath: "/srv"},
		{Engine: "calibre-web", Category: "application", Image: "docker.io/linuxserver/calibre-web", DefaultVersion: "latest", Port: 8083, DataPath: "/config"},
		{Engine: "transmission", Category: "application", Image: "docker.io/linuxserver/transmission", DefaultVersion: "latest", Port: 9091, DataPath: "/config"},
		{Engine: "glances", Category: "application", Image: "docker.io/nicolargo/glances", DefaultVersion: "latest", Port: 61208, DataPath: "/glances/conf"},
		{
			Engine: "stalwart", Category: "application", Image: "docker.io/stalwartlabs/stalwart", DefaultVersion: "latest", Port: 8080, DataPath: "/var/lib/stalwart",
			Ports:  []int{25, 465, 587, 993},
			Fields: []Field{{Name: "STALWART_HOSTNAME", Label: "Mail server hostname", Type: "text", Required: true, Max: 253, Pattern: `^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+$`}},
		},
		{Engine: "kavita", Category: "application", Image: "docker.io/linuxserver/kavita", DefaultVersion: "latest", Port: 5000, DataPath: "/config"},
	}
}
