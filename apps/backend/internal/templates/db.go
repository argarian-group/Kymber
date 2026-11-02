package templates

func databasePresets() []Template {
	return []Template{
		{
			Engine: "mysql", Category: "database", Image: "docker.io/library/mysql", DefaultVersion: "latest", Port: 3306, DataPath: "/var/lib/mysql",
			HealthCmd: `mysqladmin ping -h 127.0.0.1 -uroot -p"$MYSQL_ROOT_PASSWORD" --silent`,
			Fields: []Field{
				{Name: "MYSQL_DATABASE", Label: "Database", Type: "text", Required: true, Default: "kymber", Max: 64, Pattern: `^[A-Za-z0-9_]+$`},
				{Name: "MYSQL_USER", Label: "Username", Type: "text", Required: true, Default: "kymber", Max: 32, Pattern: `^[A-Za-z0-9_]+$`},
				passwordField("MYSQL_PASSWORD", "Password"),
				passwordField("MYSQL_ROOT_PASSWORD", "Root password"),
			},
			Connection: Connection{Scheme: "mysql", DatabaseField: "MYSQL_DATABASE", UserField: "MYSQL_USER", PasswordField: "MYSQL_PASSWORD"},
		},
		{
			Engine: "mariadb", Category: "database", Image: "docker.io/library/mariadb", DefaultVersion: "latest", Port: 3306, DataPath: "/var/lib/mysql",
			HealthCmd: `mariadb-admin ping -h 127.0.0.1 -uroot -p"$MARIADB_ROOT_PASSWORD" --silent`,
			Fields: []Field{
				textField("MARIADB_DATABASE", "Database", "kymber"),
				textField("MARIADB_USER", "Username", "kymber"),
				passwordField("MARIADB_PASSWORD", "Password"),
				passwordField("MARIADB_ROOT_PASSWORD", "Root password"),
			},
			Connection: Connection{Scheme: "mysql", DatabaseField: "MARIADB_DATABASE", UserField: "MARIADB_USER", PasswordField: "MARIADB_PASSWORD"},
		},
		{
			Engine: "postgres", Category: "database", Image: "docker.io/library/postgres", DefaultVersion: "latest", Port: 5432, DataPath: "/var/lib/postgresql",
			HealthCmd:  `pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB"`,
			Fields:     []Field{textField("POSTGRES_DB", "Database", "kymber"), textField("POSTGRES_USER", "Username", "kymber"), passwordField("POSTGRES_PASSWORD", "Password")},
			Connection: Connection{Scheme: "postgres", DatabaseField: "POSTGRES_DB", UserField: "POSTGRES_USER", PasswordField: "POSTGRES_PASSWORD"},
		},
		{
			Engine: "percona", Category: "database", Image: "docker.io/percona/percona-server", DefaultVersion: "latest", Port: 3306, DataPath: "/var/lib/mysql",
			HealthCmd:  `mysqladmin ping -h 127.0.0.1 -uroot -p"$MYSQL_ROOT_PASSWORD" --silent`,
			Fields:     []Field{textField("MYSQL_DATABASE", "Database", "kymber"), textField("MYSQL_USER", "Username", "kymber"), passwordField("MYSQL_PASSWORD", "Password"), passwordField("MYSQL_ROOT_PASSWORD", "Root password")},
			Connection: Connection{Scheme: "mysql", DatabaseField: "MYSQL_DATABASE", UserField: "MYSQL_USER", PasswordField: "MYSQL_PASSWORD"},
		},
		{
			Engine: "timescale", Category: "database", Image: "docker.io/timescale/timescaledb", DefaultVersion: "latest-pg17", Port: 5432, DataPath: "/var/lib/postgresql",
			HealthCmd:  `pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB"`,
			Fields:     []Field{textField("POSTGRES_DB", "Database", "kymber"), textField("POSTGRES_USER", "Username", "kymber"), passwordField("POSTGRES_PASSWORD", "Password")},
			Connection: Connection{Scheme: "postgres", DatabaseField: "POSTGRES_DB", UserField: "POSTGRES_USER", PasswordField: "POSTGRES_PASSWORD"},
		},
		{
			Engine: "mongodb", Category: "database", Image: "docker.io/library/mongo", DefaultVersion: "latest", Port: 27017, DataPath: "/data/db",
			HealthCmd: `mongosh --quiet -u "$MONGO_INITDB_ROOT_USERNAME" -p "$MONGO_INITDB_ROOT_PASSWORD" --authenticationDatabase admin --eval 'db.adminCommand({ping:1})'`,
			Fields:    []Field{textField("MONGO_INITDB_ROOT_USERNAME", "Root username", "kymber"), passwordField("MONGO_INITDB_ROOT_PASSWORD", "Root password")},
		},
		{
			Engine: "couchdb", Category: "database", Image: "docker.io/library/couchdb", DefaultVersion: "latest", Port: 5984, DataPath: "/opt/couchdb/data",
			HealthCmd: `curl -fsS -u "$COUCHDB_USER:$COUCHDB_PASSWORD" http://127.0.0.1:5984/_up`,
			Fields:    []Field{textField("COUCHDB_USER", "Admin username", "kymber"), passwordField("COUCHDB_PASSWORD", "Admin password")},
		},
		{
			Engine: "arangodb", Category: "database", Image: "docker.io/library/arangodb", DefaultVersion: "latest", Port: 8529, DataPath: "/var/lib/arangodb3",
			HealthCmd: `curl -fsS -u root:"$ARANGO_ROOT_PASSWORD" http://127.0.0.1:8529/_api/version`,
			Fields:    []Field{passwordField("ARANGO_ROOT_PASSWORD", "Root password")},
		},
		{
			Engine: "neo4j", Category: "database", Image: "docker.io/library/neo4j", DefaultVersion: "latest", Port: 7687, DataPath: "/data",
			Fields: []Field{{Name: "NEO4J_AUTH", Label: "Credentials (neo4j/<password>)", Type: "password", Required: true, Secret: true, Min: 8, Max: 128, Pattern: `^neo4j/.{8,}$`}},
		},
		{
			Engine: "redis", Category: "database", Image: "docker.io/library/redis", DefaultVersion: "latest", Port: 6379, DataPath: "/data",
			HealthCmd:  `redis-cli -a "$REDIS_PASSWORD" ping | grep PONG`,
			Fields:     []Field{passwordField("REDIS_PASSWORD", "Password")},
			Connection: Connection{Scheme: "redis", PasswordField: "REDIS_PASSWORD"},
		},
		{
			Engine: "valkey", Category: "database", Image: "docker.io/valkey/valkey", DefaultVersion: "latest", Port: 6379, DataPath: "/data",
			HealthCmd: `valkey-cli -a "$VALKEY_PASSWORD" ping | grep PONG`,
			Fields:    []Field{passwordField("VALKEY_PASSWORD", "Password")},
		},
		{
			Engine: "keydb", Category: "database", Image: "docker.io/eqalpha/keydb", DefaultVersion: "latest", Port: 6379, DataPath: "/data",
			HealthCmd: `keydb-cli -a "$KEYDB_PASSWORD" ping | grep PONG`,
			Fields:    []Field{passwordField("KEYDB_PASSWORD", "Password")},
		},
		{
			Engine: "dragonfly", Category: "database", Image: "docker.io/dragonflydb/dragonfly", DefaultVersion: "latest", Port: 6379, DataPath: "/data",
			Fields: []Field{passwordField("DRAGONFLY_PASSWORD", "Password")},
		},
		{Engine: "memcached", Category: "database", Image: "docker.io/library/memcached", DefaultVersion: "latest", Port: 11211},
		{Engine: "cassandra", Category: "database", Image: "docker.io/library/cassandra", DefaultVersion: "latest", Port: 9042, DataPath: "/var/lib/cassandra"},
		{
			Engine: "scylladb", Category: "database", Image: "docker.io/scylladb/scylla", DefaultVersion: "latest", Port: 9042, DataPath: "/var/lib/scylla",
			HealthCmd: "nodetool status",
		},
		{Engine: "yugabyte", Category: "database", Image: "docker.io/yugabytedb/yugabyte", DefaultVersion: "latest", Port: 5433, DataPath: "/home/yugabyte/yb_data"},
	}
}
