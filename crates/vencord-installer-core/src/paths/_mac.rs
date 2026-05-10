use std::env;
use std::path::{Path, PathBuf};

use crate::paths::DiscordLocation;

/// Returns a list of available DiscordLocations on the machine.
pub fn get_discord_locations() -> Vec<DiscordLocation> {
    let Some(home_dir) = env::home_dir() else {
        return Vec::new();
    };

    const KNOWN_NAMES: [&str; 4] = [
        "Discord.app",
        "Discord Canary.app",
        "Discord PTB.app",
        "Discord Development.app",
    ];

    let search_paths = [
        Path::new("/Applications"),
        &home_dir.join("Applications"),
        // Some users have Discord on their desktop?
        // &home_dir.join("Desktop"),
    ];

    let mut locations = Vec::new();

    // <search_paths>/<KNOWN_NAMES>
    for base in &search_paths {
        for location in KNOWN_NAMES {
            let full_path = base.join(location);

            if let Some(discord_location) = DiscordLocation::from_path(&full_path) {
                locations.push(discord_location);
            }
        }
    }

    locations
}

/// Returns the path to the data directory.
pub(crate) fn get_data_path_impl() -> Option<PathBuf> {
    let Some(home_dir) = env::home_dir() else {
        return None;
    };

    let dir = &home_dir
        .join("Library")
        .join("Application Support")
        .join("Vencord");

    Some(dir.clone())
}
