# Web console for the samay timeserver — v1 design

Status: approved for implementation
Date: 2026-08-23

## Goal

Add a minimal web console to `samay` so the timeserver can be monitored and
lightly operated without SSHing in and running `chronyc` by hand. This spec
covers v1 (MVP) scope only.

## v1 scope (prioritized)

1. **Auth** — login page; credentials are a hardcoded username/password read
   from `.env` on the server; successful login creates a server-side session,
   returned to the browser as an `HttpOnly` cookie. All API routes below
   (except login) require a valid session.
2. **Dashboard** — current server time, UTC time, system timezone, chronyd
   running/stopped, synchronized yes/no, stratum, reference source, offset,
   frequency correction, root delay, root dispersion, last sync time / time
   since last sync, configured source count, reachable source count, client
   count.
3. **NTP Sources** — read-only table merging `chronyc sources -v` and
   `chronyc sourcestats -v`: address, stratum, reach, last-rx, offset,
   jitter, selected/candidate/other status.
4. **Client Monitoring** — read-only table from `chronyc clients`: client
   address, last request, request count, active/stale/offline (derived from
   how recent the last request was).
5. **Service Operations** — Force Sync, Step Clock, Restart chronyd. Each is
   a POST action; the frontend requires an explicit confirm dialog before
   calling restart or step (both can cause a visible clock jump / brief
   downtime).
6. **Diagnostics** — a handful of pass/warn/fail checks computed from data
   already fetched for the dashboard (chronyd running, synchronized, root
   dispersion below threshold, at least one reachable source, etc). No new
   data source — pure derivation.

## Explicitly deferred (not built in v1)

- **Config editing** (add/remove NTP servers, edit ACL/allowed networks) —
  `chrony.conf` is currently a read-only ConfigMap mount; making it
  writable, persisted, and safely reloadable is real new architecture, so
  `PUT /sources` and `PUT /access-rules` are v2. Note `GET /api/sources`
  (v1, see below) already shows every configured source — reachable or
  not — since `chronyc sources` reports on configured servers regardless
  of current reachability, so read visibility into configured sources
  exists from day one; only the mutation is deferred. There is no v1 (or
  planned v2) read endpoint for access rules, because `chronyc` has no
  command to report configured allow/deny ACLs at runtime — a v2
  `access-rules` feature would need to parse `chrony.conf` directly.
- **History graphs** (offset/frequency/jitter over time) — needs a
  time-series store; no Prometheus/Grafana in this cluster yet.
- **Alerts engine** — recommended to live in Prometheus once that exists,
  not reimplemented here.
- **Logs/Events feed** — needs chrony `log` directives plus a tailer/parser.
- **Client rate-limiting**.

## Architecture

- Add a second container, `console`, to the existing `timeserver` pod
  (alongside `chronyd`). Set `shareProcessNamespace: true` on the pod.
- Both containers already share (or will share) the `chrony-run` `emptyDir`
  at `/run/chrony`, so `console` can reach chronyd's command socket at
  `/run/chrony/chronyd.sock`.
- `console`'s image bundles the compiled Go binary plus the `chronyc` CLI
  (from the `chrony` apk package). The backend never runs an arbitrary
  shell command — see Security below.
- The pod is `hostNetwork: true` (pod-wide), so `console`'s HTTP port (e.g.
  `8080`) is reachable directly on the VM's IP, same as chronyd's UDP 123.
  DNS/subdomain for it (e.g. `console.iocare.in`) is out of scope here.
- Go backend serves the built React static assets itself via `go:embed` —
  one binary, one container, no separate frontend deploy or CORS setup.

## Security

This is the constraint the whole API design follows: **the web UI never
gets access to raw shell/Linux commands, only a fixed set of high-level
operations.**

- The backend's `internal/chrony` package exposes typed Go functions
  (`GetTracking()`, `GetSources()`, `GetClients()`, `ForceSync()`,
  `StepClock()`, `RestartService()`), each of which shells out to a single
  **fixed, whitelisted** `chronyc` subcommand via `exec.Command` with an
  argv slice — never `sh -c` with interpolated strings, so there is no
  command-injection surface. No endpoint accepts a raw command or forwards
  arbitrary arguments to `chronyc`.
- When v2 adds mutating config endpoints (`PUT /sources`, `PUT
  /access-rules`), any user-supplied value (hostname/IP/CIDR) must be
  validated against a strict allowlist pattern before being placed in an
  argv element — noted here so the constraint isn't forgotten later.
- `console`'s pod-level `securityContext` is **more restricted** than
  chronyd's: `runAsUser: 0` (needed only so it shares UID with chronyd for
  the restart signal — see below), `capabilities: drop: [ALL]`, no
  additions. It does not need `SYS_TIME` (it never adjusts the clock
  itself — chronyd does) or `NET_BIND_SERVICE` (its HTTP port is >1024).
- **Restart mechanism**: `POST /service/restart` finds chronyd's PID via
  the shared process namespace and sends it `SIGTERM`. Because chronyd runs
  as PID 1 of its own container, Kubernetes' normal container restart
  policy brings it back — no Kubernetes API access, RBAC, or
  `ServiceAccount` token is needed by `console` at all.
- Every mutating action (`sync`, `step`, `restart`) is logged server-side
  (timestamp + action + session user) to stdout for a basic audit trail.
- Session cookie is `HttpOnly` + `SameSite=Lax`. Since v1 is served over
  plain HTTP on the VM's public IP, note this in the README as a caveat:
  put a TLS-terminating reverse proxy in front before treating this as
  more than a lightly-protected admin tool.

## API surface (v1)

```
POST /api/auth/login        { username, password } -> sets session cookie
POST /api/auth/logout
GET  /api/auth/session       -> current session state (for frontend boot)

GET  /api/status             -> lightweight health: running, synchronized, stratum, counts
GET  /api/tracking           -> full chronyc tracking detail (refid, offset, freq, root delay/dispersion, leap status)
GET  /api/sources            -> merged sources + sourcestats table
GET  /api/clients            -> client table

POST /api/sync                -> force burst/sync
POST /api/clock/step          -> makestep
POST /api/service/restart     -> SIGTERM chronyd (see Security)
```

All routes above except `POST /api/auth/login` require a valid session
cookie.

## Backend (Go)

- `cmd/server` — entrypoint, loads `.env`, wires routes
- `internal/chrony` — `chronyc` exec wrapper + output parsers, one function
  per whitelisted operation
- `internal/auth` — session middleware, cookie issuance/validation
- `internal/api` — HTTP handlers, thin translation from `internal/chrony`
  structs to JSON

Sessions are stored in an in-memory map (single replica; a pod restart logs
the operator out, which is acceptable for v1).

## Frontend (React + shadcn)

- Vite + React + TypeScript + Tailwind + shadcn/ui
- Pages: Login, Dashboard, Sources, Clients, Diagnostics, sharing a common
  nav shell. Service Operations buttons (Force Sync / Step Clock / Restart)
  live on the Dashboard, each with a shadcn confirm dialog for the two
  clock/service-affecting ones.
- Data refresh via polling (5–10s interval) — no websockets in v1.

## Repo layout

```
console/
  backend/     # Go module (cmd/server, internal/chrony, internal/auth, internal/api)
  frontend/    # React app; `npm run build` output is embedded into the Go binary
  Dockerfile   # multi-stage: build frontend -> build Go binary -> alpine+chrony runtime
```

## k8s changes

- `k8s/deployment.yaml`: add the `console` container to the `timeserver`
  pod spec; add `shareProcessNamespace: true`; mount the existing
  `chrony-run` volume into `console` too; add a `console-env` (Secret, not
  ConfigMap, since it holds the login password) providing the `.env`
  values.
- No new Service is required for v1 given `hostNetwork: true` already
  exposes the console port on the VM directly, same as chronyd's UDP 123.
