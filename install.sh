#!/usr/bin/env bash
set -euo pipefail

repo="https://github.com/argarian-group/Kymber.git"
install_dir="${KYMBER_INSTALL_DIR:-/opt/kymber}"
logfile=$(mktemp /tmp/kymber-install.XXXXXX)
step_number=0
total_steps=7
spinner=('⠋' '⠙' '⠹' '⠸' '⠼' '⠴' '⠦' '⠧' '⠇' '⠏')
green=''
red=''
reset=''
if [[ -t 1 && -z "${NO_COLOR:-}" ]]; then
	green=$'\033[32m'
	red=$'\033[31m'
	reset=$'\033[0m'
fi

run_step() {
	local label=$1
	local pid frame=0
	shift
	step_number=$((step_number + 1))
	printf '  [%d/%d] %s ' "$step_number" "$total_steps" "$label"
	"$@" >>"$logfile" 2>&1 &
	pid=$!
	if [[ -t 1 ]]; then
		while kill -0 "$pid" 2>/dev/null; do
			printf '\r  [%d/%d] %s %s' "$step_number" "$total_steps" "$label" "${spinner[frame]}"
			frame=$(((frame + 1) % ${#spinner[@]}))
			sleep 0.12
		done
	fi
	if wait "$pid"; then
		printf '\r'
		printf ' %sconcluído%s\n' "$green" "$reset"
	else
		local result=$?
		printf '\r'
		printf ' %serro%s\n\n' "$red" "$reset"
		printf '\nLog completo: %s\n' "$logfile" >&2
		exit "$result"
	fi
}

if [[ "$EUID" -ne 0 ]]; then
	printf '%s\n' "Run this installer as root: curl -fsSL https://raw.githubusercontent.com/argarian-group/Kymber/main/install.sh | sudo -i bash" >&2
	exit 1
fi

if ! command -v systemctl >/dev/null 2>&1; then
	printf '%s\n' "systemd is required" >&2
	exit 1
fi

printf '\nKymber — instalação\nDiretório: %s\n\n' "$install_dir"
if [[ -t 0 ]]; then
	read -r -p 'Deseja continuar com a instalação/atualização? [s/N] ' answer
else
	answer=${KYMBER_INSTALL_CONFIRM:-}
fi
if [[ ! "$answer" =~ ^[sS]([iI][mM])?$ ]]; then
	printf 'Instalação cancelada.\n'
	exit 0
fi

install_debian() {
	. /etc/os-release
	apt-get update || return
	DEBIAN_FRONTEND=noninteractive apt-get install -y ca-certificates curl git iproute2 jq openssh-server openssl sudo || return
	install -m 0755 -d /etc/apt/keyrings || return
	curl -fsSL "https://download.docker.com/linux/$ID/gpg" -o /etc/apt/keyrings/docker.asc || return
	chmod a+r /etc/apt/keyrings/docker.asc || return
	cat > /etc/apt/sources.list.d/docker.sources <<EOF
Types: deb
URIs: https://download.docker.com/linux/$ID
Suites: $VERSION_CODENAME
Components: stable
Architectures: $(dpkg --print-architecture)
Signed-By: /etc/apt/keyrings/docker.asc
EOF
	apt-get update || return
	DEBIAN_FRONTEND=noninteractive apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin || return
}

install_arch() {
	pacman -Syu --noconfirm || return
	pacman -S --needed --noconfirm ca-certificates curl docker docker-buildx docker-compose git iproute2 jq openssh openssl sudo || return
}

install_redhat() {
	. /etc/os-release
	case "$ID" in
		fedora) docker_repo="https://download.docker.com/linux/fedora/docker-ce.repo" ;;
		rhel|ol) docker_repo="https://download.docker.com/linux/rhel/docker-ce.repo" ;;
		*) docker_repo="https://download.docker.com/linux/centos/docker-ce.repo" ;;
	esac
	min_version=8
	if [[ "$ID" == centos ]]; then
		min_version=9
	fi
	if [[ "$ID" != fedora ]] && { [[ "${VERSION_ID%%.*}" -lt "$min_version" ]] || [[ "${VERSION_ID%%.*}" -gt 10 ]]; }; then
		printf '%s\n' "Supported Red Hat-compatible versions: 8, 9, and 10 (CentOS Stream 9 and 10)" >&2
		return 1
	fi
	dnf install -y ca-certificates curl git iproute jq openssh-server openssl sudo || return
	curl -fsSL "$docker_repo" -o /etc/yum.repos.d/docker-ce.repo || return
	dnf install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin || return
}

start_services() {
	ssh-keygen -A && systemctl enable --now docker "$ssh_service"
}

prepare_checkout() {
	if [[ -d "$install_dir/.git" ]]; then
		git -C "$install_dir" pull --ff-only
	elif [[ -e "$install_dir" ]]; then
		printf 'Install directory already exists and is not a Git checkout: %s\n' "$install_dir" >&2
		return 1
	else
		git clone --depth 1 "$repo" "$install_dir"
	fi
}

prepare_configuration() {
	mkdir -p .local && chmod 700 .local || return
	if [[ ! -e .env ]]; then
		cat > .env <<EOF
APP_PORT=3000
HTTP_PORT=80
HTTPS_PORT=443
SECURE_ONBOARD=false
TRAEFIK_CONFIG_TOKEN=$(openssl rand -hex 32)
TRAEFIK_ACME_EMAIL=
KYMBER_METRICS_TOKEN=$(openssl rand -hex 32)
KYMBER_GIT_BUILDS_ENABLED=true
KYMBER_BUILD_WORKER_TOKEN=
DB_NAME=kymber
DB_USER=kymber
DB_PASSWORD=$(openssl rand -hex 32)
DB_ROOT_PASSWORD=$(openssl rand -hex 32)
EOF
		[[ -s .env ]] || return 1
		chmod 600 .env || return
	fi
}

compose_port() {
	local name=$1
	local default=$2
	local value
	value=$(awk -F= -v name="$name" '$1 == name {value=$2} END {gsub(/["[:space:]]/, "", value); print value}' .env)
	printf '%s' "${value:-$default}"
}

print_addresses() {
	local addresses
	local http_port app_port
	http_port=$(compose_port HTTP_PORT 80)
	app_port=$(compose_port APP_PORT 3000)
	mapfile -t addresses < <(ip -o -4 addr show up scope global | awk '$2 !~ /^(docker|br-|veth|cni|podman|virbr|vboxnet|vmnet)/ {split($4, a, "/"); print a[1]}' | sort -Vu)
	if ((${#addresses[@]} == 0)); then
		printf '  Nenhum IPv4 global detectado.\n'
		return
	fi
	for address in "${addresses[@]}"; do
		if [[ "$http_port" == 80 ]]; then
			printf '  http://%s\n' "$address"
		else
			printf '  http://%s:%s\n' "$address" "$http_port"
		fi
		if [[ "$app_port" != "$http_port" ]]; then
			printf '  http://%s:%s\n' "$address" "$app_port"
		fi
	done
}

check_ports() {
	local port id container_name listening compose_ids
	local ports=($(compose_port HTTP_PORT 80) $(compose_port APP_PORT 3000) $(compose_port HTTPS_PORT 443))
	compose_ids=$(docker compose ps -q)
	for port in "${ports[@]}"; do
		if [[ ! "$port" =~ ^[0-9]{1,5}$ ]] || ((port < 1 || port > 65535)); then
			printf 'Invalid host port in .env: %s\n' "$port"
			return 1
		fi
		for id in $(docker ps -q); do
			if docker inspect --format '{{json .NetworkSettings.Ports}}' "$id" | jq -e --arg port "$port" '[.[]? // [] | .[]? | select(.HostPort == $port)] | length > 0' >/dev/null; then
				container_name=$(docker inspect --format '{{.Name}}' "$id")
				if grep -Fxq "$id" <<<"$compose_ids"; then
					continue
				fi
				printf 'Port %s is already published by container %s\n' "$port" "$container_name"
				return 1
			fi
		done
		listening=$(ss -H -ltn "sport = :$port")
		if [[ -n "$listening" ]]; then
			printf 'Port %s is already listening on the host:\n%s\n' "$port" "$listening"
			return 1
		fi
	done
}

os_id=$( . /etc/os-release; printf '%s' "$ID")
case "$os_id" in
	debian|ubuntu)
		ssh_service=ssh
		install_function=install_debian
		;;
	arch)
		ssh_service=sshd
		install_function=install_arch
		;;
	fedora|rhel|centos|rocky|almalinux|ol)
		ssh_service=sshd
		install_function=install_redhat
		;;
	*)
		printf '%s\n' "Supported systems: Debian, Ubuntu, Arch Linux, Fedora, RHEL, CentOS Stream, Rocky Linux, AlmaLinux, and Oracle Linux" >&2
		exit 1
		;;
esac

run_step "Ativando Docker e SSH" start_services
run_step "Baixando o Kymber" prepare_checkout
cd "$install_dir"
run_step "Gerando configuração segura" prepare_configuration
run_step "Verificando portas HTTP/HTTPS" check_ports
run_step "Configurando SSH local" env -u SUDO_UID -u SUDO_GID ./scripts/setup-local-ssh.sh
run_step "Construindo e iniciando os containers" docker compose up -d --build --wait --wait-timeout 180
printf '\nContainers:\n'
docker compose ps --format 'table {{.Name}}\t{{.State}}\t{{.Status}}'
printf '\nAcesse o Kymber pelos endereços detectados:\n'
print_addresses
printf '\nInstalação: %s\nBuilds Git: ativados\n' "$install_dir"
printf 'Atenção: HTTP não criptografa credenciais nem sessões; configure HTTPS antes de usar em produção.\n'
rm -f "$logfile"
