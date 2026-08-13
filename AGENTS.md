# AGENTS.md - AI Agent Instructions

## Project Overview

`goprecords` is a Go command-line program that generates uptime reports for hosts based on input record files from `uptimed`. It supports importing records into SQLite and querying for reports, or reporting directly from a stats directory.

## Build & Test Commands

```bash
# Build
go build -o goprecords ./cmd/goprecords
# or: mage

# Run tests
go test ./...
# Full check (unit + race): mage test
# Race only: mage testRace

# Run integration tests
./goprecords test

# Install to GOPATH/bin
mage install
```

## Docker image & f3s deploy

Image build and registry push live in the **`conf`** repo (`f3s/goprecords`), not in this tree. The Justfile there builds from **`docker-image/Justfile`**, which sets **`SRC`** to this **`goprecords`** checkout (default `/home/paul/git/goprecords`).

**Release checklist**

1. Bump **`internal/version/version.go`** (`Tag`, semver).
2. In **`conf`**: set **`docker-image/Justfile`** `TAG` to the same version; set **`helm-chart/templates/deployment.yaml`** `image:` to `registry.lan.buetow.org:30001/goprecords:<Tag>`; update **`f3s/goprecords/README.md`** example tags if present.
3. From **`conf/f3s/goprecords`**: run **`just build-push`** (tags and pushes **`r0.lan.buetow.org:30001/goprecords:<Tag>`**; the cluster pulls via **`registry.lan.buetow.org:30001`**).
4. Commit and tag **`goprecords`**; push branch and tag to `origin`.
5. Commit **`conf`**; push **`master`** to Codeberg (remote **`master`**) **and** to **`forgejo`** (`git push forgejo master`, `ssh://git@code.f3s.buetow.org:2022/snonux/conf.git`) — Argo pulls from the in-cluster Forgejo (`forgejo.services.svc.cluster.local/snonux/conf.git`). The old **`r0`**/**`r1`** git server remotes are retired and read-only.
6. Sync the app: from **`conf/f3s/goprecords`**, **`just sync`** (or wait for Argo automated sync). Confirm with **`kubectl rollout status deployment/goprecords -n services`** and the deployment image tag.

Utility targets in **`conf/f3s/goprecords/Justfile`**: **`status`**, **`logs`**, **`restart`**.

## Project Structure

```
goprecords/
├── cmd/goprecords/main.go    # Entry point, CLI handling
├── internal/
│   ├── goprecords/           # Core logic (db, aggregate, report, parse, order, types)
│   └── version/version.go    # Version constant
├── Dockerfile                # Multi-stage image (daemon binary)
├── Magefile.go               # Build automation
├── fixtures/                 # Test fixtures and expected outputs
└── go.mod                    # Go 1.21, modernc.org/sqlite
```

## Code Style

- Go 1.21
- No comments in code unless explicitly requested
- Functions should be ~30 lines; refactor when approaching 50 lines
- Move code from main.go to ./internal package when main.go grows too large
- Follow existing patterns in the codebase

## Categories & Metrics

- Categories: `Host`, `Kernel`, `KernelMajor`, `KernelName`
- Metrics: `Boots`, `Uptime`, `Score`, `Downtime`, `Lifespan`
- Output formats: `Plaintext`, `Markdown`, `Gemtext`, `HTML`
- `Downtime` and `Lifespan` metrics only apply to `Host` category
- Host classification (`server`/`workstation`/`hybrid`/`unknown`, shown as `S`/`W`/`H`/`U` in the `Cls` column of host reports) lives in `internal/hostclass`; source of truth is a `HOSTNAME.class` file in the stats dir, mirrored into the `host_class` table on `import`

## Key Types

- `Category`, `Metric`, `OutputFormat` - enums for report configuration
- `Aggregate` - per-entity stats (host, kernel, etc.)
- `HostAggregate` - extends Aggregate with LastKernel, LastUpdated, Class, Downtime, Lifespan
- `hostclass.Class` - per-host classification (Unknown, Server, Workstation, Hybrid)
