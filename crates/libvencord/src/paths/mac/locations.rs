use std::path::{Path, PathBuf};

use crate::paths::branch::{DiscordBranch, DiscordLocation};
use crate::paths::shared::{is_location_openasar, is_location_patched};

const DISCORD_LOCATIONS: [&str; 4] = [
    "Discord.app",
    "Discord Canary.app",
    "Discord PTB.app",
    "Discord Development.app",
];

/// Returns a list of available DiscordLocations on the machine.
pub fn get_discord_locations() -> Option<Vec<DiscordLocation>> {
    let home_dir = std::env::var("HOME").ok()?;
    let home_dir_path = Path::new(&home_dir);

    let paths = [
        // TODO: add ~/Desktop because people are dumb
        Path::new("/Applications"),
        &home_dir_path.join("Applications"),
    ];

    let locations: Vec<DiscordLocation> = paths
        .iter()
        .flat_map(|base| {
            DISCORD_LOCATIONS.iter().filter_map(|&discord_location| {
                let full_path = base.join(discord_location);
                full_path.exists().then(|| parse_discord_location(&full_path))
            })
        })
        .flatten()
        .collect();

    Some(locations).filter(|l| !l.is_empty())
}

fn parse_discord_location(full_path: &Path) -> Option<DiscordLocation> {
    if !full_path.join(get_discord_resource_location()).exists() {
        return None;
    }

    let discord_location = full_path
        .file_name()
        .and_then(|n| n.to_str())
        .unwrap_or_default();
    
    let patched = is_location_patched(full_path, &false);

    Some(DiscordLocation {
        name: discord_location.to_string(),
        path: full_path.to_string_lossy().into_owned(),
        branch: DiscordBranch::from_path(discord_location),
        patched,
        openasar: is_location_openasar(full_path, patched),
        // we shouldn't care about these things on macOS
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
    let home_dir = std::env::var("HOME").unwrap_or_default();
    let dir = Path::new(&home_dir)
        .join("Library")
        .join("Application Support")
        .join(data_dir)
        .join("dist");

    std::fs::create_dir_all(&dir).ok();

    dir
}

/// Returns the path to the resources directory for Discord.
pub fn get_discord_resource_location() -> PathBuf {
    PathBuf::new()
        .join("Contents")
        .join("Resources")
}