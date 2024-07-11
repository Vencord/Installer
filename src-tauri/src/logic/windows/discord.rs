use std::path::PathBuf;

use crate::logic::shared::{DiscordBranch, DiscordInstall};

const DISCORD_NAMES: &[&str] = &[
    "Discord",
    "DiscordPTB",
    "DiscordCanary",
    "DiscordDevelopment",
];

pub fn find_discords() -> Vec<DiscordInstall> {
    let dir = dirs::data_local_dir().expect("Unable to find AppData/Local");

    let mut discords: Vec<DiscordInstall> = vec![];

    let Ok(children) = dir.read_dir() else {
        return discords;
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

    discords
}

pub fn parse_discord(path: PathBuf) -> Option<DiscordInstall> {
    let path_str = path.to_str()?.to_owned();
    let base = path.file_name()?.to_str()?.to_owned();

    let mut app_path_str = "".to_owned();
    let mut is_patched: bool = false;

    let Ok(children) = path.read_dir() else {
        return None;
    };

    for child_result in children {
        let Ok(child) = child_result else { continue };

        if !child.file_type().map(|t| t.is_dir()).unwrap_or(false) {
            continue;
        }

        if child
            .file_name()
            .to_str()
            .map(|n| n.starts_with("app-"))
            .unwrap_or(false)
        {
            let app_path = child.path();
            let resources = app_path.join("resources");

            if !resources.exists() {
                continue;
            };

            let Some(app_name) = app_path.file_name().and_then(|n| n.to_str()) else {
                continue;
            };

            if app_name > app_path_str.as_str() {
                if let Some(path_str) = app_path.to_str() {
                    app_path_str = path_str.to_owned();
                    is_patched = resources.join("_app.asar").exists()
                }
            }
        }
    }

    Some(DiscordInstall {
        path: path_str,
        app_path: Some(app_path_str),
        branch: DiscordBranch::from_filename(&base),
        is_patched,
        is_flatpak: false,
    })
}
