use clap::{Args, CommandFactory};
use console::style;
use dialoguer::{Input, Select};

use vencord_installer_core::{
    download, Error,
    patch::patch_mod::Installer,
    paths::{get_data_path, get_discord_locations, DiscordLocation},
};

use crate::commands::Cli;

#[derive(Debug, Args, Clone)]
pub struct PatchArgs {
    #[arg(long, short)]
    install: bool,

    #[arg(long, short)]
    uninstall: bool,

    #[arg(long, short)]
    repair: bool,

    #[arg(long = "install-openasar", short = 'o')]
    install_openasar: bool,

    #[arg(long = "uninstall-openasar", short = 'x')]
    uninstall_openasar: bool,

    #[arg(long, short, value_name = "PATH")]
    location: Option<String>,
}

pub async fn execute(args: PatchArgs) -> Result<(), Error> {
    if args.install && args.uninstall {
        return Err(Error::ErrInvalidArguments(
            "You cannot use install and uninstall together.",
        ));
    }

    if args.repair && (args.install_openasar || args.uninstall_openasar) {
        return Err(Error::ErrInvalidArguments(
            "Repair cannot be used with OpenAsar actions.",
        ));
    }

    match (
        args.install,
        args.uninstall,
        args.repair,
        args.install_openasar,
        args.uninstall_openasar,
    ) {
        (true, _, _, openasar, _) => install(true, openasar, args.location).await?,
        (_, true, _, _, openasar) => uninstall(true, openasar, args.location).await?,
        (_, _, true, _, _) => repair(args.location).await?,
        _ => select_options(args).await?,
    }

    Ok(())
}

async fn get_location(path: Option<String>) -> Result<DiscordLocation, Error> {
    match path {
        Some(path) => DiscordLocation::from_path(&path)
            .ok_or(Error::ErrLocationInvalid),
        None => select_location().await,
    }
}

async fn maybe_download() -> Result<(), Error> {
    if std::env::var("VENCORD_DEV_INSTALL").map_or(true, |v| v != "1") {
        download().await?;
    }

    Ok(())
}

async fn install(
    vencord: bool,
    openasar: bool,
    path: Option<String>,
) -> Result<(), Error> {
    let location = get_location(path).await?;
    let mut installer = Installer::new(location.clone(), get_data_path())?;

    if vencord && !location.is_vencord {
        maybe_download().await?;
        installer.patch().await?;
    } else if openasar && !location.is_openasar {
        installer.patch_openasar().await?;
    } else {
        installer.repair().await?;

        if location.is_vencord {
            maybe_download().await?;
            installer.patch().await?;
        }

        if location.is_openasar {
            installer.patch_openasar().await?;
        }
    }

    Ok(())
}

async fn uninstall(
    vencord: bool,
    openasar: bool,
    path: Option<String>,
) -> Result<(), Error> {
    let location = get_location(path).await?;
    let mut installer = Installer::new(location.clone(), get_data_path())?;

    if vencord && location.is_vencord {
        installer.unpatch().await?;
    }

    if openasar && location.is_openasar {
        installer.unpatch_openasar().await?;
    }

    Ok(())
}

async fn repair(path: Option<String>) -> Result<(), Error> {
    let location = get_location(path).await?;

    maybe_download().await?;

    let mut installer = Installer::new(location.clone(), get_data_path())?;

    installer.repair().await?;

    if location.is_vencord {
        installer.patch().await?;
    }

    if location.is_openasar {
        installer.patch_openasar().await?;
    }

    Ok(())
}

async fn select_options(args: PatchArgs) -> Result<(), Error> {
    let options: [(&str, fn() -> std::pin::Pin<Box<dyn std::future::Future<Output = Result<(), Error>>>>); 5] = [
        ("Install Vencord", || Box::pin(install(true, false, None))),
        ("Uninstall Vencord", || Box::pin(uninstall(true, false, None))),
        ("Repair Vencord", || Box::pin(repair(None))),
        ("Install OpenAsar", || Box::pin(install(false, true, None))),
        ("Uninstall OpenAsar", || Box::pin(uninstall(false, true, None))),
    ];

    let mut items: Vec<_> = options.iter().map(|(name, _)| *name).collect();
    items.extend(["View Help Menu", "Exit"]);

    let choice = tokio::task::spawn_blocking(move || {
        Select::new()
            .with_prompt(
                style("Use ↑ ↓ and Enter to select an option")
                    .bold()
                    .to_string(),
            )
            .items(&items)
            .default(0)
            .interact()
    })
    .await?.unwrap();

    match choice {
        0..=4 => {
            options[choice].1().await?;
            Box::pin(select_options(args)).await?;
        }
        5 => {
            Cli::command().print_help()?;
            println!();
        }
        _ => {}
    }

    Ok(())
}

async fn select_location() -> Result<DiscordLocation, Error> {
    let locations = get_discord_locations();

    let mut items: Vec<_> = locations
        .iter()
        .map(|l| {
            let tags = [
                l.is_vencord.then_some("INSTALLED"),
                l.is_openasar.then_some("OPENASAR"),
            ]
            .into_iter()
            .flatten()
            .collect::<Vec<_>>()
            .join(", ");

            format!(
                "{}{} - {:?}",
                l.branch,
                if tags.is_empty() {
                    String::new()
                } else {
                    format!(" [{}]", tags)
                },
                l.path
            )
        })
        .collect();

    items.push("Custom Location".into());

    let idx = tokio::task::spawn_blocking(move || {
        Select::new()
            .with_prompt("Select a Discord location")
            .items(&items)
            .default(0)
            .interact()
    })
    .await?.unwrap();

    if idx == locations.len() {
        let path = tokio::task::spawn_blocking(|| {
            Input::<String>::new()
                .with_prompt("Enter custom Discord path")
                .interact_text()
        })
        .await?.unwrap();

        return DiscordLocation::from_path(&path)
            .ok_or(Error::ErrLocationInvalid);
    }

    Ok(locations[idx].clone())
}
