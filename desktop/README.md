# Omnifin Desktop

A native desktop client for a self-hosted Omnifin server.

This is **not** a local Omnifin server — it's a thin native window (Tauri / Rust) that points at an Omnifin server you've deployed elsewhere (your homelab, a VPS, etc). Think of it like Discord Desktop is to Discord servers, or Notion Desktop is to notion.so.

## First launch

When you open the app for the first time, it asks for your Omnifin server URL (e.g. `https://omnifin.example.com` or `http://192.168.1.10:8056`). That URL is saved to:

- macOS: `~/Library/Application Support/dev.jay739.omnifin/config.json`
- Linux: `~/.config/dev.jay739.omnifin/config.json`
- Windows: `%APPDATA%\dev.jay739.omnifin\config.json`

On subsequent launches the app opens straight to the saved URL.

## Changing the server

Delete the config file above, or implement the `clear_server_url` Tauri command from a menu item (see `src-tauri/src/lib.rs`).

## Building from source

Requires:
- Rust toolchain (stable)
- Node.js 20+
- Platform-specific:
  - **Linux**: `libwebkit2gtk-4.1-dev libgtk-3-dev libayatana-appindicator3-dev librsvg2-dev`
  - **macOS**: Xcode Command Line Tools
  - **Windows**: WebView2 (ships with Windows 11), MSVC build tools

```sh
cd desktop
npm install
npm run build
```

Output appears in `src-tauri/target/release/bundle/`:
- `dmg/Omnifin_*.dmg` (macOS)
- `deb/omnifin-desktop_*.deb` + `appimage/omnifin-desktop_*.AppImage` (Linux)
- `msi/Omnifin_*.msi` + `nsis/Omnifin_*-setup.exe` (Windows)

## Project layout

```
desktop/
├── src-tauri/        Rust backend
│   ├── src/
│   │   ├── main.rs    Windows entry point (hides console)
│   │   └── lib.rs     App logic, config persistence, IPC commands
│   ├── icons/         App icons (reused from web favicon)
│   ├── capabilities/  Tauri 2.x permission model
│   ├── Cargo.toml
│   └── tauri.conf.json
├── dist/              Frontend bundle
│   └── index.html     First-run server URL prompt (only HTML in the app)
└── package.json       For the Tauri CLI (no other JS dependencies)
```
