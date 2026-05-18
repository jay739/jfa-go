use serde::{Deserialize, Serialize};
use std::fs;
use std::path::PathBuf;
use tauri::{AppHandle, Manager, Url, WebviewUrl, WebviewWindowBuilder};

#[derive(Serialize, Deserialize, Default, Clone)]
struct AppConfig {
    server_url: Option<String>,
}

fn config_file(app: &AppHandle) -> PathBuf {
    let dir = app
        .path()
        .app_config_dir()
        .expect("could not resolve app config dir");
    let _ = fs::create_dir_all(&dir);
    dir.join("config.json")
}

fn load_config(app: &AppHandle) -> AppConfig {
    fs::read_to_string(config_file(app))
        .ok()
        .and_then(|s| serde_json::from_str::<AppConfig>(&s).ok())
        .unwrap_or_default()
}

fn save_config(app: &AppHandle, cfg: &AppConfig) -> Result<(), String> {
    let json = serde_json::to_string_pretty(cfg).map_err(|e| e.to_string())?;
    fs::write(config_file(app), json).map_err(|e| e.to_string())
}

// Frontend → backend command: persist the chosen server URL.
#[tauri::command]
fn set_server_url(app: AppHandle, url: String) -> Result<(), String> {
    let trimmed = url.trim().trim_end_matches('/').to_string();
    if trimmed.is_empty() {
        return Err("URL cannot be empty".into());
    }
    // Validate parseable + http(s) before persisting and navigating.
    let parsed = Url::parse(&trimmed).map_err(|e| format!("invalid URL: {}", e))?;
    if !matches!(parsed.scheme(), "http" | "https") {
        return Err(format!("only http and https URLs are supported (got {})", parsed.scheme()));
    }
    save_config(
        &app,
        &AppConfig {
            server_url: Some(trimmed.clone()),
        },
    )?;
    // Navigate the main window to the freshly saved URL using Tauri's typed
    // navigate API — no string interpolation into JavaScript.
    if let Some(win) = app.get_webview_window("main") {
        win.navigate(parsed).map_err(|e| e.to_string())?;
    }
    Ok(())
}

// Frontend → backend command: read the saved URL on launch.
#[tauri::command]
fn get_server_url(app: AppHandle) -> Option<String> {
    load_config(&app).server_url
}

// Frontend → backend command: forget the saved URL (settings → "change server").
#[tauri::command]
fn clear_server_url(app: AppHandle) -> Result<(), String> {
    save_config(&app, &AppConfig::default())?;
    if let Some(win) = app.get_webview_window("main") {
        // Navigate back to the bundled setup page.
        if let Ok(setup) = Url::parse("tauri://localhost/index.html") {
            let _ = win.navigate(setup);
        }
    }
    Ok(())
}

pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .invoke_handler(tauri::generate_handler![
            set_server_url,
            get_server_url,
            clear_server_url
        ])
        .setup(|app| {
            let handle = app.handle().clone();
            let cfg = load_config(&handle);
            // If a server URL was previously saved and still parses, jump
            // straight to it. Otherwise (no URL, or stored URL is corrupted)
            // fall back to the bundled setup page rather than panicking.
            let target = cfg
                .server_url
                .as_deref()
                .and_then(|s| Url::parse(s).ok())
                .map(WebviewUrl::External)
                .unwrap_or_else(|| WebviewUrl::App("index.html".into()));
            let _win = WebviewWindowBuilder::new(app, "main", target)
                .title("Omnifin")
                .inner_size(1280.0, 800.0)
                .min_inner_size(720.0, 480.0)
                .center()
                .build()?;
            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
