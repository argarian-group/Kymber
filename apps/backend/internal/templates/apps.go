package templates

func applicationPresets() []Template {
	return []Template{
		{Engine: "nextcloud", Category: "application", Image: "docker.io/library/nextcloud", DefaultVersion: "latest", Port: 80, DataPath: "/var/www/html", Fields: []Field{textField("NEXTCLOUD_ADMIN_USER", "Admin username", "kymber"), passwordField("NEXTCLOUD_ADMIN_PASSWORD", "Admin password"), textField("SQLITE_DATABASE", "SQLite database file", "nextcloud")}},
		{Engine: "gitea", Category: "application", Image: "docker.io/gitea/gitea", DefaultVersion: "latest", Port: 3000, DataPath: "/data"},
		{Engine: "n8n", Category: "application", Image: "docker.io/n8nio/n8n", DefaultVersion: "latest", Port: 5678, DataPath: "/home/node/.n8n", Fields: []Field{passwordField("N8N_ENCRYPTION_KEY", "Encryption key")}},
		{Engine: "vikunja", Category: "application", Image: "docker.io/vikunja/vikunja", DefaultVersion: "latest", Port: 3456, DataPath: "/db", Fields: []Field{passwordField("VIKUNJA_SERVICE_SECRET", "Service secret"), textField("VIKUNJA_SERVICE_PUBLICURL", "Public URL", "http://localhost:3456/")}},
		{Engine: "kaneo", Category: "application", Image: "docker.io/kyantech/kaneo", DefaultVersion: "latest", Port: 5173},
		{Engine: "passbolt", Category: "application", Image: "docker.io/passbolt/passbolt", DefaultVersion: "latest", Port: 80, DataPath: "/etc/passbolt/gpg"},
		{Engine: "vaultwarden", Category: "application", Image: "docker.io/vaultwarden/server", DefaultVersion: "latest", Port: 80, DataPath: "/data"},
		{Engine: "jellyfin", Category: "application", Image: "docker.io/jellyfin/jellyfin", DefaultVersion: "latest", Port: 8096, DataPath: "/config"},
		{Engine: "mealie", Category: "application", Image: "docker.io/hkotel/mealie", DefaultVersion: "latest", Port: 9000, DataPath: "/app/data"},
		{Engine: "paperless-ngx", Category: "application", Image: "docker.io/paperlessngx/paperless-ngx", DefaultVersion: "latest", Port: 8000, DataPath: "/usr/src/paperless/data"},
		{Engine: "freshrss", Category: "application", Image: "docker.io/linuxserver/freshrss", DefaultVersion: "latest", Port: 80, DataPath: "/config"},
		{Engine: "syncthing", Category: "application", Image: "docker.io/syncthing/syncthing", DefaultVersion: "latest", Port: 8384, DataPath: "/var/syncthing"},
		{Engine: "home-assistant", Category: "application", Image: "docker.io/homeassistant/home-assistant", DefaultVersion: "stable", Port: 8123, DataPath: "/config"},
		{Engine: "filebrowser", Category: "application", Image: "docker.io/filebrowser/filebrowser", DefaultVersion: "latest", Port: 80, DataPath: "/srv"},
		{Engine: "stirling-pdf", Category: "application", Image: "docker.io/stirlingtools/stirling-pdf", DefaultVersion: "latest", Port: 8080, DataPath: "/configs"},
		{Engine: "changedetection", Category: "application", Image: "docker.io/dgtlmoon/changedetection.io", DefaultVersion: "latest", Port: 5000, DataPath: "/datastore"},
		{Engine: "linkding", Category: "application", Image: "docker.io/sissbruecker/linkding", DefaultVersion: "latest", Port: 9090, DataPath: "/etc/linkding/data"},
		{Engine: "calibre-web", Category: "application", Image: "docker.io/linuxserver/calibre-web", DefaultVersion: "latest", Port: 8083, DataPath: "/config"},
		{Engine: "transmission", Category: "application", Image: "docker.io/linuxserver/transmission", DefaultVersion: "latest", Port: 9091, DataPath: "/config"},
		{Engine: "qbittorrent", Category: "application", Image: "docker.io/qbittorrentofficial/qbittorrent-nox", DefaultVersion: "latest", Port: 8080, DataPath: "/config"},
		{Engine: "gonic", Category: "application", Image: "docker.io/sentriz/gonic", DefaultVersion: "latest", Port: 80, DataPath: "/data"},
		{Engine: "homer", Category: "application", Image: "docker.io/b4bz/homer", DefaultVersion: "latest", Port: 8080, DataPath: "/www/assets"},
		{Engine: "it-tools", Category: "application", Image: "docker.io/corentinth/it-tools", DefaultVersion: "latest", Port: 80},
		{Engine: "glances", Category: "application", Image: "docker.io/nicolargo/glances", DefaultVersion: "latest", Port: 61208, DataPath: "/glances/conf"},
		{Engine: "kavita", Category: "application", Image: "docker.io/linuxserver/kavita", DefaultVersion: "latest", Port: 5000, DataPath: "/config"},
	}
}
