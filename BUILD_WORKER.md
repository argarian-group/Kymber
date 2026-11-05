# Git build worker security boundary

Git builds stay disabled until the worker boundary passes the acceptance checks below. Rootless BuildKit alone is not the boundary: its OCI worker uses `network.host`, so build steps inherit the BuildKit daemon's network namespace. The daemon must therefore run in its own isolated network namespace, on a network with no route to Kymber, its database, server agents, or the host. BuildKit's rootless-mode guidance documents `rootlesskit --net=gvisor-tap-vsock --copy-up=/etc --disable-host-loopback buildkitd` as the preferred network isolation option: <https://github.com/moby/buildkit/blob/master/docs/rootless.md>.

## Worker contract

- The eventual job runner must start one ephemeral rootless BuildKit daemon per job; never mount the host Docker socket or join the API, data, or proxy networks.
- Give the daemon only a read-only source bundle, an empty output directory, and a bounded cache. Do not pass application secrets, server credentials, API tokens, or registry credentials into build steps.
- Keep build-step egress disabled. Do not add a general internet route to make dependency downloads work; first provide a separately controlled, allow-listed mirror/proxy and verify it cannot reach private, link-local, host, or control-plane addresses.
- Do not fetch Git URLs from the API process. A future source fetcher must reject local/private destinations, redirects into them, and oversized archives before it hands a source bundle to the worker.
- Export an OCI image archive to a per-job output volume. Enforce a 1 GiB archive limit while streaming, reject absolute paths, `..` traversal and links escaping the output root, verify OCI layout and content digests, then transfer it to the selected server over the existing authenticated SSH path. Do not push user-built images using Kymber registry credentials.
- Destroy the worker and job volume after completion or timeout. Keep only bounded logs and the resulting image digest in durable deployment history.

## Enablement checks

Before enabling a Git deployment endpoint, run an integration job proving that the build cannot connect to the API, MariaDB, Docker socket, host gateway, metadata/link-local addresses, or arbitrary internet destinations; that allowed mirror access is restricted to the documented hostnames; that cancellation kills the worker; and that output overflow, malformed OCI archives, digest mismatch, and worker restart all fail closed. Verify the final image digest on the target server before deployment.

Compose now has an opt-in `build-worker` profile with a pinned rootless BuildKit daemon using Docker `network_mode: none`. This gives its `network.host` build steps only the worker's isolated network namespace, with no API, database, host, or internet route, and mounts no Docker socket, secrets, or API volume. Build state is limited to a 2 GiB tmpfs; the container is capped at 4 GiB RAM, 2 CPUs and 512 processes. A local `FROM scratch` build completed and exported OCI, and the backend validator accepted its archive. The profile remains a preflight daemon: it is not per-job and is not connected to job submission, SSH delivery, or image import. The egress proxy, source handoff, artifact broker and host-side policy are still absent. Git builds must remain unavailable rather than weaken the network boundary. Rootless execution is not a VM boundary; the configured unconfined seccomp/AppArmor settings make kernel isolation a remaining risk.

The backend now has an unconnected OCI archive validator at `apps/backend/internal/artifacts/oci.go`. It spools at most 1 GiB, rejects unsafe paths and special files, verifies every blob's SHA-256 and checks index/manifest references. It has not yet been connected to an export endpoint or SSH transfer. The target currently uses Docker, so an OCI-to-Docker import step (or a separately validated registry transport) is still required before deployment.
