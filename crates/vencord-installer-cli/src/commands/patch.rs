use clap::Args;
use console::style;
use dialoguer::Select;

use vencord_installer_core::{
    Error, OPENASAR_URL, download, get_dist_path,
    patch::patch_mod::Installer,
    paths::{
        branch::DiscordLocation, locations::get_discord_locations,
        shared::get_custom_discord_location,
    },
};

#[derive(Debug, Args)]
pub struct PatchArgs {
    /// Install Vencord client mod
    #[arg(long, short)]
    pub install: bool,
    /// Uninstall Vencord client mod
    #[arg(long, short)]
    pub uninstall: bool,
    /// Repair Vencord client mod
    #[arg(long, short)]
    pub repair: bool,
    /// Install OpenAsar
    #[arg(long = "install-openasar", short = 'o')]
    pub install_openasar: bool,
    /// Uninstall OpenAsar
    #[arg(long = "uninstall-openasar", short = 'x')]
    pub uninstall_openasar: bool,
    /// Custom Discord location
    #[arg(long, short, value_name = "PATH")]
    pub custom: Option<String>,
}

pub async fn execute(args: PatchArgs) -> Result<(), Error> {
    if args.install && args.uninstall {
        return Err(Error::ErrInvalidArguments(
            "You cannot use install and uninstall commands together.",
        ));
    }

    if args.custom.is_some() && !(args.install_openasar || args.uninstall_openasar) {
        return Err(Error::ErrInvalidArguments(
            "You must specify an install or uninstall when using --custom!",
        ));
    }

    if args.repair && (args.install_openasar || args.uninstall_openasar) {
        return Err(Error::ErrInvalidArguments(
            "Repair cannot be used with OpenAsar install/uninstall commands.",
        ));
    }

    if args.install || args.install_openasar {
        install(args.install, args.install_openasar, args.custom).await?;
    } else if args.uninstall || args.uninstall_openasar {
        uninstall(args.uninstall, args.uninstall_openasar, args.custom).await?;
    } else if args.repair {
        repair(args.custom).await?;
    } else {
        select_options().await?;
    }

    Ok(())
}

async fn install(
    client_mod: bool,
    openasar: bool,
    custom_path: Option<String>,
) -> Result<(), Error> {
    let selected_location: DiscordLocation;

    if let Some(path) = custom_path {
        selected_location = match get_custom_discord_location(&path) {
            Some(location) => location,
            _ => return Err(Error::ErrLocationInvalid),
        };
    } else {
        selected_location = select_location().await?;
    }

    let mut installer = Installer::new(selected_location.clone(), Some(get_dist_path(None)));

    if client_mod && !selected_location.patched {
        if std::env::var("VENCORD_DEV_INSTALL").map_or(true, |v| v != "1") {
            download().await?;
        }

        installer.patch().await?;
    } else if openasar && !selected_location.openasar {
        installer.patch_openasar(OPENASAR_URL).await?;
    } else {
        log::info!("Already installed, skipping!");
    }

    Ok(())
}

async fn uninstall(
    client_mod: bool,
    openasar: bool,
    custom_path: Option<String>,
) -> Result<(), Error> {
    let selected_location: DiscordLocation;

    if let Some(path) = custom_path {
        selected_location = match get_custom_discord_location(&path) {
            Some(location) => location,
            _ => return Err(Error::ErrLocationInvalid),
        };
    } else {
        selected_location = select_location().await?;
    }

    let mut installer = Installer::new(selected_location.clone(), None);

    if client_mod && selected_location.patched {
        installer.unpatch().await?;
    } else if openasar && selected_location.openasar {
        installer.unpatch_openasar().await?;
    } else {
        log::info!("Not installed, skipping!");
    }

    Ok(())
}

async fn repair(custom_path: Option<String>) -> Result<(), Error> {
    let selected_location: DiscordLocation;

    if let Some(path) = custom_path {
        selected_location = match get_custom_discord_location(&path) {
            Some(location) => location,
            _ => return Err(Error::ErrLocationInvalid),
        };
    } else {
        selected_location = select_location().await?;
    }

    if std::env::var("VENCORD_DEV_INSTALL").map_or(true, |v| v != "1") {
        download().await?;
    }

    let mut installer = Installer::new(selected_location.clone(), Some(get_dist_path(None)));

    if selected_location.patched {
        installer.patch().await?;
    }
    if selected_location.openasar {
        installer.patch_openasar(OPENASAR_URL).await?;
    }

    Ok(())
}

pub async fn select_options() -> Result<(), Error> {
    let options = [
        "Install Vencord",
        "Uninstall Vencord",
        "Repair Vencord",
        "Install OpenAsar",
        "Uninstall OpenAsar",
        "Exit",
    ];

    let selection = tokio::task::spawn_blocking(move || {
        Select::new()
            .with_prompt(
                style("Use ↑ ↓ and Enter to select an option")
                    .bold()
                    .to_string(),
            )
            .items(&options)
            .default(0)
            .interact()
    })
    .await;

    let Ok(Ok(choice)) = selection else {
        return Err(Error::ErrOther("Failed to read selection"));
    };

    match choice {
        0 => {
            install(true, false, None).await?;
            Box::pin(select_options()).await?;
        }
        1 => {
            uninstall(true, false, None).await?;
            Box::pin(select_options()).await?;
        }
        2 => {
            repair(None).await?;
            Box::pin(select_options()).await?;
        }
        3 => {
            install(false, true, None).await?;
            Box::pin(select_options()).await?;
        }
        4 => {
            uninstall(false, true, None).await?;
            Box::pin(select_options()).await?;
        }
        _ => {}
    }

    Ok(())
}

async fn select_location() -> Result<DiscordLocation, Error> {
    let locations = get_discord_locations().ok_or(Error::ErrLocationInvalid)?;

    let items: Vec<String> = locations
        .iter()
        .map(|location| {
            let mut instance = Vec::new();
            instance.push(location.branch.to_string());
            if location.is_flatpak {
                instance.push("Flatpak".to_owned());
            }

            let mut tags = Vec::new();
            if location.patched {
                tags.push("[INSTALLED]");
            }
            if location.openasar {
                tags.push("[OPENASAR]");
            }

            let tags_str = if tags.is_empty() {
                String::new()
            } else {
                format!(" {}", tags.join(" "))
            };

            format!(
                "{}{} – {}",
                instance.join(", "),
                tags_str,
                location.path.to_string()
            )
        })
        .collect();

    let selection = tokio::task::spawn_blocking(move || {
        Select::new()
            .with_prompt(
                style("Use ↑ ↓ and Enter to select a Discord location")
                    .bold()
                    .to_string(),
            )
            .items(&items)
            .default(0)
            .interact()
    })
    .await;

    match selection {
        Ok(Ok(idx)) => Ok(locations[idx].clone()),
        _ => Err(Error::ErrLocationInvalid),
    }
}
