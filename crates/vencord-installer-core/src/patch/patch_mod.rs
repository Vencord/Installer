use std::path::{Path, PathBuf};

use crate::Error;
#[cfg(feature = "generate_asar")]
use crate::asar::write_app_asar;
#[cfg(target_os = "windows")]
use crate::paths::locations::is_scuffed_install;
use crate::paths::shared::resource_dir_path;
use crate::paths::{branch::DiscordLocation, shared::is_asar_original};
use crate::update::download::download_file;

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

    // MARK: - Vencord

    pub async fn patch(&mut self) -> Result<(), Error> {
        if self.discord_location.patched {
            return Err(Error::ErrLocationPatched);
        }

        let data_path = self.data_path.as_ref().ok_or(Error::ErrNoDataPath)?;

        let res_dir = resource_dir_path(
            &self.discord_location,
            self.discord_location.is_system_electron,
        );

        let live_path = res_dir.join(if self.discord_location.is_system_electron {
            "app.asar.unpacked"
        } else {
            "app.asar"
        });
        let mod_backup = live_path.with_file_name("_app.asar");
        let custom_asar = data_path.join("app.asar");

        #[cfg(feature = "generate_asar")]
        self.generate_patcher_asar(&custom_asar, data_path).await?;

        let mut opts = vec![];

        if !self.discord_location.openasar {
            let root_original = self.find_root_original().await?;
            if root_original != live_path {
                super::rename(&root_original, &live_path, &mut opts);
            }
        }

        super::rename(&live_path, &mod_backup, &mut opts);
        super::copy(&custom_asar, &live_path, &mut opts);
        super::execute(&opts, &self.discord_location).await?;

        #[cfg(target_os = "linux")]
        if self.discord_location.is_flatpak {
            super::cmd(&self.grant_flatpak_permissions()?, &mut opts);
        }

        self.discord_location.patched = true;

        Ok(())
    }

    // MARK: - Vencord (Remove)

    pub async fn unpatch(&mut self) -> Result<(), Error> {
        let res_dir = resource_dir_path(
            &self.discord_location,
            self.discord_location.is_system_electron,
        );

        let live_path = res_dir.join(if self.discord_location.is_system_electron {
            "app.asar.unpacked"
        } else {
            "app.asar"
        });

        let mod_backup = live_path.with_file_name("_app.asar");

        if !mod_backup.exists() {
            return Err(Error::ErrLocationNotPatched);
        }

        let mut opts = vec![];

        super::remove(&live_path, &mut opts);
        super::rename(&mod_backup, &live_path, &mut opts);
        super::execute(&opts, &self.discord_location).await?;

        self.discord_location.patched = false;

        Ok(())
    }

    // MARK: - OpenAsar

    #[cfg(feature = "openasar")]
    pub async fn patch_openasar(&mut self, url: &str) -> Result<(), Error> {
        if self.discord_location.openasar {
            return Err(Error::ErrLocationPatched);
        }
        let data_path = self.data_path.as_ref().ok_or(Error::ErrNoDataPath)?;
        let res_dir = resource_dir_path(
            &self.discord_location,
            self.discord_location.is_system_electron,
        );

        let root_original = self.find_root_original().await?;
        let permanent_backup = res_dir.join("app.asar.original");
        let dl_path = data_path.join("open.asar");

        download_file(url, dl_path.clone()).await?;

        let mut opts = vec![];

        if root_original != permanent_backup {
            super::copy(&root_original, &permanent_backup, &mut opts);
        }

        let target = if self.discord_location.patched {
            res_dir.join("_app.asar")
        } else {
            res_dir.join(if self.discord_location.is_system_electron {
                "app.asar.unpacked"
            } else {
                "app.asar"
            })
        };

        super::copy(&dl_path, &target, &mut opts);
        super::remove(&dl_path, &mut opts);
        super::execute(&opts, &self.discord_location).await?;

        #[cfg(target_os = "linux")]
        if self.discord_location.is_flatpak {
            super::cmd(&self.grant_flatpak_permissions()?, &mut opts);
        }

        self.discord_location.openasar = true;

        Ok(())
    }

    // MARK: - OpenAsar (Remove)

    #[cfg(feature = "openasar")]
    pub async fn unpatch_openasar(&mut self) -> Result<(), Error> {
        let res_dir = resource_dir_path(
            &self.discord_location,
            self.discord_location.is_system_electron,
        );
        let permanent_backup = res_dir.join("app.asar.original");

        if !permanent_backup.exists() {
            return Err(Error::ErrLocationNotPatched);
        }

        let target = if self.discord_location.patched {
            res_dir.join("_app.asar")
        } else {
            res_dir.join(if self.discord_location.is_system_electron {
                "app.asar.unpacked"
            } else {
                "app.asar"
            })
        };

        let mut opts = vec![];

        super::copy(&permanent_backup, &target, &mut opts);
        super::remove(&permanent_backup, &mut opts);
        super::execute(&opts, &self.discord_location).await?;

        self.discord_location.openasar = false;

        Ok(())
    }

    // MARK: - Repair

    pub async fn repair(&mut self) -> Result<(), Error> {
        let res_dir = resource_dir_path(
            &self.discord_location,
            self.discord_location.is_system_electron,
        );
        let root_original = self.find_root_original().await?;
        let live_path = res_dir.join(if self.discord_location.is_system_electron {
            "app.asar.unpacked"
        } else {
            "app.asar"
        });

        let mut opts = vec![];

        if root_original != live_path {
            super::copy(&root_original, &live_path, &mut opts);
        }

        let artifacts = [
            "_app.asar",
            "app.asar.backup",
            "app.asar.original",
            "_app.asar.unpacked",
        ];

        for name in artifacts {
            let path = res_dir.join(name);
            if path.exists() && path != live_path {
                super::remove(&path, &mut opts);
            }
        }

        super::execute(&opts, &self.discord_location).await?;

        self.discord_location.patched = false;
        self.discord_location.openasar = false;

        Ok(())
    }
}

// MARK: - Misc

impl Installer {
    async fn find_root_original(&self) -> Result<PathBuf, Error> {
        let res_dir = resource_dir_path(
            &self.discord_location,
            self.discord_location.is_system_electron,
        );

        let names = [
            "app.asar.original",
            "app.asar.backup",
            "_app.asar",
            "app.asar",
        ];

        for name in names {
            let path = res_dir.join(name);
            if path.exists() && is_asar_original(&path).await {
                return Ok(path);
            }
        }

        Err(Error::ErrPleaseReinstallDiscord)
    }

    #[cfg(target_os = "linux")]
    pub fn grant_flatpak_permissions(&self) -> Result<String, Error> {
        let data_path = self.data_path.clone().ok_or(Error::ErrNoDataPath)?;
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
        args.push(&format!("--filesystem={}", &data_path.to_string_lossy()));

        Ok(format!("flatpak {}", args.join(" ")))
    }

    #[cfg(feature = "generate_asar")]
    async fn generate_patcher_asar(
        &self,
        destination: &Path,
        data_path: &Path,
    ) -> Result<(), Error> {
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
                &json!({ "name": "discord", "main": "index.js" }).to_string(),
            ),
        ];

        write_app_asar(destination, &entries).await
    }
}
