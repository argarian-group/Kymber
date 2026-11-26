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
	base=$5
	dockerfile=$6
	expected=${7:-}
	case "$ref" in
		''|-*|*[!A-Za-z0-9._/-]*|*..*|*//*|*@\{*) exit 2 ;;
	esac
	for part in $(printf '%s' "$ref" | tr / ' '); do
		case "$part" in .*) exit 2 ;; *.|*.lock) exit 2 ;; esac
	done
	export GIT_TERMINAL_PROMPT=0 GIT_CONFIG_NOSYSTEM=1 GIT_CONFIG_GLOBAL=/dev/null GIT_LFS_SKIP_SMUDGE=1
	if [ -n "${KYMBER_GIT_AUTH_HEADER:-}" ]; then
		export GIT_CONFIG_COUNT=1 GIT_CONFIG_KEY_0=http.extraHeader GIT_CONFIG_VALUE_0="$KYMBER_GIT_AUTH_HEADER"
	fi
	unset KYMBER_GIT_AUTH_HEADER
	if [ -s "$dir/ssh_key" ]; then
		export GIT_SSH_COMMAND="ssh -i '$dir/ssh_key' -o IdentitiesOnly=yes -o UserKnownHostsFile='$dir/known_hosts' -o StrictHostKeyChecking=yes -o BatchMode=yes"
	fi
	git -c protocol.ext.allow=never -c protocol.file.allow=never clone --no-checkout --depth=1 "$url" "$dir/source" || exit 1
	git -C "$dir/source" -c protocol.ext.allow=never -c protocol.file.allow=never fetch --no-tags --depth=1 origin "$ref" || exit 1
	commit=$(git -C "$dir/source" rev-parse --verify 'FETCH_HEAD^{commit}') || exit 1
	if [ -n "$expected" ] && [ "$commit" != "$expected" ]; then
		git -C "$dir/source" -c protocol.ext.allow=never -c protocol.file.allow=never fetch --no-tags --depth=1 origin "$expected" || { printf 'queued commit %s is no longer reachable\n' "$expected" >&2; exit 1; }
		commit=$(git -C "$dir/source" rev-parse --verify 'FETCH_HEAD^{commit}') || exit 1
		[ "$commit" = "$expected" ] || exit 1
	fi
	unset GIT_CONFIG_COUNT GIT_CONFIG_KEY_0 GIT_CONFIG_VALUE_0 GIT_SSH_COMMAND
	rm -f "$dir/ssh_key"
	case "$commit" in *[!0-9a-f]*|'') exit 1 ;; esac
	[ "${#commit}" -eq 40 ] || exit 1
	git -C "$dir/source" -c core.hooksPath=/dev/null checkout --detach --force "$commit" || exit 1
	rm -rf "$dir/source/.git"
	printf '%s' "$commit" > "$dir/commit"
	root=$(readlink -f "$dir/source") || exit 1
	context=$(readlink -f "$root/$base") || { printf 'base directory not found\n' >&2; exit 1; }
	case "$context/" in "$root"/*) ;; *) printf 'base directory escapes the repository\n' >&2; exit 1 ;; esac
	pack=$(jq -r '.build_pack // "dockerfile"' "$dir/job.json")
	case "$pack" in
		nixpacks)
			command -v nixpacks >/dev/null 2>&1 || { printf 'nixpacks is not installed on the build host\n' >&2; exit 1; }
			set --
			for option in install build start; do
				value=$(jq -r --arg option "$option" '.commands[$option] // ""' "$dir/job.json")
				[ -n "$value" ] && set -- "$@" "--$option-cmd" "$value"
			done
			while IFS= read -r arg; do
				set -- "$@" --env "$arg"
			done < "$dir/build-args"
			rm -rf "$context/.nixpacks"
			nixpacks build "$context" --out "$context" "$@" || exit 1
			file="$context/.nixpacks/Dockerfile"
			;;
		static)
			publish=$(jq -r '.publish_directory // ""' "$dir/job.json")
			case "/$publish/" in */../*|*/./*) printf 'invalid publish directory\n' >&2; exit 1 ;; esac
			site=$(readlink -f "$context/$publish") || { printf 'publish directory not found\n' >&2; exit 1; }
			case "$site/" in "$context"/*) ;; *) printf 'publish directory escapes the base directory\n' >&2; exit 1 ;; esac
			[ -d "$site" ] || { printf 'publish directory not found: %s\n' "$publish" >&2; exit 1; }
			relative=.${site#"$context"}
			case "$relative" in *[!A-Za-z0-9._/@+-]*) printf 'unsupported characters in publish directory\n' >&2; exit 1 ;; esac
			file="$dir/Dockerfile.static"
			printf 'FROM nginx:stable-alpine\nCOPY ["%s/", "/usr/share/nginx/html/"]\n' "$relative" > "$file"
			;;
		*)
			file=$(readlink -f "$context/$dockerfile") || { printf 'Dockerfile not found\n' >&2; exit 1; }
			case "$file" in "$root"/*) ;; *) printf 'Dockerfile escapes the repository\n' >&2; exit 1 ;; esac
			[ -f "$file" ] || { printf 'Dockerfile not found: %s\n' "$dockerfile" >&2; exit 1; }
			if [ -s "$dir/build-args" ]; then
				# Like Coolify: declare every build argument in each stage, so Dockerfiles without ARG lines still receive them.
				names=$(cut -d= -f1 "$dir/build-args" | tr '\n' ' ')
				awk -v names="$names" '
					BEGIN { count = split(names, arg, " ") }
					{ print }
					toupper($1) == "FROM" { pending = 1 }
					pending && !/\\[ \t]*$/ { for (i = 1; i <= count; i++) print "ARG " arg[i]; pending = 0 }
				' "$file" > "$dir/Dockerfile.args" || exit 1
				[ -f "$file.dockerignore" ] && cp "$file.dockerignore" "$dir/Dockerfile.args.dockerignore"
				file="$dir/Dockerfile.args"
			fi
			;;
	esac
	exec "$KYMBER_BUILD_RUNNER" "$context" "$dir/artifact.oci.tar" "$server" "$file" "$dir/build-args"
fi

case "$KYMBER_API_URL" in
	https://*|http://localhost|http://localhost:*|http://127.0.0.1|http://127.0.0.1:*) ;;
	*) printf 'KYMBER_API_URL must use HTTPS or loopback HTTP\n' >&2; exit 2 ;;
esac

mkdir -p "$KYMBER_BUILD_WORK_DIR"
api=${KYMBER_API_URL%/}
headers="Authorization: Bearer $KYMBER_BUILD_WORKER_TOKEN"
if [ -z "${KYMBER_BUILD_SERVER_ID:-}" ]; then
	if ! KYMBER_BUILD_SERVER_ID=$(curl --connect-timeout 10 --max-time 30 --silent "$api/internal/build-worker/server" -H "$headers" | jq -er '.server_id // error(.error // "API unreachable")' 2>/dev/null); then
		printf 'Waiting for the API and an online local server (register it under Servers)\n' >&2
		exit 1
	fi
fi

report_failure() {
	job=$1
	attempt=$2
	message=$3
	curl --connect-timeout 10 --max-time 30 --fail --silent --show-error -X POST "$api/internal/build-worker/jobs/$job/fail" \
		-H "$headers" -H "X-Build-Attempt: $attempt" -H 'Content-Type: application/json' \
		--data "$(jq -cn --arg message "$message" '{message:$message}')" >/dev/null || true
}

send_log() {
	tail -c 262144 "$dir/build.log" 2>/dev/null | curl --connect-timeout 10 --max-time 30 --silent -o /dev/null -X PUT "$api/internal/build-worker/jobs/$job/log" \
		-H "$headers" -H "X-Build-Attempt: $attempt" -H 'Content-Type: text/plain; charset=utf-8' --data-binary @- || true
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
	base=$(jq -r '.base_directory // ""' "$response")
	dockerfile=$(jq -r '.dockerfile_path // "Dockerfile"' "$response")
	expected=$(jq -r '.commit // "" | select(test("^([0-9a-f]{40})?$"))' "$response")
	auth_header=$(jq -r '.auth.header // ""' "$response")
	jq -r '.auth.ssh_key // ""' "$response" > "$dir/ssh_key"
	jq -r '.auth.known_hosts // ""' "$response" > "$dir/known_hosts"
	[ -s "$dir/known_hosts" ] && [ "$(wc -c < "$dir/ssh_key")" -gt 1 ] || rm -f "$dir/ssh_key"
	jq 'del(.auth)' "$response" > "$dir/job.json.tmp" && mv "$dir/job.json.tmp" "$response"
	jq -r '.build_args // [] | .[] | select(type == "string" and test("^[A-Za-z_][A-Za-z0-9_]*=[^\\n\\r]*$"))' "$response" > "$dir/build-args"
	if [ "$server" != "$KYMBER_BUILD_SERVER_ID" ]; then
		rm -rf "$dir"
		sleep 5
		continue
	fi
	if ! printf '%s' "$url" | grep -Eq '^(https://|ssh://[A-Za-z0-9._-]{1,64}@)[A-Za-z0-9.-]{1,253}(:[0-9]{1,5})?(/[A-Za-z0-9_][A-Za-z0-9._-]{0,99}){2,20}(\.git)?$'; then
		report_failure "$job" "$attempt" 'Controller rejected repository URL.'
		rm -rf "$dir"
		continue
	fi
	paths_ok=1
	case "$base" in /*) paths_ok=0 ;; esac
	case "$dockerfile" in /*|'') paths_ok=0 ;; esac
	case "/$base/$dockerfile/" in */../*|*/./*) paths_ok=0 ;; esac
	if [ "$paths_ok" -ne 1 ]; then
		report_failure "$job" "$attempt" 'Controller rejected build paths.'
		rm -rf "$dir"
		continue
	fi
	KYMBER_GIT_AUTH_HEADER=$auth_header setsid timeout --signal=TERM --kill-after=10s 2h "$0" --run-job "$url" "$ref" "$dir" "$server" "$base" "$dockerfile" "$expected" >"$dir/build.log" 2>&1 &
	build_pid=$!
	active_pid=$build_pid
	active_dir=$dir
	failed=0
	heartbeat_elapsed=0
	while kill -0 "$build_pid" 2>/dev/null; do
		sleep 3
		send_log
		heartbeat_elapsed=$((heartbeat_elapsed + 3))
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
	send_log
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
