# Upload client installation

`scripts/goprecords-upload-client.sh` is a POSIX `sh` script that uploads
uptimed record files to a running `goprecords` daemon. It works on FreeBSD,
Linux, and OpenBSD, and runs as root or as a regular user.

`scripts/goprecords-upload-client-darwin.fish` is a fish-shell variant for
macOS (Darwin). It handles both Intel (`/usr/local/var/uptimed`) and Apple
Silicon (`/opt/homebrew/var/uptimed`) Homebrew prefixes and uses macOS-native
commands (`sw_vers`, `sysctl`) for OS and CPU identification.

A copy of the POSIX script is also kept in `contrib/` for backward compatibility.

## Prerequisites

Install `curl` and `uptimed` on every client host and ensure `uptimed` is
running before setting up uploads.

| OS | Package manager |
|----|-----------------|
| FreeBSD | `pkg install curl uptimed` |
| Rocky Linux / Fedora | `sudo dnf install curl uptimed` |
| OpenBSD | `pkg_add curl uptimed` |
| macOS (Intel) | `brew install curl uptimed` |
| macOS (Apple Silicon) | `brew install curl uptimed` |

## Token path

| Privilege | Path |
|-----------|------|
| root | `/etc/goprecords-upload.token` (mode `600`) |
| non-root | `${XDG_CONFIG_HOME:-$HOME/.config}/goprecords-upload-<HOST>/token` (mode `600`) |

Override with `GOPRECORDS_TOKEN_FILE`.

## Environment variables

| Variable | Default (root) | Default (non-root) |
|----------|----------------|--------------------|
| `GOPRECORDS_HOST` | **required** | **required** |
| `GOPRECORDS_TOKEN_FILE` | `/etc/goprecords-upload.token` | `$XDG_CONFIG_HOME/goprecords-upload-<HOST>/token` |
| `GOPRECORDS_BASE_URL` | `https://goprecords.f3s.buetow.org` | same |

## Step 1 — Issue a token on the server

`HOSTNAME` must match the short stats name used in every upload URL (e.g.
`f0`, `pi2`, `earth`, `blowfish`).

```bash
# daemon running in Kubernetes
kubectl exec -n services deployment/goprecords -- \
  goprecords --create-client-key HOSTNAME -stats-dir=/data/stats

# daemon running locally
goprecords --create-client-key HOSTNAME -stats-dir=/var/lib/goprecords/stats
```

The plaintext token is printed once. Store it only on the client.
Re-running `--create-client-key` for the same hostname replaces the previous token.

## Step 2 — Install the script

**FreeBSD**

```sh
doas install -m 755 scripts/goprecords-upload-client.sh \
  /usr/local/bin/goprecords-upload-client.sh
```

**Linux (root)**

```sh
sudo install -m 755 scripts/goprecords-upload-client.sh \
  /usr/local/bin/goprecords-upload-client.sh
```

**OpenBSD**

```sh
doas install -m 755 scripts/goprecords-upload-client.sh \
  /usr/local/bin/goprecords-upload-client.sh
```

**Linux (user session, e.g. earth)**

```sh
install -m 700 scripts/goprecords-upload-client.sh \
  ~/.local/bin/goprecords-upload-client.sh
```

**macOS (user session)**

The Darwin variant is a fish script and runs as a regular user (no root needed
for Homebrew-installed `uptimed`):

```sh
install -m 700 scripts/goprecords-upload-client-darwin.fish \
  ~/.local/bin/goprecords-upload-client-darwin.fish
```

Ensure `fish` is on `PATH` (it is when installed via Homebrew). The script
requires `curl`, `uptimed`, and `uprecords` — all provided by `brew install uptimed`.
Set `GOPRECORDS_RECORDS_FILE` only when your `uptimed` records file is outside
the standard Homebrew paths.

## Step 3 — Store the token

**FreeBSD / Linux (root)**

```sh
# FreeBSD
umask 077
echo 'TOKEN' | doas tee /etc/goprecords-upload.token
doas chmod 600 /etc/goprecords-upload.token

# Linux
umask 077
echo 'TOKEN' | sudo tee /etc/goprecords-upload.token
sudo chmod 600 /etc/goprecords-upload.token
```

**OpenBSD**

```sh
umask 077
echo 'TOKEN' | doas tee /etc/goprecords-upload.token
doas chmod 600 /etc/goprecords-upload.token
```

**Linux (user session)**

```sh
mkdir -p ~/.config/goprecords-upload-earth
umask 077
echo 'TOKEN' > ~/.config/goprecords-upload-earth/token
```

**macOS (user session)**

Replace `mymac` with the `GOPRECORDS_HOST` value you chose:

```sh
mkdir -p ~/.config/goprecords-upload-mymac
chmod 700 ~/.config/goprecords-upload-mymac
echo 'TOKEN' > ~/.config/goprecords-upload-mymac/token
chmod 600 ~/.config/goprecords-upload-mymac/token
```

## Step 4 — Automate

### FreeBSD — hourly cron

Add to root's crontab (`crontab -e` as root, or `/etc/crontab`). `curl` must
be on `PATH` for cron:

```cron
PATH=/bin:/sbin:/usr/bin:/usr/sbin:/usr/local/bin:/usr/local/sbin
0 * * * * root /usr/bin/env GOPRECORDS_HOST=f0 /usr/local/bin/goprecords-upload-client.sh
```

Repeat with the matching `GOPRECORDS_HOST` value on each host (`f1`, `f2`, `f3`, …).

### Linux (Rocky/Fedora) — hourly systemd timer, system-wide

`/etc/goprecords-upload.env` (mode `644`):

```ini
GOPRECORDS_HOST=pi0
```

`/etc/systemd/system/goprecords-upload.service`:

```ini
[Unit]
Description=Upload uptimed stats to goprecords

[Service]
Type=oneshot
EnvironmentFile=/etc/goprecords-upload.env
Environment=GOPRECORDS_TOKEN_FILE=/etc/goprecords-upload.token
ExecStart=/usr/local/bin/goprecords-upload-client.sh
```

`/etc/systemd/system/goprecords-upload.timer`:

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

```sh
sudo systemctl daemon-reload
sudo systemctl enable --now goprecords-upload.timer
```

Use a different `GOPRECORDS_HOST` in `/etc/goprecords-upload.env` on each Pi
(`pi1`, `pi2`, `pi3`).

### Linux — hourly systemd timer, user session (earth)

`~/.config/systemd/user/goprecords-upload-earth.service`:

```ini
[Unit]
Description=Upload uptimed stats to goprecords

[Service]
Type=oneshot
Environment=GOPRECORDS_HOST=earth
ExecStart=%h/.local/bin/goprecords-upload-client.sh
```

`~/.config/systemd/user/goprecords-upload-earth.timer`:

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

Enable lingering so uploads continue without an active login session:

```sh
sudo loginctl enable-linger "$USER"
systemctl --user daemon-reload
systemctl --user enable --now goprecords-upload-earth.timer
```

### OpenBSD — daily cron via /etc/daily.local

OpenBSD's `uptimed` only updates its records file once per day, so a daily run
is sufficient. Add to `/etc/daily.local`:

```sh
GOPRECORDS_HOST=blowfish /usr/local/bin/goprecords-upload-client.sh
```

Adjust `GOPRECORDS_HOST` for each OpenBSD host (`fishfinger`, etc.).

### macOS — hourly LaunchAgent

Create `~/Library/LaunchAgents/org.buetow.goprecords-upload.plist`.
Replace `mymac` with your `GOPRECORDS_HOST` value and adjust the path to
`fish` if your Homebrew prefix differs:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>org.buetow.goprecords-upload</string>
  <key>ProgramArguments</key>
  <array>
    <string>/opt/homebrew/bin/fish</string>
    <string>/Users/paul/.local/bin/goprecords-upload-client-darwin.fish</string>
  </array>
  <key>EnvironmentVariables</key>
  <dict>
    <key>GOPRECORDS_HOST</key>
    <string>mymac</string>
    <key>PATH</key>
    <string>/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin</string>
  </dict>
  <key>StartInterval</key>
  <integer>3600</integer>
  <key>StandardOutPath</key>
  <string>/tmp/goprecords-upload.log</string>
  <key>StandardErrorPath</key>
  <string>/tmp/goprecords-upload.err</string>
  <key>RunAtLoad</key>
  <true/>
</dict>
</plist>
```

Load it with:

```sh
launchctl load ~/Library/LaunchAgents/org.buetow.goprecords-upload.plist
```

For Intel Macs, replace `/opt/homebrew/bin/fish` with `/usr/local/bin/fish`.

### macOS — fish shell supersync integration

If you use the `supersync` fish function from the dotfiles, you can call the
upload script directly from your shell session by adding a call to your
`supersync` function or running it manually:

```fish
GOPRECORDS_HOST=mymac fish ~/.local/bin/goprecords-upload-client-darwin.fish
```

## Step 5 — Test one run

**FreeBSD**

```sh
doas env GOPRECORDS_HOST=f0 /usr/local/bin/goprecords-upload-client.sh
```

**Linux (root)**

```sh
sudo env GOPRECORDS_HOST=pi0 /usr/local/bin/goprecords-upload-client.sh
```

**OpenBSD**

```sh
doas env GOPRECORDS_HOST=blowfish /usr/local/bin/goprecords-upload-client.sh
```

**Linux (user session)**

```sh
GOPRECORDS_HOST=earth ~/.local/bin/goprecords-upload-client.sh
```

**macOS (user session)**

```sh
GOPRECORDS_HOST=mymac fish ~/.local/bin/goprecords-upload-client-darwin.fish
```
