# goprecords - Global uptime records

`goprecords` is a Go command-line program that generates uptime reports for hosts based on the input record files from `uptimed`. It supports importing records into SQLite and querying for reports, reporting directly from a stats directory, or (optionally) running as an HTTP server that exposes the same reports and authenticated uploads.

## Features

- Supports multiple categories: `Host`, `Kernel`, `KernelMajor`, and `KernelName`
- Supports multiple metrics: `Boots`, `Uptime`, `Score`, `Downtime`, and `Lifespan`
- Output formats available: `Plaintext`, `Markdown`, `Gemtext`, and `HTML`
- Provides top entries based on the specified limit

Whereas:

* `Boots`: Total count of system boots.
* `Uptime`: Total hosts uptimes.
* `Score`: A meta score, calculated combining all other metrics.
* `Downtime`: Total hosts downtimes.
* `Lifespan`: Downtimes plus uptimes.

## Usage

- **`-version`** — Print version and exit.
- **`import`** — Import `.records` files into a SQLite database (repeatable: replaces existing data).
  - `-stats-dir` (required): Directory containing `*.records` files.
  - `-db`: Database path (default: `goprecords.db`).
- **`query`** — Generate reports from an existing SQLite database.
  - `-db`: Database path (default: `goprecords.db`).
  - `-category`, `-metric`, `-limit`, `-output-format`, `-all`, `-include-kernel`, `-stats-order` (same as below).
- **No subcommand** — Generate reports directly from a stats directory (no DB).
  - `-stats-dir` (required): The uptimed raw record input dir.
  - `-category`: One of `Host`, `Kernel`, `KernelMajor`, `KernelName` (default: `Host`).
  - `-metric`: One of `Boots`, `Uptime`, `Score`, `Downtime`, `Lifespan` (default: `Uptime`).
  - `-limit`: Max entries (default: 20).
  - `-output-format`: `Plaintext`, `Markdown`, `Gemtext`, or `HTML` (default: `Plaintext`).
  - `-all`: Generate all reports except `Kernel`.
  - `-include-kernel`: Include `Kernel` when using `-all`.
  - `-stats-order`: Comma-separated `Category:Metric` order for `-all`.

### Example usage

```bash
# Build (or use mage)
go build -o goprecords ./cmd/goprecords

# Import then query (repeatable)
./goprecords import -stats-dir=~/git/uprecords/stats -db=goprecords.db
./goprecords query -db=goprecords.db -category=Host -metric=Uptime -limit=10 -output-format=Markdown

# Or report directly from files
./goprecords -stats-dir=~/git/uprecords/stats -all -stats-order="Host:Uptime,Host:Boots"
```

### Daemon mode (HTTP API, plain HTTP)

**Backward compatibility:** `import`, `query`, and no-subcommand reporting from `-stats-dir` behave as before. The HTTP server is **additional**; CLI-only workflows are still supported with **no intentional behavior removal**.

`goprecords` serves **plain HTTP only** (Go `net/http` without TLS). Terminate TLS externally—Kubernetes Ingress, a reverse proxy (nginx, Caddy, …), or a load balancer—when clients require HTTPS.

```bash
goprecords --daemon -stats-dir="$HOME/git/uprecords/stats" -listen=":8080"
```

(`-daemon` and `--daemon` are equivalent.)

**Stats directory vs databases**

- **Stats tree** — **`-stats-dir`** (required) or **`GOPRECORDS_STATS_DIR`**: uptimed files (`HOST.txt`, `HOST.records`, …). The daemon builds **`GET /report`** responses from this directory only; it does **not** read the **`import` / `query`** SQLite file (**`-db`**, default `goprecords.db`).
- **Listen address** — **`-listen`** or **`GOPRECORDS_LISTEN`** (default `:8080`).
- **Upload auth database** — **`-auth-db`**: SQLite file holding **per-client** API key hashes (default `<stats-dir>/goprecords-auth.db`). This is separate from the aggregates DB used by **`import`** / **`query`**.

**Endpoints**

- **`GET /health`** and **`GET /livez`** — Liveness (`200`, body `ok`).
- **`GET /readyz`** — Readiness: checks the stats directory is readable/writable (and the auth DB parent directory when **`auth-db`** is outside the stats tree). Returns **`503`** if not ready.
- **`GET /report`** — Same report options as **`query`** and no-subcommand mode (aggregated from the stats directory). **Only `GET`** is supported (other methods → **`405`**). Query parameters use the same names and defaults as the CLI flags; you can also use the aliases **`Category`**, **`Metric`**, and **`OutputFormat`**:
  - **`category`**, **`metric`**, **`limit`** (default `20`)
  - **`output-format`** / **`OutputFormat`**: **`Plaintext`**, **`Markdown`**, **`Gemtext`**, or **`HTML`** (default `Plaintext`); the response **`Content-Type`** matches (e.g. `text/html` for HTML)
  - **`all`**, **`include-kernel`**: booleans (`true` / `false`, `1` / `0`, `yes` / `no`)
  - **`stats-order`**: comma-separated **`Category:Metric`** list when **`all`** is true
- **`PUT /upload/{HOSTNAME}/{kind}`** — Writes one file under the stats directory; **`204`** on success. The **`HOSTNAME`** path segment must match the hostname the client key was issued for when auth is enforced.

**HTTP API reference with curl**

- **`GET /health`**
  - Purpose: basic liveness probe.
  - Success: `200 OK`, body `ok`.
  - Example:

```bash
curl -i "http://127.0.0.1:8080/health"
```

- **`GET /livez`**
  - Purpose: Kubernetes-style liveness probe (same semantics as `/health`).
  - Success: `200 OK`, body `ok`.
  - Example:

```bash
curl -i "http://127.0.0.1:8080/livez"
```

- **`GET /readyz`**
  - Purpose: readiness probe (verifies required directories are readable and writable).
  - Success: `200 OK`, body `ok`.
  - Not ready: `503 Service Unavailable`.
  - Example:

```bash
curl -i "http://127.0.0.1:8080/readyz"
```

- **`GET /report`**
  - Purpose: generate reports from the daemon stats directory.
  - Method: only `GET` (`405` for other methods).
  - Main query parameters:
    - `category`: `Host`, `Kernel`, `KernelMajor`, `KernelName`
    - `metric`: `Boots`, `Uptime`, `Score`, `Downtime`, `Lifespan`
    - `output-format`: `Plaintext`, `Markdown`, `Gemtext`, `HTML`
    - `limit`: integer (default `20`)
    - `all`, `include-kernel`: booleans
    - `stats-order`: comma-separated `Category:Metric`
    - aliases: `Category`, `Metric`, `OutputFormat`
  - Content types:
    - `Plaintext`/`Markdown`: `text/plain; charset=utf-8`
    - `Gemtext`: `text/gemini; charset=utf-8`
    - `HTML`: `text/html; charset=utf-8`
  - Examples:

```bash
curl -fsS "http://127.0.0.1:8080/report?category=Host&metric=Uptime&limit=10&output-format=Plaintext"
curl -fsS "http://127.0.0.1:8080/report?category=Kernel&metric=Boots&output-format=Gemtext"
curl -fsS "http://127.0.0.1:8080/report?Category=Host&Metric=Score&OutputFormat=HTML&limit=5"
curl -fsS "http://127.0.0.1:8080/report?all=true&include-kernel=true&output-format=Markdown"
```

- **`PUT /upload/{HOSTNAME}/{kind}`**
  - Purpose: upload host files into the daemon stats directory.
  - Method: only `PUT` (`405` for other methods).
  - Success: `204 No Content`.
  - Auth behavior:
    - If no keys exist in auth DB: upload is accepted without `Authorization`.
    - If one or more keys exist: requires `Authorization: Bearer <token>`.
    - Missing/invalid header: `401 Unauthorized` (with `WWW-Authenticate: Bearer ...`).
    - Wrong host/token pair: `403 Forbidden`.
  - Examples:

```bash
# without auth (works only when no client keys exist in auth DB)
curl -i -X PUT --data-binary @./myhost.records \
  "http://127.0.0.1:8080/upload/myhost/records"

# with auth (required when keys exist)
TOKEN="..."  # from --create-client-key
curl -i -X PUT --data-binary @./myhost.records \
  -H "Authorization: Bearer $TOKEN" \
  "http://127.0.0.1:8080/upload/myhost/records"
```

- **Create upload client key (CLI companion flow)**
  - This is not an HTTP API endpoint. Key creation is intentionally CLI-only.
  - Command examples:

```bash
goprecords --create-client-key myhost -stats-dir="$HOME/git/uprecords/stats"
goprecords --create-client-key otherbox -auth-db=/var/lib/goprecords/goprecords-auth.db
```

**`GET /report` examples**

```bash
curl -fsS "http://127.0.0.1:8080/report?category=Host&metric=Uptime&limit=10&output-format=Markdown"
curl -fsS "http://127.0.0.1:8080/report?OutputFormat=HTML&limit=5"
```

For **`kind`** in the upload URL, the server creates or overwrites this file in the stats directory:

| `kind` in URL   | File created        |
|-----------------|---------------------|
| `txt`           | `HOSTNAME.txt`      |
| `cur.txt`       | `HOSTNAME.cur.txt`  |
| `records`       | `HOSTNAME.records`  |
| `os.txt`        | `HOSTNAME.os.txt`   |
| `cpuinfo.txt`   | `HOSTNAME.cpuinfo.txt` |

**Per-client keys** — When the auth database has **at least one** key, each upload must send **`Authorization: Bearer <token>`** where **`<token>`** is the secret printed once by **`goprecords --create-client-key HOSTNAME`** (or **`-create-client-key`**). The token is validated for that **`HOSTNAME`** only (same string as in the upload path). You need **`-stats-dir`** (to derive the default auth DB path) or **`-auth-db`**:

```bash
goprecords --create-client-key myhost -stats-dir="$HOME/git/uprecords/stats"
goprecords --create-client-key otherbox -auth-db=/var/lib/goprecords/goprecords-auth.db
```

Creating a key for a hostname that already has one **replaces** the previous secret.

Example **`curl`** uploads with the **`Authorization`** header:

```bash
TOKEN="…"   # output of --create-client-key

curl -fsS -X PUT --data-binary @./myhost.records \
  -H "Authorization: Bearer $TOKEN" \
  "http://127.0.0.1:8080/upload/myhost/records"

curl -fsS -X PUT --data-binary @./myhost.txt \
  -H "Authorization: Bearer $TOKEN" \
  "http://127.0.0.1:8080/upload/myhost/txt"

curl -fsS -X PUT --data-binary @./myhost.cur.txt \
  -H "Authorization: Bearer $TOKEN" \
  "http://127.0.0.1:8080/upload/myhost/cur.txt"

curl -fsS -X PUT --data-binary @./myhost.os.txt \
  -H "Authorization: Bearer $TOKEN" \
  "http://127.0.0.1:8080/upload/myhost/os.txt"

curl -fsS -X PUT --data-binary @./myhost.cpuinfo.txt \
  -H "Authorization: Bearer $TOKEN" \
  "http://127.0.0.1:8080/upload/myhost/cpuinfo.txt"
```

### Setting up a new upload client

Use the same **`HOSTNAME`** everywhere: the argument to **`--create-client-key`**, each **`PUT /upload/HOSTNAME/{kind}`** URL, and (by convention) the basename of your uptimed files under the stats directory. Pick a stable short name (for example the first label of **`hostname -f`**).

**1. Issue a token on the server** (where the daemon’s stats tree and auth DB live). The plaintext token is printed **once**; store it only on the client.

```bash
# Daemon on this host; auth DB defaults to <stats-dir>/goprecords-auth.db
goprecords --create-client-key myhost -stats-dir=/var/lib/goprecords/stats

# Or pass auth DB explicitly
goprecords --create-client-key myhost -auth-db=/var/lib/goprecords/goprecords-auth.db
```

If **`goprecords` runs in Kubernetes**, run the same subcommand in the pod (adjust namespace and stats mount if needed):

```bash
kubectl exec -n services deployment/goprecords -- \
  goprecords --create-client-key myhost -stats-dir=/data/stats
```

Re-running **`--create-client-key`** for the same hostname **replaces** the previous token; update the client immediately after rotation.

**2. On the client**, keep the token in a root-owned or user-private file (for example mode **`600`**) and use **`curl`** with **`Authorization: Bearer`** as in the examples above. Set the base URL to your daemon (plain HTTP behind the pod or **`https://`** behind ingress).

**3. Source files** — Typical uptimed locations: **`/var/spool/uptimed/records`** (many Linux distributions and **FreeBSD** **`pkg`** installs), **`/usr/local/var/uptimed/records`** (macOS/Homebrew), **`/var/db/uptimed/records`** (some other BSD layouts). The upload client checks those paths in that order. Use **`uprecords`** from the **`uptimed`** package to regenerate **`txt`** and **`cur.txt`**-equivalent content when you upload those kinds. **`os.txt`** and **`cpuinfo.txt`** are often **`/etc/os-release`** and **`/proc/cpuinfo`** on Linux.

**4. Automate (optional)** — A **`oneshot`** systemd service can call a small script that **`PUT`**s each present file; a **`systemd.timer`** (for example **`OnCalendar=hourly`**) triggers it. For a **user** timer, enable lingering if uploads should run while no graphical session is active:

```bash
sudo loginctl enable-linger "$USER"
```

If there are **no** keys in the auth database, uploads are accepted without **`Authorization`** (useful for local testing only).

### Manual hourly upload (single host, not config-managed)

Use this when the machine is **not** deployed from a Rex/Ansible repo. The unified script is **`scripts/goprecords-upload-client.sh`** (POSIX **`sh`**: Linux, FreeBSD, OpenBSD; runs as root or as a regular user). A copy is also kept in **`contrib/`** for backward compatibility.

The script works on all host types:

| Host class | OS | Privilege | Token path |
|------------|----|-----------|------------|
| **f0–f3** | FreeBSD | root via `doas` | `/etc/goprecords-upload.token` |
| **pi0–pi3** | Rocky Linux (aarch64) | root via `sudo` | `/etc/goprecords-upload.token` |
| **earth** (laptop) | Fedora / Linux | user (no root) | `$XDG_CONFIG_HOME/goprecords-upload-<HOST>/token` |
| **blowfish**, **fishfinger** | OpenBSD | root (Rex) | deployed by Rex from template |

**SSH vs upload name:** **`GOPRECORDS_HOST`** must stay the **short** stats name (**`f0`**, **`pi2`**, **`earth`**). For **SSH**, use a real host — e.g. **`ssh -p 22 paul@f0.lan.buetow.org`** or **`paul@192.168.1.130`**, not **`f0.lan`** (incomplete / invalid). OpenBSD frontends may use **port 2** in Rex; **f0–f3** and **Pis** use **port 22**.

**Privilege on FreeBSD:** many hosts (e.g. **f0–f3**) ship **`doas`** and no **`sudo`**. Use **`doas`** for install, writing the token, and one-off test runs below; on Linux (Pis) use **`sudo`**.

**Token path (non-root):** when run as a non-root user the script defaults to **`${XDG_CONFIG_HOME:-$HOME/.config}/goprecords-upload-${GOPRECORDS_HOST}/token`** (mode **`600`**). Override with **`GOPRECORDS_TOKEN_FILE`**.

**On each host**

1. Install **`curl`** and **`uptimed`**, and ensure **`uptimed`** is running.
2. Install the script (path is up to you; **`755`**, owned by **`root`** for system-wide use or **`700`** in `~/.local/bin` for user installs):

   **Linux (root)**

   ```bash
   sudo install -m 755 scripts/goprecords-upload-client.sh /usr/local/bin/goprecords-upload-client.sh
   ```

   **FreeBSD (root)**

   ```bash
   doas install -m 755 scripts/goprecords-upload-client.sh /usr/local/bin/goprecords-upload-client.sh
   ```

   **Linux (user, e.g. earth)**

   ```bash
   install -m 700 scripts/goprecords-upload-client.sh ~/.local/bin/goprecords-upload-client.sh
   ```

3. Create a token on the goprecords server (hostname must match **`GOPRECORDS_HOST`** exactly, e.g. **`f0`**, **`pi2`**, **`earth`**):

   ```bash
   kubectl exec -n services deployment/goprecords -- \
     goprecords --create-client-key f0 -stats-dir=/data/stats
   ```

4. Save the printed secret, mode **`600`**:

   **Linux / FreeBSD (root)**

   ```bash
   # Linux
   umask 077
   sudo sh -c ‘cat > /etc/goprecords-upload.token’
   sudo chmod 600 /etc/goprecords-upload.token
   # FreeBSD
   umask 077
   doas sh -c ‘cat > /etc/goprecords-upload.token’
   doas chmod 600 /etc/goprecords-upload.token
   ```

   **Linux (user, e.g. earth)**

   ```bash
   mkdir -p ~/.config/goprecords-upload-earth
   umask 077
   cat > ~/.config/goprecords-upload-earth/token
   ```

5. **FreeBSD — hourly cron** (example for **`f0`**; use **`/etc/crontab`** or **`root`**’s crontab). **`curl`** must be on **`PATH`** for cron (often **`/usr/local/bin`**):

   ```cron PATH=/bin:/sbin:/usr/bin:/usr/sbin:/usr/local/bin:/usr/local/sbin
   0 * * * * root /usr/bin/env GOPRECORDS_HOST=f0 /usr/local/bin/goprecords-upload-client.sh
   ```

   Repeat for **`f1`**, **`f2`**, **`f3`**, … with the matching **`GOPRECORDS_HOST`**.

6. **Linux (systemd) — hourly timer, system-wide** (example for **`pi0`**). **`/etc/goprecords-upload.env`** (mode **`644`**, root):

   ```ini
   GOPRECORDS_HOST=pi0
   ```

   **`/etc/systemd/system/goprecords-upload.service`**:

   ```ini
   [Unit]
   Description=Upload uptimed stats to goprecords

   [Service]
   Type=oneshot
   EnvironmentFile=/etc/goprecords-upload.env
   Environment=GOPRECORDS_TOKEN_FILE=/etc/goprecords-upload.token
   ExecStart=/usr/local/bin/goprecords-upload-client.sh
   ```

   **`/etc/systemd/system/goprecords-upload.timer`**:

   **`OnActiveSec`** runs once soon after **`enable --now`**, so the first upload is not delayed until the next full hour (otherwise run **`systemctl start goprecords-upload.service`** manually once).

   ```ini
   [Unit]
   Description=Hourly uptimed upload to goprecords

   [Timer]
   OnCalendar=hourly
   OnActiveSec=90s
   RandomizedDelaySec=300
   Persistent=true

   [Install]
   WantedBy=timers.target
   ```

   Then:

   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable --now goprecords-upload.timer
   ```

   Use a different **`GOPRECORDS_HOST`** in **`/etc/goprecords-upload.env`** on **`pi1`**, **`pi2`**, **`pi3`**.

7. **Linux (systemd) — hourly timer, user session** (example for **`earth`** laptop). **`~/.config/systemd/user/goprecords-upload-earth.service`**:

   ```ini
   [Unit]
   Description=Upload uptimed stats to goprecords

   [Service]
   Type=oneshot
   Environment=GOPRECORDS_HOST=earth
   ExecStart=%h/.local/bin/goprecords-upload-client.sh
   ```

   **`~/.config/systemd/user/goprecords-upload-earth.timer`**:

   ```ini
   [Unit]
   Description=Hourly uptimed upload to goprecords (earth)

   [Timer]
   OnCalendar=hourly
   OnActiveSec=90s
   RandomizedDelaySec=300
   Persistent=true

   [Install]
   WantedBy=timers.target
   ```

   Then (enable lingering so uploads continue without an active login session):

   ```bash
   sudo loginctl enable-linger "$USER"
   systemctl --user daemon-reload
   systemctl --user enable --now goprecords-upload-earth.timer
   ```

**Environment (optional)**

| Variable | Default (root) | Default (non-root) |
|----------|----------------|--------------------|
| **`GOPRECORDS_HOST`** | (required) | (required) |
| **`GOPRECORDS_TOKEN_FILE`** | **`/etc/goprecords-upload.token`** | **`$XDG_CONFIG_HOME/goprecords-upload-<HOST>/token`** |
| **`GOPRECORDS_BASE_URL`** | **`https://goprecords.f3s.buetow.org`** | same |

**Test one run**

**Linux (root)**

```bash
sudo env GOPRECORDS_HOST=pi0 /usr/local/bin/goprecords-upload-client.sh
```

**FreeBSD (root)**

```bash
doas env GOPRECORDS_HOST=f0 /usr/local/bin/goprecords-upload-client.sh
```

**Linux (user)**

```bash
GOPRECORDS_HOST=earth ~/.local/bin/goprecords-upload-client.sh
```

## Test

Run the test subcommand (fixture comparison and import/query parity):

```bash
./goprecords test
```

Or run Go unit tests:

```bash
go test ./...
# or: mage test
```

## End-to-end usage

### Use `uptimed` to produce uptime statistics

First, you need to generate uptime statistics from all hosts by installing and running `uptimed`. There's a package available for most common Linux and *BSD distributions nowadays. It's also available for macOS (Darwin) via Homebrew. For example, under Fedora, run `sudo dnf install uptimed`. https://github.com/rpodgorny/uptimed

### Collect all uptime records to a central location

Second, you must collect the `records` files produced by `uptimed`. Which are the raw uptime statistic files continuously updated by `uptimed`. Depending on the operating system used, the location of the records file can vary. It is advisable to store all the record files in a central git repository. 

An example records file looks like this:

```
11175544:1658053426:OpenBSD 7.1
10033984:1669229566:OpenBSD 7.2
7701011:1642849465:OpenBSD 7.0
3900089:1650550947:OpenBSD 7.1
3573912:1654452258:OpenBSD 7.1
2132201:1640713822:OpenBSD 7.0
88762:1640625045:OpenBSD 7.0
18452:1640603646:OpenBSD 7.0
3408:1642846040:OpenBSD 7.0
2315:1640622113:OpenBSD 7.0
1190:1654451052:OpenBSD 7.1
334:1650550601:OpenBSD 7.1
310:1669229245:OpenBSD 7.2
310:1640624443:OpenBSD 7.0
261:1640624769:OpenBSD 7.0
144:1669229090:OpenBSD 7.2
```

... whereas the first number is the total uptime since boot, and the second is the boot time. The last column identifies the operating system and kernel version.

`guprecords` does not provide any out-of-the-box solution for the collection part. I use a quick-and-dirty `Makefile` in a `uptimes.git` repository, where I can run `make push` to manually collect and push the uptime statistics of the current host to the git repository. I log in to all my machines anyway, sooner or later, and I also automatically run a shell script (via my login RC file) to re-collect the stats when they weren't collected for a week or so. So this solution is "just good enough" for me for now:

```
manual:
    records_path=/var/spool/uptimed/records; \
    test -f /usr/local/var/uptimed/records && records_path=/usr/local/var/uptimed/records; \
    test -f /var/db/uptimed/records && records_path=/var/db/uptimed/records; \
    cp $$records_path ./stats/$$(hostname | cut -d. -f1).records
    uprecords -a -m 100 > ./stats/$$(hostname | cut -d. -f1).txt
    uprecords -a | grep '^->' > ./stats/$$(hostname | cut -d. -f1).cur.txt
    git add ./stats/*
    git commit -a -m 'new uptime
push: manual
    git add ./stats/$(hostname)*
    git commit -a -m 'new stats'
    git pull origin master
    git push origin master
```

### Generate global uptime stats

Third, import and query (or report from files):

```
./goprecords import -stats-dir=$HOME/git/uprecords/stats -db=goprecords.db
./goprecords query -db=goprecords.db -all
```

Or without a database:

```
./goprecords -stats-dir=$HOME/git/uprecords/stats -all
```

... to generate something like the following:

## Top 20 Boots's by Host

Boots is the total number of host boots over the entire lifespan.

```
+-----+----------------+-------+------------------------------+
| Pos |           Host | Boots |                  Last Kernel |
+-----+----------------+-------+------------------------------+
|  1. |  alphacentauri |   671 |      FreeBSD 11.4-RELEASE-p7 |
|  2. |           mars |   207 |          Linux 3.2.0-4-amd64 |
|  3. |         *earth |   182 | Linux 6.14.5-300.fc42.x86_64 |
|  4. |       callisto |   153 |  Linux 4.0.4-303.fc22.x86_64 |
|  5. |       dionysus |   136 |     FreeBSD 13.0-RELEASE-p11 |
|  6. |      tauceti-e |   120 |          Linux 3.2.0-4-amd64 |
|  7. |      *makemake |    76 |  Linux 6.9.9-200.fc40.x86_64 |
|  8. |        *uranus |    59 |                  NetBSD 10.1 |
|  9. |          pluto |    51 |          Linux 3.2.0-4-amd64 |
| 10. |      mega15289 |    50 |                Darwin 23.4.0 |
| 11. |    *fishfinger |    43 |                  OpenBSD 7.6 |
| 12. |          *t450 |    43 |         FreeBSD 14.2-RELEASE |
| 13. |   *mega-m3-pro |    41 |                Darwin 24.4.0 |
| 14. |         phobos |    40 |      Linux 3.4.0-CM-g1dd7cdf |
| 15. |       mega8477 |    40 |                Darwin 13.4.0 |
| 16. |      *blowfish |    38 |                  OpenBSD 7.6 |
| 17. |            sun |    33 |     FreeBSD 10.3-RELEASE-p24 |
| 18. |            *f2 |    25 |      FreeBSD 14.2-RELEASE-p1 |
| 19. |            *f1 |    20 |      FreeBSD 14.2-RELEASE-p1 |
| 20. |           moon |    20 |      FreeBSD 14.0-RELEASE-p3 |
+-----+----------------+-------+------------------------------+
```

## Top 20 Uptime's by Host

Uptime is the total uptime of a host over the entire lifespan.

```
+-----+----------------+-----------------------------+-----------------------------------+
| Pos |           Host |                      Uptime |                       Last Kernel |
+-----+----------------+-----------------------------+-----------------------------------+
|  1. |         vulcan |   4 years, 5 months, 6 days | Linux 3.10.0-1160.81.1.el7.x86_64 |
|  2. |            sun |  3 years, 9 months, 26 days |          FreeBSD 10.3-RELEASE-p24 |
|  3. |        *uranus |   3 years, 9 months, 5 days |                       NetBSD 10.1 |
|  4. |      *blowfish |  3 years, 5 months, 16 days |                       OpenBSD 7.6 |
|  5. |         *earth |   3 years, 5 months, 6 days |      Linux 6.14.5-300.fc42.x86_64 |
|  6. |          uugrn |   3 years, 5 months, 5 days |           FreeBSD 11.2-RELEASE-p4 |
|  7. |      deltavega |  3 years, 1 months, 21 days | Linux 3.10.0-1160.11.1.el7.x86_64 |
|  8. |          pluto | 2 years, 10 months, 29 days |               Linux 3.2.0-4-amd64 |
|  9. |    *fishfinger |  2 years, 9 months, 11 days |                       OpenBSD 7.6 |
| 10. |        tauceti |  2 years, 3 months, 19 days |               Linux 3.2.0-4-amd64 |
| 11. |      mega15289 | 1 years, 12 months, 17 days |                     Darwin 23.4.0 |
| 12. |      tauceti-f |  1 years, 9 months, 18 days |               Linux 3.2.0-3-amd64 |
| 13. |          *t450 |  1 years, 4 months, 28 days |              FreeBSD 14.2-RELEASE |
| 14. |       mega8477 |  1 years, 3 months, 25 days |                     Darwin 13.4.0 |
| 15. |          host0 |   1 years, 3 months, 9 days |          FreeBSD 6.2-RELEASE-p5   |
| 16. |      *makemake |   1 years, 3 months, 5 days |       Linux 6.9.9-200.fc40.x86_64 |
| 17. |      tauceti-e |  1 years, 2 months, 20 days |               Linux 3.2.0-4-amd64 |
| 18. |   *mega-m3-pro | 0 years, 12 months, 13 days |                     Darwin 24.4.0 |
| 19. |       callisto | 0 years, 10 months, 31 days |       Linux 4.0.4-303.fc22.x86_64 |
| 20. |  alphacentauri | 0 years, 10 months, 28 days |           FreeBSD 11.4-RELEASE-p7 |
+-----+----------------+-----------------------------+-----------------------------------+
```

## Top 20 Score's by Host

Score is calculated by combining all other metrics.

```
+-----+----------------+-------+-----------------------------------+
| Pos |           Host | Score |                       Last Kernel |
+-----+----------------+-------+-----------------------------------+
|  1. |        *uranus |   342 |                       NetBSD 10.1 |
|  2. |         vulcan |   275 | Linux 3.10.0-1160.81.1.el7.x86_64 |
|  3. |            sun |   238 |          FreeBSD 10.3-RELEASE-p24 |
|  4. |         *earth |   236 |      Linux 6.14.5-300.fc42.x86_64 |
|  5. |      *blowfish |   218 |                       OpenBSD 7.6 |
|  6. |          uugrn |   211 |           FreeBSD 11.2-RELEASE-p4 |
|  7. |  alphacentauri |   201 |           FreeBSD 11.4-RELEASE-p7 |
|  8. |      deltavega |   193 | Linux 3.10.0-1160.11.1.el7.x86_64 |
|  9. |          pluto |   182 |               Linux 3.2.0-4-amd64 |
| 10. |    *fishfinger |   176 |                       OpenBSD 7.6 |
| 11. |       dionysus |   156 |          FreeBSD 13.0-RELEASE-p11 |
| 12. |      mega15289 |   147 |                     Darwin 23.4.0 |
| 13. |        tauceti |   141 |               Linux 3.2.0-4-amd64 |
| 14. |      *makemake |   131 |       Linux 6.9.9-200.fc40.x86_64 |
| 15. |      tauceti-f |   108 |               Linux 3.2.0-3-amd64 |
| 16. |          *t450 |   106 |              FreeBSD 14.2-RELEASE |
| 17. |      tauceti-e |    96 |               Linux 3.2.0-4-amd64 |
| 18. |       callisto |    86 |       Linux 4.0.4-303.fc22.x86_64 |
| 19. |       mega8477 |    80 |                     Darwin 13.4.0 |
| 20. |          host0 |    76 |          FreeBSD 6.2-RELEASE-p5   |
+-----+----------------+-------+-----------------------------------+
```

## Top 20 Downtime's by Host

Downtime is the total downtime of a host over the entire lifespan.

```
+-----+----------------+-----------------------------+------------------------------+
| Pos |           Host |                    Downtime |                  Last Kernel |
+-----+----------------+-----------------------------+------------------------------+
|  1. |       dionysus |  8 years, 3 months, 16 days |     FreeBSD 13.0-RELEASE-p11 |
|  2. |        *uranus |  6 years, 7 months, 31 days |                  NetBSD 10.1 |
|  3. |  alphacentauri | 5 years, 11 months, 18 days |      FreeBSD 11.4-RELEASE-p7 |
|  4. |      *makemake |   3 years, 2 months, 2 days |  Linux 6.9.9-200.fc40.x86_64 |
|  5. |           moon |   2 years, 1 months, 1 days |      FreeBSD 14.0-RELEASE-p3 |
|  6. |       callisto |  1 years, 5 months, 15 days |  Linux 4.0.4-303.fc22.x86_64 |
|  7. |      mega15289 |  1 years, 4 months, 24 days |                Darwin 23.4.0 |
|  8. |          *t450 |  1 years, 2 months, 13 days |         FreeBSD 14.2-RELEASE |
|  9. |           mars |  1 years, 2 months, 10 days |          Linux 3.2.0-4-amd64 |
| 10. |      tauceti-e |  0 years, 12 months, 9 days |          Linux 3.2.0-4-amd64 |
| 11. |         sirius |  0 years, 8 months, 20 days |   Linux 2.6.32-042stab111.12 |
| 12. |         *earth |  0 years, 6 months, 19 days | Linux 6.14.5-300.fc42.x86_64 |
| 13. |         deimos |  0 years, 5 months, 15 days |  Linux 4.4.5-300.fc23.x86_64 |
| 14. |            *f0 |  0 years, 4 months, 20 days |      FreeBSD 14.2-RELEASE-p1 |
| 15. |            *f2 |  0 years, 4 months, 19 days |      FreeBSD 14.2-RELEASE-p1 |
| 16. |            *f1 |  0 years, 4 months, 18 days |      FreeBSD 14.2-RELEASE-p1 |
| 17. |        joghurt |   0 years, 2 months, 9 days |    FreeBSD 7.0-PRERELEASE    |
| 18. |          host0 |   0 years, 2 months, 1 days |     FreeBSD 6.2-RELEASE-p5   |
| 19. |      fibonacci |  0 years, 1 months, 11 days |     FreeBSD 5.3-RELEASE-p15  |
| 20. |          cobol |   0 years, 1 months, 8 days |     FreeBSD 10.1-RELEASE-p24 |
+-----+----------------+-----------------------------+------------------------------+
```

## Top 20 Lifespan's by Host

Lifespan is the total uptime + the total downtime of a host.

```
+-----+----------------+-----------------------------+-----------------------------------+
| Pos |           Host |                    Lifespan |                       Last Kernel |
+-----+----------------+-----------------------------+-----------------------------------+
|  1. |        *uranus |  10 years, 4 months, 5 days |                       NetBSD 10.1 |
|  2. |       dionysus |  8 years, 6 months, 17 days |          FreeBSD 13.0-RELEASE-p11 |
|  3. |  alphacentauri |  6 years, 9 months, 13 days |           FreeBSD 11.4-RELEASE-p7 |
|  4. |         vulcan |   4 years, 5 months, 6 days | Linux 3.10.0-1160.81.1.el7.x86_64 |
|  5. |      *makemake |   4 years, 4 months, 7 days |       Linux 6.9.9-200.fc40.x86_64 |
|  6. |         *earth | 3 years, 10 months, 23 days |      Linux 6.14.5-300.fc42.x86_64 |
|  7. |            sun |  3 years, 10 months, 2 days |          FreeBSD 10.3-RELEASE-p24 |
|  8. |      *blowfish |  3 years, 5 months, 17 days |                       OpenBSD 7.6 |
|  9. |          uugrn |   3 years, 5 months, 5 days |           FreeBSD 11.2-RELEASE-p4 |
| 10. |      mega15289 |   3 years, 4 months, 9 days |                     Darwin 23.4.0 |
| 11. |      deltavega |  3 years, 1 months, 21 days | Linux 3.10.0-1160.11.1.el7.x86_64 |
| 12. |          pluto | 2 years, 10 months, 30 days |               Linux 3.2.0-4-amd64 |
| 13. |    *fishfinger |  2 years, 9 months, 13 days |                       OpenBSD 7.6 |
| 14. |          *t450 |   2 years, 6 months, 9 days |              FreeBSD 14.2-RELEASE |
| 15. |           moon |  2 years, 4 months, 25 days |           FreeBSD 14.0-RELEASE-p3 |
| 16. |        tauceti |  2 years, 3 months, 22 days |               Linux 3.2.0-4-amd64 |
| 17. |       callisto |  2 years, 3 months, 13 days |       Linux 4.0.4-303.fc22.x86_64 |
| 18. |      tauceti-e |  2 years, 1 months, 29 days |               Linux 3.2.0-4-amd64 |
| 19. |      tauceti-f |  1 years, 9 months, 20 days |               Linux 3.2.0-3-amd64 |
| 20. |           mars |  1 years, 8 months, 19 days |               Linux 3.2.0-4-amd64 |
+-----+----------------+-----------------------------+-----------------------------------+
```

## Top 20 Boots's by KernelMajor

Boots is the total number of host boots over the entire lifespan.

```
+-----+----------------+-------+
| Pos |    KernelMajor | Boots |
+-----+----------------+-------+
|  1. |  FreeBSD 10... |   551 |
|  2. |     Linux 3... |   550 |
|  3. |     Linux 5... |   162 |
|  4. |    *Linux 6... |   162 |
|  5. |     Linux 4... |   161 |
|  6. |  FreeBSD 11... |   153 |
|  7. |  FreeBSD 13... |   116 |
|  8. |  *OpenBSD 7... |    91 |
|  9. | *FreeBSD 14... |    79 |
| 10. |   Darwin 13... |    40 |
| 11. |   Darwin 23... |    33 |
| 12. |   FreeBSD 5... |    25 |
| 13. |     Linux 2... |    22 |
| 14. |   Darwin 21... |    17 |
| 15. |   Darwin 15... |    15 |
| 16. |  *Darwin 24... |    13 |
| 17. |   Darwin 22... |    12 |
| 18. |   Darwin 18... |    11 |
| 19. |   OpenBSD 4... |    10 |
| 20. |   FreeBSD 6... |    10 |
+-----+----------------+-------+
```

## Top 20 Uptime's by KernelMajor

Uptime is the total uptime of a host over the entire lifespan.

```
+-----+----------------+------------------------------+
| Pos |    KernelMajor |                       Uptime |
+-----+----------------+------------------------------+
|  1. |     Linux 3... | 15 years, 10 months, 25 days |
|  2. |  *OpenBSD 7... |   6 years, 9 months, 24 days |
|  3. |  FreeBSD 10... |    5 years, 9 months, 9 days |
|  4. |     Linux 5... |  4 years, 10 months, 21 days |
|  5. |    *Linux 6... |    2 years, 8 months, 3 days |
|  6. |     Linux 4... |   2 years, 7 months, 22 days |
|  7. |  FreeBSD 11... |   2 years, 4 months, 28 days |
|  8. |     Linux 2... |  1 years, 11 months, 21 days |
|  9. | *FreeBSD 14... |    1 years, 6 months, 1 days |
| 10. |   Darwin 13... |   1 years, 3 months, 25 days |
| 11. |   FreeBSD 6... |    1 years, 3 months, 9 days |
| 12. |   Darwin 23... |   0 years, 11 months, 9 days |
| 13. |   OpenBSD 4... |   0 years, 8 months, 12 days |
| 14. |   Darwin 21... |    0 years, 8 months, 2 days |
| 15. |   Darwin 18... |    0 years, 7 months, 5 days |
| 16. |   Darwin 22... |   0 years, 6 months, 22 days |
| 17. |   Darwin 15... |   0 years, 6 months, 15 days |
| 18. |   FreeBSD 5... |   0 years, 5 months, 18 days |
| 19. |  *Darwin 24... |   0 years, 4 months, 16 days |
| 20. |  FreeBSD 13... |    0 years, 4 months, 2 days |
+-----+----------------+------------------------------+
```

## Top 20 Score's by KernelMajor

Score is calculated by combining all other metrics.

```
+-----+----------------+-------+
| Pos |    KernelMajor | Score |
+-----+----------------+-------+
|  1. |     Linux 3... |  1045 |
|  2. |  *OpenBSD 7... |   435 |
|  3. |  FreeBSD 10... |   406 |
|  4. |     Linux 5... |   317 |
|  5. |    *Linux 6... |   179 |
|  6. |     Linux 4... |   175 |
|  7. |  FreeBSD 11... |   159 |
|  8. |     Linux 2... |   121 |
|  9. | *FreeBSD 14... |    98 |
| 10. |   Darwin 13... |    80 |
| 11. |   FreeBSD 6... |    75 |
| 12. |   Darwin 23... |    56 |
| 13. |   OpenBSD 4... |    39 |
| 14. |   Darwin 21... |    38 |
| 15. |   Darwin 18... |    32 |
| 16. |   Darwin 22... |    30 |
| 17. |   Darwin 15... |    29 |
| 18. |  FreeBSD 13... |    25 |
| 19. |   FreeBSD 5... |    25 |
| 20. |  *Darwin 24... |    21 |
+-----+----------------+-------+
```

## Top 20 Boots's by KernelName

Boots is the total number of host boots over the entire lifespan.

```
+-----+------------+-------+
| Pos | KernelName | Boots |
+-----+------------+-------+
|  1. |     *Linux |  1057 |
|  2. |   *FreeBSD |   944 |
|  3. |    *Darwin |   146 |
|  4. |   *OpenBSD |   101 |
|  5. |    *NetBSD |     1 |
+-----+------------+-------+
```

## Top 20 Uptime's by KernelName

Uptime is the total uptime of a host over the entire lifespan.

```
+-----+------------+-----------------------------+
| Pos | KernelName |                      Uptime |
+-----+------------+-----------------------------+
|  1. |     *Linux | 27 years, 8 months, 25 days |
|  2. |   *FreeBSD |  11 years, 5 months, 3 days |
|  3. |   *OpenBSD |   7 years, 5 months, 5 days |
|  4. |    *Darwin |   4 years, 8 months, 4 days |
|  5. |    *NetBSD |   0 years, 1 months, 1 days |
+-----+------------+-----------------------------+
```

## Top 20 Score's by KernelName

Score is calculated by combining all other metrics.

```
+-----+------------+-------+
| Pos | KernelName | Score |
+-----+------------+-------+
|  1. |     *Linux |  1839 |
|  2. |   *FreeBSD |   799 |
|  3. |   *OpenBSD |   474 |
|  4. |    *Darwin |   304 |
|  5. |    *NetBSD |     2 |
+-----+------------+-------+
```
