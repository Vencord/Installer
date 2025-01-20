use crate::handler::handler::{error_with_code, ErrorCode, InstallerResult};
#[cfg(target_os = "windows")]
use crate::paths::locations::is_scuffed_install;
use crate::paths::shared::{resource_dir_path, use_appropriate_asar};
use crate::paths::branch::DiscordLocation;
use crate::update::download::download_file;

#[derive(Default)]
pub struct OpenAsarInstaller;
impl OpenAsarInstaller {

    pub fn new() -> Self {
        OpenAsarInstaller
    }

    // MARK: - Patch
    pub async fn patch(&self, discord_to_patch: DiscordLocation, patched_asar_file_url: &str, user_agent: &str) -> InstallerResult<()> {
        if discord_to_patch.openasar {
            return error_with_code("This location is already patched", ErrorCode::ErrAlreadyPatched);
        }

        #[cfg(target_os = "windows")]
        if is_scuffed_install(&discord_to_patch.name) {
            return error_with_code(&format!("This installation is scuffed: {:#?}", discord_to_patch.path), ErrorCode::ErrWindowsMovedDirectory);
        }

        let resource_dir = resource_dir_path(&discord_to_patch, discord_to_patch.is_system_electron);
        let asar_path = resource_dir.join(use_appropriate_asar(discord_to_patch.patched));
        let dl_tmp_asar_path = resource_dir.join("app.asar.tmp");

        download_file(
            patched_asar_file_url, 
            dl_tmp_asar_path.clone(), 
            user_agent
        ).await?;

        tokio::fs::rename(&asar_path, resource_dir.join("app.asar.backup")).await?;
        tokio::fs::rename(&dl_tmp_asar_path, &asar_path).await?;

        Ok(())
    }

    // MARK: - Unpatch
    pub async fn unpatch(&self, discord_to_patch: DiscordLocation) -> InstallerResult<()> {
        if !discord_to_patch.openasar {
            return error_with_code("This location is already unpatched", ErrorCode::ErrAlreadyUnpatched);
        }

        let resource_dir = resource_dir_path(&discord_to_patch, discord_to_patch.is_system_electron);
        let asar_path = resource_dir.join(use_appropriate_asar(discord_to_patch.patched));

        tokio::fs::remove_file(&asar_path).await?;

        let backup_paths = [
            resource_dir.join("app.asar.backup"),
            resource_dir.join("app.asar.original")
        ];

        match backup_paths.iter().find(|&path| path.exists()) {
            Some(backup) => tokio::fs::rename(backup, asar_path).await?,
            None => println!("No backup file found to restore?! How did we get here?"),
        }

        Ok(())
    }
}