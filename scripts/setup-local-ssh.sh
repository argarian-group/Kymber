#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
private_key="$root/.local/kymber-local-ssh"
public_key="$private_key.pub"
installed_marker="$root/.local/kymber-local-ssh-installed"
host_key="/etc/ssh/ssh_host_ed25519_key.pub"
owner_uid="${SUDO_UID:-$(id -u)}"
owner_gid="${SUDO_GID:-$(id -g)}"

mkdir -p "$root/.local"
chmod 700 "$root/.local"
if [[ ! -e "$private_key" && ! -e "$public_key" ]]; then
  umask 077
  ssh-keygen -q -t ed25519 -N "" -C kymber-local -f "$private_key"
elif [[ ! -s "$private_key" || ! -s "$public_key" ]]; then
  printf '%s\n' "Local SSH keypair is incomplete: $private_key" >&2
  exit 1
fi
if [[ "$(stat -c %u "$private_key")" != "$owner_uid" ]]; then
	if [[ "$EUID" -eq 0 ]]; then
		chown "$owner_uid:$owner_gid" "$private_key" "$public_key"
	else
		sudo chown "$owner_uid:$owner_gid" "$private_key" "$public_key"
	fi
fi
chmod 600 "$private_key"
chmod 644 "$public_key"
if [[ ! -s "$host_key" ]]; then
  printf '%s\n' "OpenSSH Ed25519 host key is missing: $host_key" >&2
  exit 1
fi

expected="$(ssh-keygen -y -f "$private_key" | awk '{print $1 " " $2}')"
actual="$(awk '{print $1 " " $2}' "$public_key")"
if [[ "$expected" != "$actual" ]]; then
  printf '%s\n' "Local SSH keypair does not match" >&2
  exit 1
fi

public_line="$(cat "$public_key")"
if [[ -s "$installed_marker" ]] && cmp -s "$installed_marker" <(printf '%s\n' "$public_line"); then
  exit 0
fi
sudo install -d -o root -g root -m 0700 /root/.ssh
sudo touch /root/.ssh/authorized_keys
if ! sudo grep -Fqx -- "$public_line" /root/.ssh/authorized_keys; then
  printf '%s\n' "$public_line" | sudo tee -a /root/.ssh/authorized_keys >/dev/null
fi
sudo chown root:root /root/.ssh/authorized_keys
sudo chmod 0600 /root/.ssh/authorized_keys
printf '%s\n' "$public_line" > "$installed_marker"
chmod 600 "$installed_marker"
