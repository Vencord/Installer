use std::path::{Path, PathBuf};

use crate::paths::branch::{DiscordBranch, DiscordLocation};
use crate::paths::shared::{
    is_location_flatpak, is_location_openasar, is_location_patched, is_location_system_electron,
};

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
    let home_dir = find_home()?;
    let home_dir_path = Path::new(&home_dir);

    let paths = [
        Path::new("/usr/share"),
        Path::new("/usr/lib64"),
        Path::new("/opt"),
        &home_dir_path.join(".local/share"),
        &home_dir_path.join(".dvm"), // https://github.com/diced/dvm
        // flatpak
        Path::new("/var/lib/flatpak/app"),
        &home_dir_path.join(".local/share/flatpak/app"),
        // new locations
        &home_dir_path.join(".config"),
    ];

    let locations: Vec<DiscordLocation> = paths
        .iter()
        .flat_map(|base| {
            DISCORD_LOCATIONS.iter().filter_map(|&discord_location| {
                let full_path = base.join(discord_location);
                if full_path.exists() {
                    parse_discord_location(&full_path)
                } else {
                    None
                }
            })
        })
        .collect();

    Some(locations).filter(|l| !l.is_empty())
}

fn parse_discord_location(full_path: &PathBuf) -> Option<DiscordLocation> {
    let mut full_path = full_path.to_path_buf();

    let discord_location = full_path
        .file_name()
        .and_then(|n| n.to_str())
        .unwrap_or_default()
        .to_string();

    let is_flatpak = is_location_flatpak(&full_path);

    if is_flatpak {
        let name = full_path
            .file_name()
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

    let resources = get_discord_resource_location();

    // For installs that use versioned app-* subdirectories (e.g. ~/.config/discordcanary),
    // find the latest one if resources aren't directly present.
    if !full_path.join(&resources).exists() {
        let app_path = std::fs::read_dir(&full_path)
            .ok()?
            .flatten()
            .filter_map(|entry| {
                let app_dir = full_path.join(entry.file_name());
                if !app_dir.is_dir() || !app_dir.join(&resources).join("app.asar").exists() {
                    return None;
                }

                let dir_name = app_dir.file_name()?.to_str()?.to_string();
                let version = dir_name
                    .strip_prefix("app-")?
                    .split('.')
                    .map(|part| part.parse::<u64>().ok())
                    .collect::<Option<Vec<_>>>()?;

                Some((version, app_dir))
            })
            .max_by(|(a_version, _), (b_version, _)| a_version.cmp(b_version))
            .map(|(_, path)| path)?;

        full_path = app_path;
    }

    if !full_path.join(&resources).join("app.asar").exists() {
        return None;
    }

    let system_electron = is_location_system_electron(&full_path);
    let patched = is_location_patched(&full_path, &system_electron);

    Some(DiscordLocation {
        name: discord_location.to_string(),
        path: full_path.to_string_lossy().into_owned(),
        branch: DiscordBranch::from_path(&discord_location),
        patched,
        openasar: is_location_openasar(&full_path, patched),
        is_flatpak: is_flatpak,
        is_system_electron: system_electron,
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
    let home_dir = find_home().unwrap_or_default();

    let dir = Path::new(&home_dir)
        .join(".config")
        .join(data_dir)
        .join("dist");

    std::fs::create_dir_all(&dir).ok();

    dir
}

/// Returns the path to the resources directory for Discord.
pub fn get_discord_resource_location() -> PathBuf {
    PathBuf::new().join("resources")
}

fn get_user_info(username: &str) -> Option<(String, u32, u32)> {
    std::fs::read_to_string("/etc/passwd")
        .ok()
        .and_then(|contents| {
            contents
                .lines()
                .find(|line| line.starts_with(&format!("{}:", username)))
                .and_then(|line| {
                    let parts: Vec<&str> = line.split(':').collect();
                    if parts.len() >= 6 {
                        let home = parts[5].to_string();
                        let uid = parts[2].parse::<u32>().ok()?;
                        let gid = parts[3].parse::<u32>().ok()?;
                        Some((home, uid, gid))
                    } else {
                        None
                    }
                })
        })
}

fn find_home() -> Option<String> {
    std::env::var("SUDO_USER")
        .or_else(|_| std::env::var("DOAS_USER"))
        .ok()
        .and_then(|user| get_user_info(&user).map(|(home, _, _)| home))
        .or_else(|| std::env::var("HOME").ok())
        .filter(|h| !h.is_empty())
}

pub async fn copy_ownership_permissions(to: &PathBuf) -> Result<(), crate::Error> {
    use std::os::unix::fs::chown;

    let (uid, gid) = std::env::var("SUDO_USER")
        .or_else(|_| std::env::var("DOAS_USER"))
        .ok()
        .and_then(|user| get_user_info(&user).map(|(_, uid, gid)| (uid, gid)))
        .ok_or(crate::Error::ErrPermissionDenied)?;

    let to = to.clone();
    tokio::task::spawn_blocking(move || chown(to, Some(uid), Some(gid)))
        .await?
        .ok();

    Ok(())
}
