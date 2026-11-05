#!/bin/sh
set -eu
umask 077
set -C

if [ "$#" -ne 3 ] || [ ! -d "$1" ]; then
	printf 'usage: %s BUILD_CONTEXT OUTPUT_OCI_TAR SERVER_ID\n' "$0" >&2
	exit 2
fi

context=$(cd "$1" && pwd -P)
output=$2
case "$output" in
	/*) ;;
	*) output=$(pwd -P)/$output ;;
esac
if [ -e "$output" ]; then
	printf 'output path already exists\n' >&2
	exit 2
fi
server=$3
case "$server" in *[!0-9a-f]*|'') exit 2 ;; esac
[ "${#server}" -eq 6 ] || exit 2

command -v docker >/dev/null 2>&1 || { printf 'docker is required on the build host\n' >&2; exit 1; }
cleanup() {
	if [ "${success:-0}" -ne 1 ]; then
		rm -f "$output"
	fi
}
trap cleanup EXIT HUP INT TERM
if ! docker buildx build --network=none --output "type=oci,dest=$output" "$context"; then
	exit 1
fi
size=$(wc -c < "$output")
if [ "$size" -gt 1073741824 ]; then
	printf 'OCI archive exceeds 1 GiB\n' >&2
	exit 1
fi
success=1
