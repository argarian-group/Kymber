#!/bin/sh
set -eu
umask 077
set -C

if [ "$#" -ne 2 ] || [ ! -d "$1" ]; then
	printf 'usage: %s BUILD_CONTEXT OUTPUT_OCI_TAR\n' "$0" >&2
	exit 2
fi

context=$(cd "$1" && pwd -P)
case "$context" in
	*,*|*:*)
		printf 'build context path contains unsupported characters\n' >&2
		exit 2
		;;
esac
output=$2
case "$output" in
	/*) ;;
	*) output=$(pwd -P)/$output ;;
esac
if [ -e "$output" ]; then
	printf 'output path already exists\n' >&2
	exit 2
fi

container=$(docker create --network none --memory 4g --cpus 2 --pids-limit 512 \
	--security-opt seccomp=unconfined --security-opt apparmor=unconfined --security-opt systempaths=unconfined \
	--mount "type=bind,src=$context,dst=/src,readonly" \
	--tmpfs /home/user/.local/share/buildkit:rw,exec,nodev,nosuid,size=2g,uid=1000,gid=1000,mode=0700 \
	--tmpfs /out:rw,nodev,nosuid,size=1100m,uid=1000,gid=1000,mode=0700 \
	--entrypoint sh moby/buildkit:v0.27.1-rootless -ec '
	rootlesskit buildkitd &
	i=0
	until buildctl debug workers >/dev/null 2>&1; do
		i=$((i + 1))
		[ "$i" -lt 30 ] || exit 1
		sleep 1
	done
	exec sleep infinity
')
cleanup() {
	docker rm -f "$container" >/dev/null 2>&1 || true
	if [ "${success:-0}" -ne 1 ]; then
		rm -f "$output"
	fi
}
trap cleanup EXIT HUP INT TERM
docker start "$container" >/dev/null
if ! docker exec "$container" buildctl build --frontend dockerfile.v0 --local context=/src --local dockerfile=/src --output type=oci,dest=/out/image.tar; then
	docker logs "$container" >&2
	exit 1
fi
docker exec "$container" cat /out/image.tar > "$output"
size=$(wc -c < "$output")
if [ "$size" -gt 1073741824 ]; then
	printf 'OCI archive exceeds 1 GiB\n' >&2
	exit 1
fi
success=1
