use std::env;
use std::path::{Path, PathBuf};

use crate::paths::DiscordLocation;

const KNOWN_NAMES: [&str; 4] = [
    "Discord",
    "DiscordPTB",
    "DiscordCanary",
    "DiscordDevelopment",
];

/// Returns a list of available DiscordLocations on the machine.
pub fn get_discord_locations() -> Vec<DiscordLocation> {
    let Ok(appdata) = std::env::var("LOCALAPPDATA") else {
        return Vec::new();
    };

    let appdata_path = Path::new(&appdata);

    let mut locations = Vec::new();

    for base in KNOWN_NAMES {
        let root = appdata_path.join(base);

        if let Some(loc) = parse_discord_location(&root) {
            locations.push(loc);
        }
    }

    locations
}

fn parse_discord_location(full_path: &Path) -> Option<DiscordLocation> {
    let best = std::fs::read_dir(full_path)
        .ok()?
        .flatten()
        .filter_map(|entry| {
            let path = entry.path();
            let name = path.file_name()?.to_str()?;
            let ver = name.strip_prefix("app-")?;

            Some((ver.to_string(), path))
        })
        .max_by(|(a, _), (b, _)| a.cmp(b))
        .map(|(_, path)| path)?;

    DiscordLocation::from_path(best)
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
pub fn get_data_path_impl() -> Option<PathBuf> {
    let Ok(appdata) = std::env::var("APPDATA") else {
        return None;
    };

    let dir = Path::new(&appdata).join("Vencord");

    std::fs::create_dir_all(&dir).ok();

    Some(dir.clone())
}

pub fn get_program_data_path() -> Option<PathBuf> {
    let Some(program_data) = env::var_os("PROGRAMDATA") else {
        return None;
    };

    let Some(username) = env::var_os("USERNAME") else {
        return None;
    };

    Some(Path::new(&program_data).join(username))
}
