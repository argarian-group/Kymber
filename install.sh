#!/usr/bin/env bash
set -euo pipefail

repo="https://github.com/argarian-group/Kymber.git"
install_dir="${KYMBER_INSTALL_DIR:-/opt/kymber}"

if [[ "$EUID" -ne 0 ]]; then
	printf '%s\n' "Run this installer as root: curl -fsSL https://raw.githubusercontent.com/argarian-group/Kymber/main/install.sh | sudo -i bash" >&2
	exit 1
fi

if ! command -v systemctl >/dev/null 2>&1; then
	printf '%s\n' "systemd is required" >&2
	exit 1
fi

case "$(. /etc/os-release; printf '%s' "$ID")" in
	debian|ubuntu)
		. /etc/os-release
		apt-get update
		DEBIAN_FRONTEND=noninteractive apt-get install -y ca-certificates curl git jq openssh-server openssl sudo
		install -m 0755 -d /etc/apt/keyrings
		curl -fsSL "https://download.docker.com/linux/$ID/gpg" -o /etc/apt/keyrings/docker.asc
		chmod a+r /etc/apt/keyrings/docker.asc
		cat > /etc/apt/sources.list.d/docker.sources <<EOF
Types: deb
URIs: https://download.docker.com/linux/$ID
Suites: $VERSION_CODENAME
Components: stable
Architectures: $(dpkg --print-architecture)
Signed-By: /etc/apt/keyrings/docker.asc
EOF
		apt-get update
		DEBIAN_FRONTEND=noninteractive apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
		ssh_service=ssh
		;;
	arch)
		pacman -Syu --noconfirm
		pacman -S --needed --noconfirm ca-certificates curl docker docker-buildx docker-compose git jq openssh openssl sudo
		ssh_service=sshd
		;;
	fedora|rhel|centos|rocky|almalinux|ol)
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
			exit 1
		fi
		dnf install -y ca-certificates curl git jq openssh-server openssl sudo
		curl -fsSL "$docker_repo" -o /etc/yum.repos.d/docker-ce.repo
		dnf install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
		ssh_service=sshd
		;;
	*)
		printf '%s\n' "Supported systems: Debian, Ubuntu, Arch Linux, Fedora, RHEL, CentOS Stream, Rocky Linux, AlmaLinux, and Oracle Linux" >&2
		exit 1
		;;
esac

ssh-keygen -A
systemctl enable --now docker "$ssh_service"

if [[ -d "$install_dir/.git" ]]; then
	git -C "$install_dir" pull --ff-only
elif [[ -e "$install_dir" ]]; then
	printf 'Install directory already exists and is not a Git checkout: %s\n' "$install_dir" >&2
	exit 1
else
	git clone --depth 1 "$repo" "$install_dir"
fi

cd "$install_dir"
mkdir -p .local
chmod 700 .local
if [[ ! -e .env ]]; then
	cat > .env <<EOF
APP_PORT=3000
HTTP_PORT=80
HTTPS_PORT=443
SECURE_ONBOARD=false
TRAEFIK_CONFIG_TOKEN=$(openssl rand -hex 32)
TRAEFIK_ACME_EMAIL=
KYMBER_METRICS_TOKEN=$(openssl rand -hex 32)
KYMBER_GIT_BUILDS_ENABLED=false
KYMBER_BUILD_WORKER_TOKEN=
DB_NAME=kymber
DB_USER=kymber
DB_PASSWORD=$(openssl rand -hex 32)
DB_ROOT_PASSWORD=$(openssl rand -hex 32)
EOF
	chmod 600 .env
fi

env -u SUDO_UID -u SUDO_GID ./scripts/setup-local-ssh.sh
docker compose up -d --build

printf '\nKymber is starting. Open http://<server-ip>\n'
printf 'Install directory: %s\n' "$install_dir"
printf 'Git builds remain disabled by default.\n'
