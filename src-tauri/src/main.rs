// Prevents additional console window on Windows in release, DO NOT REMOVE!!
//#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

#[cfg(all(not(debug_assertions), windows))]
fn remove_windows_console() {
    unsafe {
        windows_sys::Win32::System::Console::FreeConsole();
    }
}

mod logic;

use std::process::Command;

use logic::discord;

#[link(name = "c")]
extern "C" {
    fn geteuid() -> u32;
}

// Learn more about Tauri commands at https://tauri.app/v1/guides/features/command
#[tauri::command]
fn find_discords() -> Vec<String> {
    discord::find_discords()
}

#[tauri::command]
fn install() -> String {
    let exe_path = std::env::var("APPIMAGE")
        .or_else(|_| std::env::current_exe().map(|p| p.to_string_lossy().to_string()))
        .unwrap();

    let mut cmd = Command::new("pkexec");
    cmd.arg("--disable-internal-agent")
        .arg(exe_path)
        .arg("install");

    dbg!(&cmd);
    let res = cmd.output().expect("failed to execute process");

    dbg!(&res);
    format!("Ran as UID {}", String::from_utf8(res.stdout).unwrap())
}

fn main() {
    if std::env::args_os().count() > 1 {
        cli();
        return;
    }

    tauri::Builder::default()
        .setup(|app| {
            start_gui(app.handle())?;

            Ok(())
        })
        .invoke_handler(tauri::generate_handler![find_discords, install])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}

fn start_gui(app: tauri::AppHandle) -> Result<(), tauri::Error> {
    #[cfg(all(not(debug_assertions), windows))]
    remove_windows_console();

    tauri::WindowBuilder::new(&app, "main", tauri::WindowUrl::App("index.html".into()))
        .title("Vencord Installer")
        .build()?;

    Ok(())
}

fn cli() {
    unsafe {
        println!("{}", geteuid());
    }
}
