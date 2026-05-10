use std::{
    fs,
    path::{Path, PathBuf},
};

use super::DiscordLocation;

const KNOWN_NAMES: [&str; 15] = [
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
pub fn get_discord_locations() -> Vec<DiscordLocation> {
    let Some(home) = find_home() else {
        return Vec::new();
    };

    let home = PathBuf::from(home);

    let search_paths = [
        PathBuf::from("/usr/share"),
        PathBuf::from("/usr/lib64"),
        PathBuf::from("/opt"),
        home.join(".local/share"),
        home.join(".dvm"),
        home.join(".config"),
        PathBuf::from("/var/lib/flatpak/app"),
        home.join(".var/app"),
        home.join(".local/share/flatpak/app"),
    ];

    search_paths
        .into_iter()
        .flat_map(|base| {
            KNOWN_NAMES.iter().filter_map(move |name| {
                let path = base.join(name);
                path.exists()
                    .then(|| parse_discord_location(path))
                    .flatten()
            })
        })
        .collect()
}

fn parse_discord_location(mut path: PathBuf) -> Option<DiscordLocation> {
    let path_str = path.to_string_lossy();

    let is_system_flatpak = path_str.contains("/var/lib/flatpak/");
    let is_user_flatpak = path_str.contains("/.var/app/");

    if is_user_flatpak {
        path = path.join("config").join("discord");
    }

    if is_system_flatpak {
        let flatpak_name = path.file_name()?.to_str()?;
        let name = flatpak_name.strip_prefix("com.discordapp.")?;

        let discord_dir = if name == "discord" {
            "discord".to_string()
        } else {
            format!("{}-{}", &name[..7], &name[7..])
        };

        path = path
            .join("current")
            .join("active")
            .join("files")
            .join(discord_dir);
    }

    let has_asar = path.join("resources").join("app.asar").exists();

    if !has_asar {
        path = fs::read_dir(&path)
            .ok()?
            .flatten()
            .filter_map(|entry| {
                let app_dir = entry.path();

                let version = app_dir
                    .file_name()?
                    .to_str()?
                    .strip_prefix("app-")?
                    .split('.')
                    .map(|p| p.parse::<u64>().ok())
                    .collect::<Option<Vec<_>>>()?;

                Some((version, app_dir))
            })
            .max_by(|a, b| a.0.cmp(&b.0))
            .map(|(_, dir)| dir)?;
    }

    let location = DiscordLocation::from_path(path)?;

    Some(location)
}

/// Returns and creates the data path for the given name.
pub(crate) fn get_data_path_impl() -> Option<PathBuf> {
    let home_dir = find_home()?;

    let dir = Path::new(&home_dir).join(".config").join("Vencord");

    Some(dir)
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
