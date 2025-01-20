use std::path::{Path, PathBuf};

use crate::paths::branch::{DiscordBranch, DiscordLocation};
use crate::paths::shared::{is_location_flatpak, is_location_openasar, is_location_patched, is_location_system_electron};

const DISCORD_LOCATIONS: [&str; 15] = [
    "Discord",
    "DiscordPTB",
    "DiscordCanary",
    "DiscordDevelopment",
    "discord",
    "discordptb",
    "discordcanary",
    "discorddevelopment",
    "discord-ptb",
    "discord-canary",
    "discord-development",
    // Flatpak
    "com.discordapp.Discord",
    "com.discordapp.DiscordPTB",
    "com.discordapp.DiscordCanary",
    "com.discordapp.DiscordDevelopment",
];

/// Returns a list of available DiscordLocations on the machine.
pub fn get_discord_locations() -> Option<Vec<DiscordLocation>> {
    let home_dir = std::env::var("HOME").ok()?;
    let home_dir_path = Path::new(&home_dir);

    let paths = [
        Path::new("/usr/share"),
        Path::new("/usr/lib64"),
        Path::new("/opt"),
        &home_dir_path.join(".local/share"),
        &home_dir_path.join(".dvm"), // https://github.com/diced/dvm
        // flatpak
        Path::new("/var/lib/flatpak/app"),
        Path::new(".local/share/flatpak/app"),
    ];

    let locations: Vec<DiscordLocation> = paths
        .iter()
        .flat_map(|base| {
            DISCORD_LOCATIONS.iter().filter_map(|&discord_location| {
                let full_path = base.join(discord_location);
                if full_path.exists() {
                    Some(parse_discord_location(&full_path))
                } else {
                    None
                }
            })
        })
        .collect();

    Some(locations).filter(|l| !l.is_empty())
}

fn parse_discord_location(full_path: &PathBuf) -> DiscordLocation {
    let mut full_path = full_path.to_path_buf();
    
    let discord_location = full_path
        .file_name()
        .and_then(|n| n.to_str())
        .unwrap_or_default()
        .to_string();

    let is_flatpak = is_location_flatpak(&full_path);

    if is_flatpak {
        let name = full_path.file_name()
            .and_then(|n| n.to_str())
            .unwrap_or("")
            .strip_prefix("com.discordapp.")
            .unwrap_or("")
            .to_lowercase();

        let discord_name = if name != "discord" {
            format!("{}-{}", &name[..7], &name[7..])
        } else {
            name
        };

        full_path = full_path.join("current/active/files").join(&discord_name);
    }

    let system_electron = is_location_system_electron(&full_path);
    let patched = is_location_patched(&full_path, &system_electron);

    DiscordLocation {
        name: discord_location.to_string(),
        path: full_path.to_string_lossy().into_owned(),
        branch: DiscordBranch::from_path(&discord_location),
        patched,
        openasar: is_location_openasar(&full_path, patched),
        is_flatpak: is_flatpak,
        is_system_electron: system_electron,
    }
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
        .join(".config")
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