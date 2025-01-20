use std::path::{Path, PathBuf};
use std::io::{Seek, BufReader, Read};

use crate::paths::branch::{DiscordBranch, DiscordLocation};

use super::locations::get_discord_resource_location;

/// Returns a DiscordLocation for the Discord installation at the given path.
/// 
/// # Arguments
/// 
/// * `path` - The path to the Discord installation.
/// 
/// # Returns
/// 
/// Returns a DiscordLocation if the path is a "valid" Discord installation, otherwise None.
pub fn get_custom_discord_location(full_path: String) -> Option<DiscordLocation> {
    let path = Path::new(&full_path);
    let last_path_component = get_last_path_component(path).unwrap_or_default();

    let is_flatpak = is_location_flatpak(path);
    let system_electron = is_location_system_electron(path);

    if !system_electron && !path.join(get_discord_resource_location()).exists() {
        return None;
    }

    let patched = is_location_patched(path, &system_electron);

    Some(DiscordLocation {
        name: last_path_component.to_string(),
        path: full_path.clone(),
        branch: DiscordBranch::from_path(last_path_component),
        patched,
        openasar: is_location_openasar(path, patched),
        is_flatpak,
        is_system_electron: system_electron,
    })
}

fn get_last_path_component(path: &Path) -> Option<&str> {
    path.file_name().and_then(|name| name.to_str())
}

/// Returns the path to the resources directory of the Discord installation.
/// 
/// # Arguments
/// 
/// * `discord_to_patch` - The DiscordLocation to get the resources directory from.
/// * `_system_electron` - Whether the Discord installation is from https://aur.archlinux.org/packages/discord_arch_electron.
pub fn resource_dir_path(discord_to_patch: &DiscordLocation, _system_electron: bool) -> PathBuf {
    let base_path = Path::new(&discord_to_patch.path);
    // compiler optimization.. sorry nerds
    #[cfg(target_os = "linux")]
    return if _system_electron {
        base_path.to_path_buf()
    } else {
        base_path.join(get_discord_resource_location())
    };

    #[cfg(not(target_os = "linux"))]
    base_path.join(get_discord_resource_location())
}

/// This is for OpenAsar, since depending on if its patched, it has a different named asar file.
pub fn use_appropriate_asar(patched: bool) -> PathBuf {
    if patched {
        PathBuf::from("_app.asar")
    } else {
        PathBuf::from("app.asar")
    }
}

/// Returns whether the Discord installation at the given path has been patched by the mod.
/// 
/// # Arguments
/// 
/// * `path` - The path to the Discord installation.
/// * `system_electron` - Whether the Discord installation is from https://aur.archlinux.org/packages/discord_arch_electron.
pub fn is_location_patched(path: &Path, system_electron: &bool) -> bool {
    let mut asar_path = path.to_path_buf();

    if !system_electron {
        asar_path = asar_path.join(get_discord_resource_location()).join("_app.asar");
    } else {
        asar_path = asar_path.join("_app.asar.unpacked");
    }

    asar_path.exists()
}

/// Returns whether the Discord installation at the given path has been patched by OpenAsar.
/// 
/// # Arguments
/// 
/// * `path` - The path to the Discord installation.
/// * `patched` - Whether the Discord installation has been patched by the mod already.
pub fn is_location_openasar(path: &Path, patched: bool) -> bool {
    let asar_path = path
        .join(get_discord_resource_location())
        .join(use_appropriate_asar(patched));

    let Ok(mut file) = std::fs::File::open(&asar_path) else {
        return false;
    };
    
    // 0x12FA
    if file.seek(std::io::SeekFrom::Start(4858)).is_err() {
        return false;
    }

    let mut buffer = [0; 1024];
    let Ok(n) = BufReader::new(file).read(&mut buffer) else {
        return false;
    };

    buffer[..n].windows(8).any(|window| window == b"OpenAsar")
}

/// Returns whether the Discord installation at the given path is a Flatpak installation.
/// 
/// # Arguments
/// 
/// * `path` - The path to the Discord installation.
pub fn is_location_flatpak(_path: &Path) -> bool {
     // compiler optimization.. sorry nerds
    #[cfg(target_os = "linux")]
    if _path.to_string_lossy().contains("/flatpak/") {
        true
    } else {
        false
    }
    
    #[cfg(not(target_os = "linux"))]
    false
}

/// Returns whether the Discord installation at the given path is from https://aur.archlinux.org/packages/discord_arch_electron.
/// 
/// # Arguments
/// 
/// * `path` - The path to the Discord installation.
pub fn is_location_system_electron(_path: &Path) -> bool {
     // compiler optimization.. sorry nerds
    #[cfg(target_os = "linux")]
    if !_path.join(get_discord_resource_location()).exists() {
        true
    } else {
        false
    }

    #[cfg(not(target_os = "linux"))]
    false
}