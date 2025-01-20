#[cfg(feature = "generate_asar")]
use std::collections::HashMap;
#[cfg(feature = "generate_asar")]
use std::fs::File;
#[cfg(feature = "generate_asar")]
use std::io::Write;
#[cfg(target_os = "linux")]
use std::{env, process::Command};
#[cfg(feature = "generate_asar")]
use serde::Serialize;

use crate::handler::handler::{error_with_code, ErrorCode, InstallerResult};
#[cfg(target_os = "linux")]
use crate::paths::locations::get_data_path;
#[cfg(target_os = "windows")]
use crate::paths::locations::is_scuffed_install;
use crate::paths::shared::resource_dir_path;
use crate::paths::branch::DiscordLocation;

#[cfg(target_os = "linux")]
extern "C" {
    fn geteuid() -> u32;
}

#[cfg(feature = "generate_asar")]
#[derive(Serialize)]
struct AsarEntry {
    size: i32,
    offset: String,
}

#[derive(Default)]
pub struct Installer;
impl Installer {
    
    pub fn new() -> Self { 
        Installer
    }

    // MARK: - Patch
    pub async fn patch(&self, discord_to_patch: DiscordLocation, patched_asar_file: &str) -> InstallerResult<()> {
        if discord_to_patch.patched {
            return error_with_code("This location is already patched", ErrorCode::ErrAlreadyPatched);
        }

        #[cfg(target_os = "windows")]
        if is_scuffed_install(&discord_to_patch.name) {
            return error_with_code("This installation is scuffed", ErrorCode::ErrWindowsMovedDirectory);
        }

        let resource_dir = resource_dir_path(&discord_to_patch, discord_to_patch.is_system_electron);
        let asar_path = resource_dir.join("app.asar");
        let _asar_path = resource_dir.join("_app.asar");

        tokio::fs::rename(&asar_path, &_asar_path).await?;
        tokio::fs::rename(patched_asar_file, &asar_path).await?;

        #[cfg(target_os = "linux")]
        if discord_to_patch.is_system_electron {
            let asar_path = resource_dir.join("app.asar.unpacked");
            let _asar_path = resource_dir.join("_app.asar.unpacked");

            tokio::fs::rename(&asar_path, &_asar_path).await?;
        }

        Ok(())
    }

    // MARK: - Unpatch
    pub async fn unpatch(&self, discord_to_patch: DiscordLocation) -> InstallerResult<()> {
        if !discord_to_patch.patched {
            return error_with_code("This location is already unpatched", ErrorCode::ErrAlreadyUnpatched);
        }

        let resource_dir = resource_dir_path(&discord_to_patch, discord_to_patch.is_system_electron);
        let asar_path = resource_dir.join("app.asar");
        let _asar_path = resource_dir.join("_app.asar");

        tokio::fs::remove_file(&asar_path).await?;
        tokio::fs::rename(&_asar_path, &asar_path).await?;

        #[cfg(target_os = "linux")]
        if discord_to_patch.is_system_electron {
            let asar_path = resource_dir.join("app.asar.unpacked");
            let _asar_path = resource_dir.join("_app.asar.unpacked");

            tokio::fs::rename(&_asar_path, &asar_path).await?;
        }

        Ok(())
    }

    #[cfg(feature = "generate_asar")]
    pub async fn write_app_asar(&self, out_file: &str, patcher_path: &str) -> InstallerResult<()> {
        let index_js = format!("require({})", serde_json::to_string(&patcher_path)?);
        let pkg_json = "{ \"name\": \"discord\", \"main\": \"index.js\" }";
        
        let mut files = HashMap::new();
        
        files.insert("index.js".to_string(), AsarEntry {
            size: index_js.len() as i32,
            offset: "0".to_string(),
        });

        files.insert("package.json".to_string(), AsarEntry {
            size: pkg_json.len() as i32,
            offset: index_js.len().to_string(),
        });

        let header = serde_json::to_string(&HashMap::from([("files".to_string(), files)]))?;
        let aligned_size = (header.len() as u32 + 3) & !3;
        
        let mut file = File::create(out_file)?;
        
        [4u32, aligned_size + 8, aligned_size + 4, header.len() as u32]
            .iter()
            .try_for_each(|&size| file.write_all(&(size as i32).to_le_bytes()))?;

        file.write_all(format!("{:<width$}", header, width = aligned_size as usize).as_bytes())?;
        file.write_all(index_js.as_bytes())?;
        file.write_all(pkg_json.as_bytes())?;

        Ok(())
    }
    
    #[cfg(target_os = "linux")]
    pub fn grant_flatpak_permissions(&self, discord_install: &DiscordLocation, files_dir: &str) -> InstallerResult<()> {
        let name = discord_install.path
            .split('/')
            .find(|s| s.starts_with("com.discordapp."))
            .unwrap_or("");

        let is_system_flatpak = discord_install.path.contains("/var");

        let mut args = vec![];

        if !is_system_flatpak {
            args.push("--user");
        }
        args.push("override");
        args.push(name);
        let filesystem_arg = format!("--filesystem={}", &files_dir);
        args.push(&filesystem_arg);
        let full_cmd = format!("flatpak {}", args.join(" "));

        if !is_system_flatpak && unsafe { geteuid() } == 0 {
            Command::new("sudo")
                .arg("-u")
                .arg(env::var("SUDO_USER").unwrap())
                .arg("sh")
                .arg("-c")
                .arg(&full_cmd)
                .output()
                .map_err(|e| e.to_string())?;
        } else {
            Command::new("sh")
                .arg("-c")
                .arg(&full_cmd)
                .output()
                .map_err(|e| e.to_string())?;
        };

        Ok(())
    }

    #[cfg(target_os = "linux")]
    pub fn fix_permissions(&self) {

    }
}