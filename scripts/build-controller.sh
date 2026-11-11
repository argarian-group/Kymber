#!/bin/sh
set -eu
umask 077

: "${KYMBER_API_URL:?Set KYMBER_API_URL}"
: "${KYMBER_BUILD_WORKER_TOKEN:?Set KYMBER_BUILD_WORKER_TOKEN}"
: "${KYMBER_BUILD_RUNNER:=scripts/build-oci-isolated.sh}"
: "${KYMBER_BUILD_WORK_DIR:=/tmp/kymber-build-controller}"
if [ "${1:-}" = --run-job ]; then
	shift
	url=$1
	ref=$2
	dir=$3
	server=$4
	case "$ref" in
		''|-*|*[!A-Za-z0-9._/-]*|*..*|*//*|*@\{*) exit 2 ;;
	esac
	for part in $(printf '%s' "$ref" | tr / ' '); do
		case "$part" in .*) exit 2 ;; *.|*.lock) exit 2 ;; esac
	done
	export GIT_TERMINAL_PROMPT=0 GIT_CONFIG_NOSYSTEM=1 GIT_CONFIG_GLOBAL=/dev/null GIT_LFS_SKIP_SMUDGE=1
	git -c protocol.ext.allow=never -c protocol.file.allow=never clone --no-checkout --depth=1 "$url" "$dir/source" || exit 1
	git -C "$dir/source" -c protocol.ext.allow=never -c protocol.file.allow=never fetch --no-tags --depth=1 origin "$ref" || exit 1
	commit=$(git -C "$dir/source" rev-parse --verify 'FETCH_HEAD^{commit}') || exit 1
	case "$commit" in *[!0-9a-f]*|'') exit 1 ;; esac
	[ "${#commit}" -eq 40 ] || exit 1
	git -C "$dir/source" -c core.hooksPath=/dev/null checkout --detach --force "$commit" || exit 1
	rm -rf "$dir/source/.git"
	printf '%s' "$commit" > "$dir/commit"
	exec "$KYMBER_BUILD_RUNNER" "$dir/source" "$dir/artifact.oci.tar" "$server"
fi

case "$KYMBER_API_URL" in
	https://*|http://localhost|http://localhost:*|http://127.0.0.1|http://127.0.0.1:*) ;;
	*) printf 'KYMBER_API_URL must use HTTPS or loopback HTTP\n' >&2; exit 2 ;;
esac

mkdir -p "$KYMBER_BUILD_WORK_DIR"
api=${KYMBER_API_URL%/}
headers="Authorization: Bearer $KYMBER_BUILD_WORKER_TOKEN"
if [ -z "${KYMBER_BUILD_SERVER_ID:-}" ]; then
	KYMBER_BUILD_SERVER_ID=$(curl --connect-timeout 10 --max-time 30 --fail --silent --show-error "$api/internal/build-worker/server" -H "$headers" | jq -er '.server_id')
fi

report_failure() {
	job=$1
	attempt=$2
	message=$3
	curl --connect-timeout 10 --max-time 30 --fail --silent --show-error -X POST "$api/internal/build-worker/jobs/$job/fail" \
		-H "$headers" -H "X-Build-Attempt: $attempt" -H 'Content-Type: application/json' \
		--data "$(jq -cn --arg message "$message" '{message:$message}')" >/dev/null || true
}

active_dir=
active_pid=
cleanup() {
	if [ -n "$active_pid" ]; then
		kill -TERM -- "-$active_pid" 2>/dev/null || true
		wait "$active_pid" 2>/dev/null || true
	fi
	if [ -n "$active_dir" ]; then
		rm -rf "$active_dir"
	fi
}
trap cleanup EXIT
trap 'exit 129' HUP
trap 'exit 130' INT
trap 'exit 143' TERM

while :; do
	dir=$(mktemp -d "$KYMBER_BUILD_WORK_DIR/job-XXXXXX")
	response=$dir/job.json
	status=$(curl --connect-timeout 10 --max-time 30 --silent --show-error -o "$response" -w '%{http_code}' -X POST "$api/internal/build-worker/claim" -H "$headers" -H "X-Build-Server-ID: $KYMBER_BUILD_SERVER_ID") || { rm -rf "$dir"; sleep 5; continue; }
	if [ "$status" = 204 ]; then
		rm -rf "$dir"
		sleep 5
		continue
	fi
	if [ "$status" != 200 ]; then
		cat "$response" >&2
		rm -rf "$dir"
		case "$status" in 401|403|404) exit 1 ;; esac
		sleep 5
		continue
	fi
	job=$(jq -er '.id | select(test("^[0-9a-f]{6}$"))' "$response")
	attempt=$(jq -er '.attempt | select(type == "number" and . >= 1 and . <= 5)' "$response")
	url=$(jq -er '.repository_url | select(type == "string")' "$response")
	ref=$(jq -er '.ref | select(type == "string")' "$response")
	server=$(jq -er '.server_id | select(type == "string" and test("^[0-9a-f]{6}$"))' "$response")
	if [ "$server" != "$KYMBER_BUILD_SERVER_ID" ]; then
		rm -rf "$dir"
		sleep 5
		continue
	fi
	if ! printf '%s' "$url" | grep -Eq '^https://github\.com/[A-Za-z0-9-]{1,39}/[A-Za-z0-9._-]{1,100}(\.git)?$'; then
		report_failure "$job" "$attempt" 'Controller rejected repository URL.'
		rm -rf "$dir"
		continue
	fi
	setsid timeout --signal=TERM --kill-after=10s 2h "$0" --run-job "$url" "$ref" "$dir" "$server" >"$dir/build.log" 2>&1 &
	build_pid=$!
	active_pid=$build_pid
	active_dir=$dir
	failed=0
	heartbeat_elapsed=0
	while kill -0 "$build_pid" 2>/dev/null; do
		sleep 15
		heartbeat_elapsed=$((heartbeat_elapsed + 15))
		kill -0 "$build_pid" 2>/dev/null || break
		[ "$heartbeat_elapsed" -lt 240 ] && continue
		heartbeat_elapsed=0
		if ! curl --connect-timeout 10 --max-time 30 --fail --silent --show-error -X POST "$api/internal/build-worker/jobs/$job/heartbeat" -H "$headers" -H "X-Build-Attempt: $attempt" >/dev/null; then
			kill -TERM -- "-$build_pid" 2>/dev/null || true
			failed=1
			break
		fi
	done
	if ! wait "$build_pid"; then
		failed=1
	fi
	if [ "$failed" -ne 0 ]; then
		tail -c 2048 "$dir/build.log" >&2 || true
		report_failure "$job" "$attempt" 'Git checkout or host build failed.'
	else
		setsid curl --connect-timeout 10 --max-time 1800 --fail --silent --show-error -X POST "$api/internal/build-worker/jobs/$job/artifact" \
			-H "$headers" -H "X-Build-Attempt: $attempt" -H "X-Git-Commit: $(cat "$dir/commit")" \
			-H 'Content-Type: application/vnd.oci.image.layout.v1.tar' --data-binary "@$dir/artifact.oci.tar" &
		upload_pid=$!
		active_pid=$upload_pid
		upload_failed=0
		heartbeat_elapsed=0
		while kill -0 "$upload_pid" 2>/dev/null; do
			sleep 15
			heartbeat_elapsed=$((heartbeat_elapsed + 15))
			kill -0 "$upload_pid" 2>/dev/null || break
			[ "$heartbeat_elapsed" -lt 240 ] && continue
			heartbeat_elapsed=0
			if ! curl --connect-timeout 10 --max-time 30 --fail --silent --show-error -X POST "$api/internal/build-worker/jobs/$job/heartbeat" -H "$headers" -H "X-Build-Attempt: $attempt" >/dev/null; then
				kill -TERM -- "-$upload_pid" 2>/dev/null || true
				upload_failed=1
				break
			fi
		done
		if ! wait "$upload_pid"; then
			upload_failed=1
		fi
		active_pid=
		if [ "$upload_failed" -ne 0 ]; then
			report_failure "$job" "$attempt" 'Artifact upload failed.'
		fi
	fi
	rm -rf "$dir"
	active_pid=
	active_dir=
done
