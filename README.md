![Omnifin](images/banner.svg)

[![Docker Hub](https://img.shields.io/docker/pulls/jayakrishnakonda/omnifin?label=docker)](https://hub.docker.com/r/jayakrishnakonda/omnifin)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

##### Downloads:
##### [docker](#docker) | [build from source](#build-from-source)

---

## About

**Omnifin** is a unified Jellyfin user-management toolkit. It bundles invites, account lifecycle, password resets, multi-channel notifications, and tight integration with the surrounding self-hosted stack (Jellyseerr, Ombi, Telegram, Discord, Matrix, Authentik) into a single binary.

> **Hard fork of [hrfee/jfa-go](https://github.com/hrfee/jfa-go).**
> All credit for the original design and the vast majority of the code belongs to Harvey Tindall and the jfa-go contributors. Omnifin diverges to focus on tighter integration with a "single pane of glass for Jellyfin" stack and may follow a different release cadence and feature roadmap.

---

## Project Status: Active

Built and used in production in the maintainer's homelab. Expect the cadence to vary; this is a single-maintainer project. Issues and pull requests are welcome.

#### Compatibility

Targets Jellyfin 10.11.x (inherited from jfa-go upstream). Compatibility with future Jellyfin releases will be maintained on a best-effort basis.

#### Alternatives

- [Wizarr](https://github.com/Wizarrrr/wizarr) — simpler invite-focused tool, supports Jellyfin/Plex/Emby, Discord-based invites.
- [hrfee/jfa-go](https://github.com/hrfee/jfa-go) — the upstream this is forked from. If you want a more conservative, generic-purpose tool with a wider user base, prefer upstream.

---

## Features

- **Invites** — single- or multi-use signup links with per-invite expiry, library access via Jellyfin profiles, captcha support, and email-the-invite-directly.
- **Account lifecycle** — set per-user expiry windows. On expiry: auto-disable, auto-delete, or disable-then-delete after N days. Send reminders before expiry.
- **Password resets** — bridges Jellyfin's PIN-based reset so users get a normal "forgot password" email flow. Also offers a self-service reset button on the My Account page.
- **My Account page** — users update their own password, email, or contact channels. Optional referral system: users can invite friends with a limited-use link.
- **Multi-channel messaging** — email (SMTP / Mailgun), Telegram bot, Discord bot, Matrix bot. Each lifecycle event (welcome, expiry, deletion, password reset, custom announcement) has its own template.
- **Bulk admin tools** — manage all users from one page: enable/disable, delete, send Markdown announcements, apply profile/library settings.
- **Integrations** — Jellyseerr & Ombi account provisioning, Discord role assignment, Telegram contact linking.
- **Customizable** — full Markdown support in user-facing messages, invite forms, and email templates.
- **Backups** — built-in scheduled BadgerDB snapshots with retention policy.
- **Activity log** — auditable record of admin actions and user lifecycle events.

## Planned features

These are the next features on the roadmap. Subject to change.

- **Markdown-folder announcement templates** — drop a `.md` file into a watched folder and have it auto-loaded as an announcement template.
- **Jellystat integration** — show per-user "last watched" and total watch time as a column on the accounts page.
- **Bulk announcement filters** — target announcements by criteria like "users expiring in N days" or "users inactive for N days."
- **Storage-per-user metric** — pull Jellyfin library size attributable to each user.
- **Telegram broadcast button** — send announcements via Telegram for users who linked it, alongside (or instead of) email.
- **Activity-based expiry auto-extend** — bump expiry when a user has watched something recently.
- **Jellyseerr request-approval webhook** — fire a templated message to the user when their request is approved.
- **Authentik SSO option** — log into the admin panel via your existing OIDC provider.

---

## Install

### Docker

A prebuilt image is published on Docker Hub at [`jayakrishnakonda/omnifin`](https://hub.docker.com/r/jayakrishnakonda/omnifin).

```sh
docker run -d \
  --name omnifin \
  -p 8056:8056 \
  -e PUID=1000 \
  -e PGID=1000 \
  -e TZ=Etc/UTC \
  -v /path/to/omnifin/data:/data \
  -v /path/to/omnifin/config:/omnifin/config \
  -v /etc/localtime:/etc/localtime:ro \
  --restart unless-stopped \
  jayakrishnakonda/omnifin:latest
```

Then open `http://localhost:8056` for the setup wizard.

#### Docker Compose

```yaml
services:
  omnifin:
    image: jayakrishnakonda/omnifin:latest
    container_name: omnifin
    restart: unless-stopped
    environment:
      - PUID=1000
      - PGID=1000
      - TZ=Etc/UTC
    ports:
      - "8056:8056"
    volumes:
      - ./omnifin/data:/data
      - ./omnifin/config:/omnifin/config
      - /etc/localtime:/etc/localtime:ro
    healthcheck:
      test: ["CMD-SHELL", "curl -f http://localhost:8056 || exit 1"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
```

#### Volumes

| Mount point inside the container | Purpose |
|---|---|
| `/data` | Database, generated config, backups. Persistent. |
| `/omnifin/config` | Optional: custom HTML templates, custom email templates, announcement template directory. |
| `/etc/localtime` (ro) | Keeps the container's timezone aligned with the host. |

> **Migrating from jfa-go?** Omnifin transparently reads the legacy `/jfa-go/config` mount path, the `jfa-go:` template namespace prefix, the `JFA_GO_CONFIG_HOST` environment variable, and `jfa-go-db-*.bak` backup files. You can rename in-place at your own pace.

### Build from source

#### Prerequisites

- Go 1.24 or later
- Node.js 20 or later, with `npm`
- `swag` (`go install github.com/swaggo/swag/cmd/swag@v1.16.4`)
- `upx` (binary compression, optional but used by default)
- Standard build tools (`make`)

#### Build

```sh
git clone https://github.com/jay739/omnifin.git
cd omnifin
make all
```

Outputs `build/omnifin` (the binary) and `build/data/` (web assets, default config, language files, license, systemd unit).

#### Build and package as a Docker image

```sh
make all
docker build -f Dockerfile.local -t omnifin:local .
```

`Dockerfile.local` packages the freshly built binary and data tree into a `debian:bookworm-slim` image with `curl` for compose healthchecks.

---

## Usage

```
Usage of omnifin:
  start              start omnifin as a daemon and run in the background.
  stop               stop a daemonized instance of omnifin.
  systemd            generate an omnifin.service file in the working directory.

Flags (excerpt):
  -config, -c        path to config file (default: $XDG_CONFIG_HOME/omnifin/config.ini)
  -data, -d          path to data directory  (default: $XDG_CONFIG_HOME/omnifin)
  -debug             enable debug logging and expose pprof at /debug/pprof
  -host              listen address override
  -port, -p          listen port override
  -restore           path to a database backup .bak file to restore from
  -swagger           expose Swagger UI at /swagger/index.html
  -help, -h          print this help
```

A first-run setup wizard binds to `0.0.0.0:8056` until the configuration is saved. After that, the regular admin panel takes over.

---

## Systemd

Omnifin does not run as a daemon by default. To install as a per-user systemd service:

```sh
omnifin systemd                           # writes ./omnifin.service in the cwd
mv omnifin.service ~/.config/systemd/user/
systemctl --user daemon-reload
systemctl --user enable --now omnifin
```

For a system-wide install, place the generated unit in `/etc/systemd/system/` and use plain `systemctl` (without `--user`).

---

## Environment variables

| Variable | Purpose |
|---|---|
| `OMNIFIN_CONFIG_HOST` | When the binary runs on the host but config still uses container-style `/omnifin/config/...` paths, set this to the host directory those paths should map to. |
| `JFA_GO_CONFIG_HOST` | Legacy alias for `OMNIFIN_CONFIG_HOST`; honored if the new variable is unset. |
| `PUID` / `PGID` (Docker) | UID/GID the process runs as. |
| `TZ` (Docker) | Timezone (e.g. `America/New_York`). |

---

## Migration notes (from jfa-go)

Omnifin is drop-in compatible with an existing jfa-go installation, with two operational changes:

1. **Image / binary name** — `jayakrishnakonda/omnifin:latest` instead of `hrfee/jfa-go:latest`.
2. **Container internal paths** — the conventional bind-mount target is `/omnifin/config` (the legacy `/jfa-go/config` mount target is still recognised, so existing compose files keep working).

The following continue to work unchanged:

- Existing `data/` directories — same BadgerDB schema, just point Omnifin at them.
- Existing `config.ini` files — `jfa-go:` template prefix and `/jfa-go/config/...` paths are still resolved.
- Existing `data/backups/jfa-go-db-*.bak` files — restorable via `-restore`.
- All existing user accounts, invites, profiles, custom emails, announcement templates.

When you next save your config from the admin UI, paths will be rewritten to the `omnifin:` prefix automatically (provided the in-app save runs through the normal config-write path).

---

## Contributing

Pull requests are welcome. For substantial changes, please open an issue first to discuss the direction.

For features and bug reports that aren't specific to Omnifin's divergence from upstream, consider opening them at [hrfee/jfa-go](https://github.com/hrfee/jfa-go) where they are likely to benefit a wider audience.

---

## License

MIT. Both the original [jfa-go](https://github.com/hrfee/jfa-go) license (Copyright 2025 Harvey Tindall) and the Omnifin license (Copyright 2026 Jayakrishna Konda) are preserved verbatim in [LICENSE](LICENSE).

---

## Acknowledgements

- **[hrfee/jfa-go](https://github.com/hrfee/jfa-go)** — Harvey Tindall and contributors. Omnifin would not exist without their work.
- **[Jellyfin](https://github.com/jellyfin/jellyfin)** — the media server this entire ecosystem is built around.
- The wider self-hosting community that maintains the surrounding stack: Jellyseerr, Ombi, Authentik, Tailscale, and many others.
