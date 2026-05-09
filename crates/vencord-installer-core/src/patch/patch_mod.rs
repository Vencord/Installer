use std::path::PathBuf;

use crate::Error;

use crate::paths::DiscordLocation;
use crate::update::download::download_file;

pub struct Installer {
    discord_location: DiscordLocation,
    data_path: PathBuf,
}

impl Installer {
    pub fn new(
        discord_location: DiscordLocation,
        data_path: Option<PathBuf>,
    ) -> Result<Self, Error> {
        #[cfg(target_os = "windows")]
        {
            if discord_location.is_scuffed {
                log::error!("You have a broken Discord install. Please reinstall Discord!");
                return Err(Error::ErrWindowsMovedDirectory);
            }

            discord_location.kill_process()?;
        }

        let data_path = data_path.ok_or(Error::ErrNoDataPath)?;

        Ok(Installer {
            discord_location,
            data_path,
        })
    }

    // MARK: - Vencord

    pub async fn patch(&mut self) -> Result<(), Error> {
        let live_path = self.discord_location.asar_path();
        let mod_backup = self.discord_location.asar_patched_path();
        let custom_asar = self.data_path.join("app.asar");

        super::asar::generate_patcher_asar(&custom_asar, &self.data_path).await?;

        let mut opts = vec![];

        super::rename(&live_path, &mod_backup, &mut opts);
        super::copy(&custom_asar, &live_path, &mut opts);

        #[cfg(target_os = "linux")]
        if self.discord_location.is_flatpak {
            super::cmd(&self.grant_flatpak_permissions()?, &mut opts);
        }

        super::execute(&opts, &self.discord_location).await?;

        Ok(())
    }

    // MARK: - Vencord (Remove)

    pub async fn unpatch(&mut self) -> Result<(), Error> {
        let live_path = self.discord_location.asar_path();
        let mod_backup = self.discord_location.asar_patched_path();

        let mut opts = vec![];

        super::remove(&live_path, &mut opts);
        super::rename(&mod_backup, &live_path, &mut opts);
        super::execute(&opts, &self.discord_location).await?;

        Ok(())
    }

    // MARK: - OpenAsar

    #[cfg(feature = "openasar")]
    pub async fn patch_openasar(&mut self) -> Result<(), Error> {
        let root_original = self.discord_location.find_asars()?.0;
        let permanent_backup = self.discord_location.asar_openasar_path();
        let dl_path = self.data_path.join("open.asar");

        download_file(crate::OPENASAR_URL, dl_path.clone()).await?;

        let mut opts = vec![];

        if root_original != permanent_backup {
            super::copy(&root_original, &permanent_backup, &mut opts);
        }

        let target = if self.discord_location.is_vencord {
            self.discord_location.asar_patched_path()
        } else {
            self.discord_location.asar_path()
        };

        super::copy(&dl_path, &target, &mut opts);
        super::remove(&dl_path, &mut opts);
        super::execute(&opts, &self.discord_location).await?;

        #[cfg(target_os = "linux")]
        if self.discord_location.is_flatpak {
            super::cmd(&self.grant_flatpak_permissions()?, &mut opts);
        }

        Ok(())
    }

    // MARK: - OpenAsar (Remove)

    #[cfg(feature = "openasar")]
    pub async fn unpatch_openasar(&mut self) -> Result<(), Error> {
        let permanent_backup = self.discord_location.asar_openasar_path();

        let target = if self.discord_location.is_vencord {
            self.discord_location.asar_patched_path()
        } else {
            self.discord_location.asar_path()
        };

        let mut opts = vec![];

        super::copy(&permanent_backup, &target, &mut opts);
        super::remove(&permanent_backup, &mut opts);
        super::execute(&opts, &self.discord_location).await?;

        Ok(())
    }

    // MARK: - Repair

    pub async fn repair(&mut self) -> Result<(), Error> {
        let locations = self.discord_location.find_asars()?;

        let mut opts = vec![];

        for asar in locations.1 {
            super::remove(&asar, &mut opts);
        }

        super::rename(&locations.0, &self.discord_location.asar_path(), &mut opts);

        super::execute(&opts, &self.discord_location).await?;

        Ok(())
    }
}

// MARK: - Misc

impl Installer {
    #[cfg(target_os = "linux")]
    pub fn grant_flatpak_permissions(&self) -> Result<String, Error> {
        log::info!(
            "Location is flatpak, granting perms to {}",
            self.data_path.to_string_lossy()
        );

        let name = self
            .discord_location
            .path
            .to_str()
            .unwrap_or("")
            .split('/')
            .find(|s| s.starts_with("com.discordapp."))
            .unwrap_or("");

        let is_system_flatpak = self
            .discord_location
            .path
            .to_string_lossy()
            .contains("/var");

        let mut args = vec![];

        if !is_system_flatpak {
            args.push("--user");
        }
        args.push("override");
        args.push(name);
        let filesystem_arg = format!("--filesystem={}", &self.data_path.to_string_lossy());
        args.push(&filesystem_arg);

        Ok(format!("flatpak {}", args.join(" ")))
    }
}
