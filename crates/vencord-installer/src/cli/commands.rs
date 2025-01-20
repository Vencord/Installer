use std::path::PathBuf;
use std::env;

use libvencord::handler::handler::{error_with_code, ErrorCode, InstallerResult};
use libvencord::paths::{branch::DiscordLocation, locations::get_data_path, shared::get_custom_discord_location};
use libvencord::patch::{patch_mod::Installer, patch_openasar::OpenAsarInstaller};
use libvencord::update::{download::prepare_dist_directory, version_check::{check_hash_from_release, check_local_version}};

use logger_rs::{info, success, warn};
use tokio::runtime::Runtime;

use crate::{OPENASAR_URL, RELEASE_TAG_DOWNLOAD, RELEASE_URL, RELEASE_URL_FALLBACK, USER_AGENT};

use super::select_location::select_location;

// MARK: - Install

pub fn install(client_mod: bool, openasar: bool, custom_path: Option<String>) -> InstallerResult<()> {
    let mut selected_location: DiscordLocation;

    let rt = Runtime::new().unwrap();
    
    if let Some(path) = custom_path {
        selected_location = match get_custom_discord_location(path) {
            Some(location) => location,
            None => return error_with_code("This location seems invalid, is it actually a Discord location?", ErrorCode::ErrUnknown)
        };
    } else {
        selected_location = select_location("Please select a Discord location by entering the corresponding number:");
    }

    info!("You selected {:?}, attempting to patch!", selected_location.path);
    info!("Using this path for dist: {}", get_dist_path().display());

    if client_mod && !selected_location.patched {
        // user may forget to set this variable..
        if env::var("VENCORD_DEV_INSTALL").map_or(true, |v| v != "1") {
            download_assets()?;
        }


        let installer = Installer::new();

        rt.block_on(installer.write_app_asar(
            &get_dist_path().join("app.asar").to_string_lossy(), 
            &get_dist_path().join("patcher.js").to_string_lossy()
        ))?;

        rt.block_on(installer.patch(
            selected_location.clone(), 
            &get_dist_path().join("app.asar").to_string_lossy()
        ))?;

        #[cfg(target_os = "linux")]
        if selected_location.is_flatpak {
            installer.grant_flatpak_permissions(selected_location.clone(), &get_dist_path().to_string_lossy())?;
        }

        selected_location.patched = true;

        success!("Successfully patched Discord!");
    } else if client_mod {
        warn!("Discord is already patched with Vencord, skipping!");
    }

    if openasar && !selected_location.openasar {
        let installer = OpenAsarInstaller::new();

        rt.block_on(installer.patch(
            selected_location.clone(), 
            OPENASAR_URL,
            USER_AGENT
        ))?;

        success!("Successfully patched Discord with OpenAsar!");
    } else if openasar {
        warn!("Discord is already patched with OpenAsar, skipping!");
    }

    info!("Done!");

    Ok(())
}

// MARK: - Uninstall

pub fn uninstall(client_mod: bool, openasar: bool, custom_path: Option<String>) -> InstallerResult<()> {
    let mut selected_location: DiscordLocation;

    let rt = Runtime::new().unwrap();

    if let Some(path) = custom_path {
        selected_location = match get_custom_discord_location(path) {
            Some(location) => location,
            None => return error_with_code("This location seems invalid, is it actually a Discord location?", ErrorCode::ErrUnknown)
        };
    } else {
        selected_location = select_location("Please select a patched Discord location by entering the corresponding number:");
    }

    info!("You selected {:?}, attempting to unpatch...", selected_location.path);

    if client_mod && selected_location.patched {
        let installer = Installer::new();
        rt.block_on(installer.unpatch(
            selected_location.clone()
        ))?;

        selected_location.patched = false;

        success!("Successfully unpatched Discord!");
    } else if client_mod {
        warn!("Discord is not patched with Vencord, skipping!");
    }

    if openasar && selected_location.openasar  {
        let installer = OpenAsarInstaller::new();
        rt.block_on(installer.unpatch(
            selected_location.clone()
        ))?;

        success!("Successfully unpatched Discord with OpenAsar!");
    } else if openasar {
        warn!("Discord is not patched with OpenAsar, skipping!");
    }

    info!("Done!");

    Ok(())
}

// MARK: - Download

fn download_assets() -> InstallerResult<()> {
    let rt = Runtime::new().unwrap();

    info!("Checking for dist files to download...");

    let latest_version = rt.block_on(check_hash_from_release(RELEASE_URL, Some(RELEASE_URL_FALLBACK), USER_AGENT));
    let local_version = rt.block_on(check_local_version(&get_dist_path(), r"// Vencord ([0-9a-zA-Z\.-]+)"));

    info!("Latest version: {}", latest_version.clone().unwrap_or_default());
    info!("Local version: {}", local_version.clone().unwrap_or_default());

    if latest_version.is_some() && latest_version != local_version {
        info!("Downloading dist files...");

        rt.block_on(prepare_dist_directory(
            &get_dist_path(),
            RELEASE_TAG_DOWNLOAD,
            USER_AGENT,
            [
                "patcher.js".to_string(),
                "preload.js".to_string(),
                "renderer.js".to_string(),
                "renderer.css".to_string(),
            ],
        ))?;
    } else {
        info!("Nothing new to download, skipping!");
    }

    Ok(())
}

// MARK: - Paths

fn get_dist_path() -> PathBuf {
    if let Ok(path) = env::var("VENCORD_USER_DATA_DIR") {
        PathBuf::from(path).join("dist")
    } else {
        get_data_path("Vencord")
    }
}
