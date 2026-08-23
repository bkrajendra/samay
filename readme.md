# Samay - chronyd timeserver

A lightweight NTP time server (`chronyd`) for keeping IoT devices synced, running
as a `hostNetwork` pod on a single-node Kubernetes cluster and exposed on
UDP 123 as `time.iocare.in`.

## How it works

- `Dockerfile` builds an `alpine`-based image with `chrony` installed. It's
  intentionally minimal so other services (e.g. a future web console) can be
  added to it later.
- `config/chrony.conf` is the source of truth for chrony's configuration. It's
  baked into `k8s/configmap.yaml` (see regeneration command there) and mounted
  into the container at runtime — so config changes don't require a rebuild.
- The pod runs with `hostNetwork: true`, so `chronyd` binds directly to the
  node's UDP 123 — no `Service` is needed. `time.iocare.in` is assumed to
  already point at this VM's IP.

## Build (CI)

Pushing to `main` triggers `.github/workflows/build.yml`, which builds
`Dockerfile` and pushes to `ghcr.io/bkrajendra/samay:latest` (and
`:sha-<short>`).

**One-time step:** after the first successful workflow run, open the package
settings for `samay` under your GitHub packages
(https://github.com/bkrajendra?tab=packages) and set visibility to **Public**,
so the cluster can pull the image without credentials.

If you'd rather keep the image private, create an image pull secret instead
and reference it from the deployment:

```
kubectl create secret docker-registry ghcr-pull \
  -n timeserver \
  --docker-server=ghcr.io \
  --docker-username=bkrajendra \
  --docker-password=github_pat_xxx \
  --docker-email=bkrajendra@gmail.com
```

then add to `k8s/deployment.yaml`'s pod spec:

```yaml
      imagePullSecrets:
        - name: ghcr-pull
```

## Local build/test (optional)

```
docker build -t samay:local .
docker run --rm -it \
  -v "$(pwd)/config/chrony.conf:/etc/chrony/chrony.conf:ro" \
  -p 123:123/udp \
  --cap-add SYS_TIME --cap-add NET_BIND_SERVICE \
  samay:local
```

## Deploy

```
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/deployment.yaml
```

If you edit `config/chrony.conf`, regenerate `k8s/configmap.yaml` before
re-applying:

```
kubectl create configmap timeserver-config -n timeserver \
  --from-file=chrony.conf=config/chrony.conf \
  --dry-run=client -o yaml > k8s/configmap.yaml
kubectl apply -f k8s/configmap.yaml
kubectl rollout restart deployment/timeserver -n timeserver
```

## Validate

Check the pod is running and see its startup logs:

```
kubectl -n timeserver get pods -o wide
kubectl -n timeserver logs deploy/timeserver
```

Check chrony's own sync/tracking state from inside the pod:

```
kubectl -n timeserver exec deploy/timeserver -- chronyc tracking
kubectl -n timeserver exec deploy/timeserver -- chronyc sources -v
```

Confirm UDP 123 is actually listening on the node (run on the VM):

```
ss -lun | grep :123
```

Query it as an external client would:

```
# Linux (chrony installed)
chronyd -Q 'server time.iocare.in iburst' -d

# Linux (ntpdate installed)
ntpdate -q time.iocare.in

# Windows
w32tm /stripchart /computer:time.iocare.in /samples:3 /dataonly
```

A successful query returns a valid offset/round-trip time from
`time.iocare.in` — that confirms the server is reachable and serving time
correctly.

## Web console

`console/` is a small web UI + API for monitoring and lightly operating the
timeserver (dashboard, NTP sources, clients, diagnostics, force sync/step
clock/restart). It runs as a second container (`console`) in the same
`timeserver` pod — see
`docs/superpowers/specs/2026-08-23-web-console-design.md` for the full
design, including the security constraint that the UI only ever calls a
fixed set of whitelisted `chronyc` operations, never raw shell commands.

### Required Secret

The console needs `CONSOLE_USERNAME`/`CONSOLE_PASSWORD` for its login page,
provided via a `console-env` Secret (not committed to git):

```
kubectl create secret generic console-env \
  -n timeserver \
  --from-literal=CONSOLE_USERNAME=admin \
  --from-literal=CONSOLE_PASSWORD='choose-a-real-password'
```

Optional overrides (same Secret, or omit for defaults): `LISTEN_ADDR`
(default `:8080`), `CHRONY_SOCKET` (default `/run/chrony/chronyd.sock`),
`COOKIE_SECURE` (default `false` — set `true` once this is behind TLS).

### Build (CI)

Pushing changes under `console/` to `main` triggers
`.github/workflows/build-console.yml`, which builds `console/Dockerfile`
and pushes to `ghcr.io/bkrajendra/samay-console:latest` (and
`:sha-<short>`). Same one-time step as the timeserver image: set the
`samay-console` package to **Public** in your GitHub packages settings
after the first successful run (or use an `imagePullSecret`, same as
described above for the timeserver image).

### Local development (optional)

Backend (needs a real or mock `chronyc`/socket to be useful; it will start
without one but API calls will report chronyd as unreachable):

```
cd console/backend
printf 'CONSOLE_USERNAME=admin\nCONSOLE_PASSWORD=devpassword\n' > .env
go run ./cmd/server
```

Frontend (dev server proxies `/api` to `localhost:8080`, see
`vite.config.ts`):

```
cd console/frontend
npm install
npm run dev
```

### Deploy

Already part of `k8s/deployment.yaml` — applying it (see Deploy above)
deploys both the `chronyd` and `console` containers together. Make sure the
`console-env` Secret exists first.

### Validate

```
kubectl -n timeserver get pods -o wide
kubectl -n timeserver logs deploy/timeserver -c console
```

Then browse to `http://<vm-ip>:8080` (or `http://time.iocare.in:8080`),
log in with the credentials from the `console-env` Secret, and check the
Dashboard, Sources, Clients, and Diagnostics pages load live data.

## DNS

`time.iocare.in` is assumed to already have an A record pointing at this VM's
public IP. No DNS resources are managed by this repo.
