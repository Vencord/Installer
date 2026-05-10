#[cfg(target_os = "macos")]
mod _mac;

#[cfg(target_os = "macos")]
pub use _mac::*;

#[cfg(target_os = "windows")]
mod _windows;

#[cfg(target_os = "windows")]
pub use _windows::*;

#[cfg(target_os = "linux")]
mod _linux;

#[cfg(target_os = "linux")]
pub use _linux::*;

mod branch;
mod location;

pub use branch::DiscordBranch;
pub use location::DiscordLocation;

pub fn get_data_path() -> Option<std::path::PathBuf> {
    let path = if let Ok(path) = std::env::var("VENCORD_USER_DATA_DIR") {
        Some(std::path::PathBuf::from(path))
    } else {
        get_data_path_impl()
    }?
    .join("dist");

    std::fs::create_dir_all(&path).ok();

    Some(path)
}
