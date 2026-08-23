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

## DNS

`time.iocare.in` is assumed to already have an A record pointing at this VM's
public IP. No DNS resources are managed by this repo.
