use std::path::{Path, PathBuf};

use crate::paths::branch::{DiscordBranch, DiscordLocation};
use crate::paths::shared::{is_location_openasar, is_location_patched};

const DISCORD_LOCATIONS: [&str; 4] = [
    "Discord",
    "DiscordPTB",
    "DiscordCanary",
    "DiscordDevelopment",
];

/// Returns a list of available DiscordLocations on the machine.
pub fn get_discord_locations() -> Option<Vec<DiscordLocation>> {
    let appdata = std::env::var("LOCALAPPDATA").ok()?;
    let appdata_path = Path::new(&appdata);

    let locations: Vec<DiscordLocation> = DISCORD_LOCATIONS
        .iter()
        .filter_map(|&discord_location| {
            let full_path = appdata_path.join(discord_location);
            if full_path.exists() {
                parse_discord_location(&full_path)
            } else {
                None
            }
        })
        .collect();

    Some(locations).filter(|l| !l.is_empty())
}

fn parse_discord_location(full_path: &Path) -> Option<DiscordLocation> {
    let discord_location = full_path
        .file_name()
        .and_then(|n| n.to_str())?
        .to_string();

    // Windows my behated
    let app_path = std::fs::read_dir(full_path)
        .ok()?
        .flatten()
        .find_map(|entry| {
            let app_dir = full_path.join(entry.file_name());
            if app_dir.is_dir() && app_dir.join(get_discord_resource_location()).exists() {
                Some(app_dir)
            } else {
                None
            }
        })?;

    let patched = is_location_patched(&app_path, &false);

    Some(DiscordLocation {
        name: discord_location.to_string(),
        path: app_path.to_string_lossy().into_owned(),
        branch: DiscordBranch::from_path(&discord_location),
        patched,
        openasar: is_location_openasar(&app_path, patched),
        // we shouldn't care about these things on Windblows
        is_flatpak: false,
        is_system_electron: false,
    })
}

/// Returns and creates the data path for the given name.
/// 
/// # Arguments
/// 
/// * `data_dir` - The name of the data directory.
/// 
/// # Returns
/// 
/// Returns the path to the data directory.
pub fn get_data_path(data_dir: &str) -> PathBuf {
    let appdata = std::env::var("APPDATA").unwrap_or_default();

    let dir = Path::new(&appdata)
        .join(data_dir)
        .join("dist");

    std::fs::create_dir_all(&dir).ok();

    dir
}

/// Returns the path to the resources directory for Discord.
pub fn get_discord_resource_location() -> PathBuf {
    PathBuf::new()
        .join("resources")
}

/// Checks if the Discord installation is in a scuffed location.
/// 
/// See: https://github.com/Vencord/Installer/issues/9
/// 
/// # Arguments
/// 
/// * `name` - The name of the Discord installation.
pub fn is_scuffed_install(name: &String) -> bool {
    let username_dir = std::env::var("USERNAME").ok().unwrap_or_default();
    let program_data_dir = std::env::var("PROGRAMDATA").ok().unwrap_or_default();

    let scuffed_path = Path::new(&program_data_dir)
        .join(username_dir)
        .join(&name);

    scuffed_path.exists()
}

