![Omnifin](images/banner.svg)

[![Docker Hub](https://img.shields.io/docker/pulls/jayakrishnakonda/omnifin?label=docker%20pulls)](https://hub.docker.com/r/jayakrishnakonda/omnifin)
[![GitHub release](https://img.shields.io/github/v/release/jay739/omnifin)](https://github.com/jay739/omnifin/releases)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Based on jfa-go](https://img.shields.io/badge/based%20on-hrfee%2Fjfa--go-6366f1)](https://github.com/hrfee/jfa-go)

> **Hard fork of [hrfee/jfa-go](https://github.com/hrfee/jfa-go) by Harvey Tindall.**
> The original design, architecture, and the vast majority of the codebase belong to Harvey Tindall and the jfa-go contributors. Omnifin exists to extend it with tighter Jellyfin integration, a redesigned UI/email system, and homelab-focused features. All original copyright is preserved — see [Credits](#credits--acknowledgements).

---

**Omnifin** is a unified Jellyfin user-management toolkit — invites, account lifecycle, password resets, multi-channel notifications, and deep Jellyfin integration, packaged into a single self-hosted binary.

##### Quick links: [Docker](#docker) · [Build from source](#build-from-source) · [Migrate from jfa-go](#migration-from-jfa-go) · [Credits](#credits--acknowledgements)

---

## What's new in Omnifin vs jfa-go

These features are exclusive to this fork and are not (yet) in upstream jfa-go.

### Jellyfin-powered announcement variables

Announcement emails and markdown templates can embed live Jellyfin library data using `{{double-brace}}` placeholders. On send, Omnifin queries Jellyfin and substitutes real values:

| Variable | Output |
|---|---|
| `{{recent_movies_grid}}` | 3-column poster grid of the 6 most recently added movies, each linked to the Jellyfin detail page |
| `{{recent_shows_grid}}` | Same for TV series |
| `{{featured_movie}}` | Full card — large poster, title, year, rating, genres, overview excerpt, "Watch Now" button |
| `{{featured_show}}` | Same for the most recently added show |
| `{{recent_movies}}` | Plain markdown bullet list (title + year + rating) |
| `{{recent_shows}}` | Same for shows |
| `{{recent_episodes}}` | Same for individual episodes |
| `{{top_genres}}` | Top 5 genres from recent movie additions |
| `{{longest_movie}}` | Title and runtime of the longest newly added movie |
| `{{user_count}}` | Total user count |
| `{{active_users_30d}}` | Users active in the last 30 days |
| `{{date}}` / `{{weekday}}` / `{{month_year}}` | Formatted date strings |
| `{{server_name}}` / `{{server_url}}` | Jellyfin server name and URL |

The **announcement preview** in the admin panel live-substitutes all variables — including poster images — so you see the final rendered result before sending.

### Redesigned email templates

All transactional emails (welcome, invite, password reset, expiry reminder, account expired, deletion, and more) have been rebuilt in MJML with:

- Per-email-type accent colour strips (green = welcome, amber = warning, red = critical, indigo = account update, blue = invite/confirmation)
- Eyebrow labels, info cards with left-border highlights, and dark-mode metadata
- "Open Jellyfin" CTA button in the welcome email
- Email-client-safe HTML with tested inline-style rendering

### Security hardening

- **Rate limiting** on all public-facing endpoints: login (10 req/min), invite signup (5 req/min), password reset (5 req/min)
- **Security response headers** on every response: `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Referrer-Policy`, `Permissions-Policy`, `Strict-Transport-Security` (when behind TLS proxy)
- **Swagger UI** moved behind authentication
- **XSS fix** in the invite form — username HTML-escaped before DOM insertion
- **Auth header panic fix** — malformed `Authorization` headers no longer crash the server
- **Backup upload cap** — 100 MB limit on restore file uploads

### UI / UX

- Dark design system with consistent color tokens
- Dashboard activity widget
- Announcement draft auto-saved to `localStorage`
- Drag-and-drop `.md` file loading into the announcement editor
- Variable chips in the announcement editor — click to insert `{{varname}}` at cursor
- Sidebar locked before login — cannot interact with navigation before authentication completes

---

## Core features (inherited from jfa-go)

- **Invites** — single- or multi-use signup links with per-invite expiry, library profiles, captcha, and email-the-invite
- **Account lifecycle** — per-user expiry with auto-disable / auto-delete; expiry reminder emails
- **Password resets** — email-based flow bridging Jellyfin's PIN system; self-service reset on the My Account page
- **My Account page** — users update password, email, notification contacts; optional referral invite links
- **Multi-channel messaging** — SMTP / Mailgun, Telegram, Discord, Matrix; per-event templates
- **Bulk admin tools** — enable/disable, delete, apply profiles, send Markdown announcements
- **Integrations** — Jellyseerr & Ombi provisioning, Discord role assignment, Telegram/Matrix contact linking
- **Backups** — scheduled BadgerDB snapshots with configurable retention
- **Activity log** — auditable record of all admin and lifecycle events

---

## Install

### Desktop client (Tauri)

A separate native desktop client is shipped alongside the server. It's **not** another server — it's a thin native window (built with Tauri + Rust) that points at an Omnifin server you've already deployed. Think of it like Discord Desktop or Notion Desktop: a native shell around the web UI.

On first launch it asks for your server URL (e.g. `https://omnifin.example.com` or `http://192.168.1.10:8056`), saves it locally, and from then on it opens straight to that server.

Downloads on every [GitHub Release](https://github.com/jay739/omnifin/releases):

- macOS: `Omnifin_<ver>_aarch64.dmg` (Apple Silicon) / `Omnifin_<ver>_x64.dmg` (Intel)
- Windows: `Omnifin_<ver>_x64-setup.exe` (NSIS installer) / `.msi`
- Linux: `omnifin-desktop_<ver>_amd64.deb` / `.AppImage`

Source lives in [`desktop/`](desktop/).

### Docker

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

Then open `http://localhost:8056` for the first-run setup wizard.

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
      - ./data:/data
      - ./config:/omnifin/config
      - /etc/localtime:/etc/localtime:ro
    healthcheck:
      test: ["CMD-SHELL", "curl -f http://localhost:8056 || exit 1"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
```

#### Volumes

| Container path | Purpose |
|---|---|
| `/data` | Database, generated config, backups. Must be persistent. |
| `/omnifin/config` | Optional: custom HTML/email templates, announcement template directory. |
| `/etc/localtime` (ro) | Keeps container timezone aligned with host. |

### Build from source

#### Prerequisites

- Go 1.24+
- Node.js 20+, npm
- `swag` — `go install github.com/swaggo/swag/cmd/swag@v1.16.4`
- `make`

#### Steps

```sh
git clone https://github.com/jay739/omnifin.git
cd omnifin
make all
# Binary: build/omnifin   Assets: build/data/
```

#### Build a local Docker image

```sh
make all
docker build -f Dockerfile.local -t omnifin:local .
```

---

## Migration from jfa-go

> The old `jayakrishnakonda/jfa-go:custom` image is **deprecated** and will receive no further updates. Migrate to `jayakrishnakonda/omnifin`.

**Change your image reference:**

```yaml
# Before
image: hrfee/jfa-go:latest
# or
image: jayakrishnakonda/jfa-go:custom

# After
image: jayakrishnakonda/omnifin:latest
```

**Everything else carries over unchanged:**

- Existing `/data` directory — same BadgerDB schema
- Existing `config.ini` — `jfa-go:` template prefix and `/jfa-go/config/...` paths are still resolved
- Existing backups (`jfa-go-db-*.bak`) — restorable via `-restore`
- All user accounts, invites, profiles, custom emails, announcement templates

The legacy `/jfa-go/config` bind-mount target is still recognised, so existing Compose files keep working without changes.

---

## Environment variables

| Variable | Purpose |
|---|---|
| `OMNIFIN_CONFIG_HOST` | Map container-style `/omnifin/config/...` paths to a host directory when running the binary directly. |
| `JFA_GO_CONFIG_HOST` | Legacy alias for `OMNIFIN_CONFIG_HOST`. |
| `PUID` / `PGID` | UID/GID the process runs as (Docker). |
| `TZ` | Timezone, e.g. `America/New_York` (Docker). |

---

## Usage

```
omnifin [command] [flags]

Commands:
  start       Start as a daemon
  stop        Stop a daemonised instance
  systemd     Write an omnifin.service unit file to the current directory

Flags (excerpt):
  -config, -c   Path to config file
  -data, -d     Path to data directory
  -debug        Enable debug logging and pprof
  -host         Listen address override
  -port, -p     Listen port override
  -restore      Path to a .bak file to restore from
  -swagger      Expose Swagger UI (requires authentication)
  -help, -h     Print help
```

---

## Roadmap

- Jellystat integration — per-user watch time column on the accounts page
- Bulk announcement filters — target by "expiring in N days" or "inactive N days"
- Telegram broadcast alongside email announcements
- Activity-based expiry auto-extension
- Authentik OIDC login for the admin panel
- Jellyseerr request-approval webhook notifications

---

## Contributing

Pull requests are welcome. For substantial changes, open an issue first.

For bugs or features that are not specific to Omnifin's divergence from upstream, consider opening them at [hrfee/jfa-go](https://github.com/hrfee/jfa-go) where they benefit a wider audience.

---

## Credits & Acknowledgements

### Original author — Harvey Tindall (hrfee)

**[jfa-go](https://github.com/hrfee/jfa-go)** was created and is maintained by **[Harvey Tindall](https://github.com/hrfee)**. The core architecture, user management engine, multi-channel notification system, invite system, MJML pipeline, and the vast majority of the Go and TypeScript code in this repository are Harvey's work.

Omnifin is a personal fork. If you want a mature, well-supported tool with a wider community, **use upstream jfa-go**. The original MIT license and copyright notice are preserved verbatim in [LICENSE](LICENSE).

All jfa-go contributors: [hrfee/jfa-go/graphs/contributors](https://github.com/hrfee/jfa-go/graphs/contributors)

### Omnifin maintainer

Jayakrishna Konda — [github.com/jay739](https://github.com/jay739)

### Stack this builds on

- **[Jellyfin](https://github.com/jellyfin/jellyfin)** — open-source media server
- **[gin-gonic/gin](https://github.com/gin-gonic/gin)** — HTTP framework
- **[MJML](https://mjml.io/)** — responsive email framework
- **[BadgerDB](https://github.com/dgraph-io/badger)** — embedded key-value store
- Jellyseerr, Ombi, Authentik, Tailscale, and the broader self-hosting community

---

## License

MIT.

Both the original jfa-go license (Copyright 2025 Harvey Tindall) and the Omnifin additions (Copyright 2026 Jayakrishna Konda) are preserved in [LICENSE](LICENSE).
