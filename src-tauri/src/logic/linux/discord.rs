use std::path::PathBuf;

use crate::logic::shared::{DiscordBranch, DiscordInstall};

const DISCORD_NAMES: &[&str] = &[
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

pub fn find_discords() -> Vec<DiscordInstall> {
    let home = dirs::home_dir().expect("HOME not set");

    let discord_dirs = [
        PathBuf::from("/usr/share"),
        "/usr/local/share".into(),
        "/usr/lib64".into(),
        "/opt".into(),
        home.join(".local/share"),
        "/var/lib/flatpak/app".into(),
        home.join(".local/share/flatpak/app"),
        home.join(".dvm"), // https://github.com/diced/dvm
    ];

    let mut discords: Vec<DiscordInstall> = vec![];

    for dir in discord_dirs.iter() {
        let Ok(children) = dir.read_dir() else {
            continue;
        };

        for child_result in children {
            let Ok(child) = child_result else { continue };

            if !child.file_type().map(|t| t.is_dir()).unwrap_or(false) {
                continue;
            }

            if !child
                .file_name()
                .to_str()
                .map(|s| DISCORD_NAMES.contains(&s))
                .unwrap_or(false)
            {
                continue;
            }

            if let Some(discord) = parse_discord(child.path()) {
                discords.push(discord);
            }
        }
    }

    discords
}

pub fn parse_discord(mut path: PathBuf) -> Option<DiscordInstall> {
    let path_str = path.to_str()?.to_owned();
    let base = path.file_name()?.to_str()?.to_owned();

    let is_flatpak = path_str.contains("/flatpak/");
    if is_flatpak && !path_str.contains("/current/active/files") {
        const REVERSE_DOMAIN_PREFIX_LENGTH: usize = "com.discordapp.".len();
        let mut discord_name = base[REVERSE_DOMAIN_PREFIX_LENGTH..].to_lowercase();
        if discord_name != "discord" {
            discord_name = format!("{}-{}", &discord_name[..7], &discord_name[7..]);
        }
        path.push("current/active/files");
        path.push(discord_name);
    }

    let resources = path.join("resources");

    if !resources.exists() {
        return None;
    }

    Some(DiscordInstall {
        path: path_str,
        app_path: None,
        branch: DiscordBranch::from_filename(&base),
        is_patched: resources.join("_app.asar").exists(),
        is_flatpak,
    })
}
