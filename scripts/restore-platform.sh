#!/bin/sh
# Usage, from the installation directory: scripts/restore-platform.sh kymber-<id>.sql.gz kymber-encryption-key.txt
set -eu

if [ "$#" -ne 2 ] || [ ! -f "$1" ] || [ ! -f "$2" ]; then
	printf 'Usage: %s <backup.sql.gz> <encryption-key-file>\n' "$0" >&2
	exit 1
fi
dump=$(cd "$(dirname "$1")" && pwd)/$(basename "$1")
key=$(tr -d ' \r\n' < "$2")
if ! printf '%s' "$key" | grep -Eq '^[0-9a-fA-F]{64}$'; then
	printf 'The key file must hold the 64-character key downloaded from Kymber.\n' >&2
	exit 1
fi
gunzip -t "$dump"

cd "$(dirname "$0")/.."
printf 'This replaces every project, app and user in this Kymber with the backup. Type "restore" to continue: '
read -r answer
[ "$answer" = restore ] || exit 1

docker compose stop backend server
docker compose up -d --wait mariadb
printf '%s' "$key" | docker compose run --rm --no-deps -T secrets-init sh -c 'rm -f /run/kymber-secrets/service-encryption-key && cat > /run/kymber-secrets/service-encryption-key && chown 65532:65532 /run/kymber-secrets/service-encryption-key && chmod 0400 /run/kymber-secrets/service-encryption-key'
gunzip -c "$dump" | docker compose exec -T mariadb sh -c 'exec mariadb -uroot -p"$MARIADB_ROOT_PASSWORD"'
docker compose start backend server
printf 'Restored. Sign in with the accounts from the backup.\n'
