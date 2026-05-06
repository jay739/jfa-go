![Omnifin](images/banner.svg)

# Omnifin

A unified Jellyfin user-management toolkit: invites, account lifecycle, password resets, multi-channel notifications, and integrations with the surrounding stack (Jellyseerr, Ombi, Telegram, Discord, Matrix, Authentik).

> **Hard fork of [hrfee/jfa-go](https://github.com/hrfee/jfa-go).**
> All credit for the original design and the vast majority of the code goes to Harvey Tindall and the jfa-go contributors. Omnifin diverges to focus on tighter integration with a self-hosted "all of Jellyfin in one place" stack and may follow different release timing and feature priorities.

---

## Status

Active. Built and used in a personal homelab; expect rough edges around features specific to that environment.

Currently tracks Jellyfin 10.11.x compatibility inherited from upstream.

---

## Features (inherited from jfa-go)

- **Invites** — single/multi-use links, per-invite expiry, library access via Jellyfin profiles, captcha, email-the-invite.
- **Account lifecycle** — set per-user expiry, auto-disable or auto-delete on expiry, send reminders before expiry, referrals.
- **Multi-channel messaging** — email (SMTP/Mailgun), Telegram bot, Discord bot, Matrix bot. Per-event templates (welcome, expiry, password reset, announcement).
- **Self-service portal** at `/my/account` — password reset, contact info update, expiry visibility.
- **Password reset bridge** — watches Jellyfin's PIN-based reset flow and emails the PIN to the user, so vanilla Jellyfin's offline reset becomes a normal "Forgot Password" experience.
- **Integrations** — Jellyseerr, Ombi, Discord roles, Telegram contact linking.
- **Customizable** — Markdown-supported templates for everything user-facing.

## What's different in Omnifin (and where it's headed)

- Module path: `github.com/jay739/omnifin`
- Binary name: `omnifin`
- Container layout: `/opt/omnifin/omnifin`, `/omnifin/config` mount target (legacy `/jfa-go/config` paths still resolved for in-place upgrades)
- Template namespace: `omnifin:` prefix (legacy `jfa-go:` still resolved)
- Backup file prefix: `omnifin-db-…` (legacy `jfa-go-db-…` files still readable)
- Updater: points at `jay739/omnifin` releases (none yet — harmlessly 404s)

Planned features (subject to change):
- Markdown-folder announcement templates (drop a `.md` file, get a template)
- Jellystat per-user "last watched" column on the accounts page
- Bulk-announcement filters (expiring soon, dormant)
- Storage-per-user metric pulled from Jellyfin
- Telegram broadcast on the announcement modal
- Activity-based expiry auto-extend
- Jellyseerr request-approval webhook hook

## Install

A Dockerfile (`Dockerfile.local`) is provided that packages the output of `make all` into a Debian-slim image with `curl` for compose healthchecks.

```sh
make all
docker build -f Dockerfile.local -t omnifin:local .
```

```sh
docker run -d \
  --name omnifin \
  -p 8056:8056 \
  -v /path/to/omnifin/data:/data \
  -v /path/to/omnifin/config:/omnifin/config \
  -v /etc/localtime:/etc/localtime:ro \
  omnifin:local
```

There are no published binaries or Docker Hub images yet.

## Build from source

Requires Go 1.24+, Node.js 20+, npm, `swag` (`go install github.com/swaggo/swag/cmd/swag@v1.16.4`), and `upx`.

```sh
make all
```

Output: `build/omnifin` plus `build/data/`.

## Usage

```
Usage of omnifin:
  start              start omnifin as a daemon and run in the background.
  stop               stop a daemonized instance of omnifin.
  systemd            generate a systemd .service file.

Flags (excerpt):
  -config, -c        alternate path to config file
  -data, -d          alternate path to data directory
  -debug             enable debug logging
  -host              alternate listen address
  -port, -p          alternate listen port
  -restore           path to database backup to restore
  -swagger           expose swagger UI at /swagger/index.html
```

A setup wizard runs on first start at `http://localhost:8056`.

## Systemd

`omnifin systemd` generates `omnifin.service` in the working directory. Move it to `~/.config/systemd/user/` (user) or `/etc/systemd/system/` (system) and `systemctl daemon-reload`.

## Contributing

This is currently a single-maintainer fork. Issues and PRs are welcome but the cadence will be slow. For upstream features and bug reports unrelated to Omnifin's divergence, prefer [hrfee/jfa-go](https://github.com/hrfee/jfa-go).

## License

MIT, inherited from jfa-go. See [LICENSE](LICENSE).

## Acknowledgements

- [hrfee/jfa-go](https://github.com/hrfee/jfa-go) — Harvey Tindall and contributors.
- The Jellyfin, Jellyseerr, and broader self-hosting community.
