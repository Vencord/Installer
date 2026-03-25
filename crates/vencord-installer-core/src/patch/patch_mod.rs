use std::path::PathBuf;

#[cfg(feature = "generate_asar")]
use crate::asar::write_app_asar;
#[cfg(target_os = "windows")]
use crate::paths::locations::is_scuffed_install;
use crate::paths::shared::{is_asar_considered_openasar, resource_dir_path, use_appropriate_asar};
use crate::paths::{branch::DiscordLocation, shared::is_asar_original};
use crate::update::download::download_file;
use crate::{Error, patch::FileOperation};

// TODO: integrate repair stuff to standard functioality

pub struct Installer {
    discord_location: DiscordLocation,
    data_path: Option<PathBuf>,
}

impl Installer {
    pub fn new(discord_location: DiscordLocation, data_path: Option<PathBuf>) -> Self {
        Installer {
            discord_location,
            data_path,
        }
    }

    // MARK: - Patch
    pub async fn patch(&mut self) -> Result<(), Error> {
        if self.discord_location.patched {
            log::error!("This Discord install is already patched, nothing to do.");
            return Err(Error::ErrLocationPatched);
        }

        let data_path = &self.data_path.clone().ok_or(Error::ErrNoDataPath)?;

        #[cfg(target_os = "windows")]
        if is_scuffed_install(&self.discord_location.name) {
            log::error!("You have a broken Discord install. Please reinstall Discord!");
            return Err(Error::ErrWindowsMovedDirectory);
        }

        let custom_asar_path = data_path.join("app.asar");

        #[cfg(feature = "generate_asar")]
        {
            use serde_json::json;

            let entries = [
                (
                    "index.js",
                    &format!(
                        "require(\"{}\")",
                        &data_path.join("patcher.js").to_string_lossy()
                    ),
                ),
                (
                    "package.json",
                    &json!({
                        "name": "discord",
                        "main": "index.js",
                    })
                    .to_string(),
                ),
            ];

            write_app_asar(&custom_asar_path, &entries).await?;
        }

        let resource_dir = resource_dir_path(
            &self.discord_location,
            self.discord_location.is_system_electron,
        );
        let asar_path = resource_dir.join("app.asar");
        let _asar_path = resource_dir.join("_app.asar");

        log::info!(
            "Patching {} using custom asar: {:?}",
            self.discord_location.path.as_str(),
            custom_asar_path
        );

        let mut opts: Vec<FileOperation> = vec![];

        super::rename(&asar_path, &_asar_path, &mut opts);
        super::copy(&custom_asar_path, &asar_path, &mut opts);

        #[cfg(target_os = "linux")]
        if self.discord_location.is_system_electron {
            let asar_path = resource_dir.join("app.asar.unpacked");
            let _asar_path = resource_dir.join("_app.asar.unpacked");

            super::rename(&asar_path, &_asar_path, &mut opts);
        }

        #[cfg(target_os = "linux")]
        if self.discord_location.is_flatpak {
            let cmd = self.grant_flatpak_permissions()?;
            log::info!("Flatpak permissions granted with command: {}", cmd);
            super::cmd(&cmd, &mut opts);
        }

        super::execute(&opts, &self.discord_location).await?;

        log::info!("Patch applied successfully!");

        self.discord_location.patched = true;

        Ok(())
    }

    // MARK: - Unpatch
    pub async fn unpatch(&mut self) -> Result<(), Error> {
        if !self.discord_location.patched {
            log::error!("This Discord install is not patched, nothing to do.");
            return Err(Error::ErrLocationNotPatched);
        }

        let resource_dir = resource_dir_path(
            &self.discord_location,
            self.discord_location.is_system_electron,
        );
        let asar_path = resource_dir.join("app.asar");
        let _asar_path = resource_dir.join("_app.asar");

        log::info!("Unpatching {}...", self.discord_location.path.as_str());

        let mut opts: Vec<FileOperation> = vec![];

        if asar_path.exists() {
            super::remove(&asar_path, &mut opts);
        }
        super::rename(&_asar_path, &asar_path, &mut opts);

        #[cfg(target_os = "linux")]
        if self.discord_location.is_system_electron {
            let asar_path = resource_dir.join("app.asar.unpacked");
            let _asar_path = resource_dir.join("_app.asar.unpacked");

            super::rename(&_asar_path, &asar_path, &mut opts);
        }

        super::execute(&opts, &self.discord_location).await?;

        log::info!("Unpatch applied successfully!");

        self.discord_location.patched = false;

        Ok(())
    }

    // MARK: - Repair
    pub async fn repair(&mut self) -> Result<(), Error> {
        const ASAR_NAMES: [&str; 6] = [
            "app.asar",
            "app.asar.unpacked",
            "app.asar.backup",
            "app.asar.original",
            "_app.asar",
            "_app.asar.unpacked",
        ];

        let resource_dir = resource_dir_path(
            &self.discord_location,
            self.discord_location.is_system_electron,
        );

        log::info!("Repairing {}...", self.discord_location.path.as_str());

        let mut opts: Vec<FileOperation> = vec![];

        let mut has_repaired: bool = false;

        for asar_name in ASAR_NAMES {
            let asar_path = resource_dir.join(asar_name);

            if asar_path.exists() {
                if is_asar_original(&asar_path).await {
                    if self.discord_location.is_system_electron {
                        super::rename(
                            &asar_path,
                            &resource_dir.join("app.asar.unpacked"),
                            &mut opts,
                        );
                    } else {
                        super::rename(&asar_path, &resource_dir.join("app.asar"), &mut opts);
                    }
                    has_repaired = true;
                } else if is_asar_considered_openasar(&asar_path) {
                    super::remove(&asar_path, &mut opts);
                } else {
                    super::remove(&asar_path, &mut opts);
                }
            }
        }

        if !has_repaired {
            return Err(Error::ErrPleaseReinstallDiscord);
        }

        super::execute(&opts, &self.discord_location).await?;

        self.discord_location.patched = false;
        self.discord_location.openasar = false;

        Ok(())
    }

    // MARK: - Patch OpenAsar
    #[cfg(feature = "openasar")]
    pub async fn patch_openasar(&mut self, patched_asar_file_url: &str) -> Result<(), Error> {
        if self.discord_location.openasar {
            log::error!("This Discord install is already patched with OpenAsar, nothing to do.");
            return Err(Error::ErrLocationPatched);
        }

        #[cfg(target_os = "windows")]
        if is_scuffed_install(&self.discord_location.name) {
            log::error!("You have a broken Discord install. Please reinstall Discord!");
            return Err(Error::ErrWindowsMovedDirectory);
        }

        let data_path = &self.data_path.clone().ok_or(Error::ErrNoDataPath)?;

        let resource_dir = resource_dir_path(
            &self.discord_location,
            self.discord_location.is_system_electron,
        );
        let asar_path = resource_dir.join(use_appropriate_asar(self.discord_location.patched));
        let dl_tmp_asar_path = data_path.join("app.asar");

        log::info!(
            "Patching {} using remote asar: {}",
            self.discord_location.path.as_str(),
            patched_asar_file_url
        );

        download_file(patched_asar_file_url, dl_tmp_asar_path.clone()).await?;

        let mut opts: Vec<FileOperation> = vec![];
        opts.push(FileOperation::Move {
            from: asar_path.clone(),
            to: resource_dir.join("app.asar.backup"),
        });

        opts.push(FileOperation::Copy {
            from: dl_tmp_asar_path,
            to: asar_path,
        });
        super::execute(&opts, &self.discord_location).await?;

        log::info!("Patch applied successfully!");

        self.discord_location.openasar = true;

        Ok(())
    }

    // MARK: - Unpatch OpenAsar
    #[cfg(feature = "openasar")]
    pub async fn unpatch_openasar(&mut self) -> Result<(), Error> {
        if !self.discord_location.openasar {
            log::error!("This Discord install is not patched with OpenAsar, nothing to do.");
            return Err(Error::ErrLocationNotPatched);
        }

        let resource_dir = resource_dir_path(
            &self.discord_location,
            self.discord_location.is_system_electron,
        );
        let asar_path = resource_dir.join(use_appropriate_asar(self.discord_location.patched));

        log::info!("Unpatching {}", self.discord_location.path.as_str());

        let mut opts: Vec<FileOperation> = vec![];

        if asar_path.exists() {
            opts.push(FileOperation::Remove {
                path: asar_path.clone(),
            });
        }

        let backup_paths = [
            resource_dir.join("app.asar.backup"),
            resource_dir.join("app.asar.original"),
        ];

        match backup_paths.iter().find(|&path| path.exists()) {
            Some(backup) => {
                opts.push(FileOperation::Move {
                    from: backup.clone(),
                    to: asar_path,
                });
            }
            _ => return Err(Error::ErrLocationInvalid),
        }

        super::execute(&opts, &self.discord_location).await?;

        log::info!("Unpatch applied successfully!");

        self.discord_location.openasar = false;

        Ok(())
    }
}

impl Installer {
    // MARK: - Flatpak Permissions
    #[cfg(target_os = "linux")]
    pub fn grant_flatpak_permissions(&self) -> Result<String, Error> {
        let data_path = self.data_path.clone().ok_or(Error::ErrNoDataPath)?;

        log::info!(
            "Location is flatpak, granting perms to {}",
            data_path.to_string_lossy()
        );

        let name = self
            .discord_location
            .path
            .split('/')
            .find(|s| s.starts_with("com.discordapp."))
            .unwrap_or("");

        let is_system_flatpak = self.discord_location.path.contains("/var");

        let mut args = vec![];

        if !is_system_flatpak {
            args.push("--user");
        }
        args.push("override");
        args.push(name);
        let filesystem_arg = format!("--filesystem={}", &data_path.to_string_lossy());
        args.push(&filesystem_arg);

        Ok(format!("flatpak {}", args.join(" ")))
    }
}
