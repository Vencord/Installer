#![warn(mismatched_lifetime_syntaxes)]
#![warn(clippy::pedantic)]
#![allow(clippy::module_name_repetitions)]

pub mod patch;
pub mod paths;
pub mod update;

pub(crate) const USER_AGENT: &str = "VencordInstaller (https://github.com/Vencord/Installer)";
pub(crate) const RELEASE_URL: &str =
    "https://api.github.com/repos/Vendicated/Vencord/releases/latest";
pub(crate) const RELEASE_URL_FALLBACK: &str = "https://vencord.dev/releases/vencord";
pub(crate) const RELEASE_TAG_DOWNLOAD: &str =
    "https://github.com/Vendicated/Vencord/releases/download/devbuild";
pub(crate) const OPENASAR_URL: &str =
    "https://github.com/GooseMod/OpenAsar/releases/download/nightly/app.asar";

pub async fn download() -> Result<(), Error> {
    let latest_version =
        update::version_check::check_hash_from_release(RELEASE_URL, RELEASE_URL_FALLBACK).await?;

    let data_path = paths::get_data_path().ok_or(Error::ErrNoDataPath)?;

    let local_version = update::version_check::check_local_version(&data_path).await?;

    if latest_version != local_version {
        update::download::prepare_dist_directory(
            &data_path,
            RELEASE_TAG_DOWNLOAD,
            ["patcher.js", "preload.js", "renderer.js", "renderer.css"],
        )
        .await?;
    }

    Ok(())
}

use thiserror::Error as ThisError;

#[derive(Debug, ThisError)]
pub enum Error {
    #[error("Permission denied. Please run the installer with appropriate permissions.")]
    ErrPermissionDenied,
    #[error("You have a broken Discord install. Please reinstall Discord!")]
    ErrWindowsMovedDirectory,
    #[error("This Discord install is already patched, nothing to do.")]
    ErrLocationPatched,
    #[error("This Discord install is not patched, nothing to do.")]
    ErrLocationNotPatched,
    #[error(
        "This Discord install seems incorrect, please specify a valid location! Hint: snap is not supported."
    )]
    ErrLocationInvalid,
    #[error("No data path specified for patching.")]
    ErrNoDataPath,
    #[error("Discord instance is corrupted or missing files. Please reinstall this Discord!")]
    ErrPleaseReinstallDiscord,
    #[error("Before patching, please make sure Discord is fully closed!")]
    ErrDiscordOpened,
    #[error("Invalid arguments provided: {0}")]
    ErrInvalidArguments(&'static str),
    #[error("Other error: {0}")]
    ErrOther(&'static str),
    #[error("Network error: {0}")]
    ErrReqwest(#[from] reqwest::Error),
    #[error("Serde json error: {0}")]
    ErrSerdeJson(#[from] serde_json::Error),
    #[error("I/O error: {0}")]
    ErrIo(#[from] std::io::Error),
    #[error("Tokio Join error: {0}")]
    ErrJoin(#[from] tokio::task::JoinError),
    #[error("Regex error: {0}")]
    ErrRegex(#[from] regex::Error),
}

impl Error {
    pub fn format_error(&self) -> String {
        match self {
            Self::ErrWindowsMovedDirectory => format!(
                "{}\n\nSometimes Discord installs to the wrong location.\n\
             Fix this before patching.\n\n\
             Delete any 'Discord' or 'Squirrel' folders in that location.\n\
             If empty, remove the parent folder too.\n\
             Then try launching Discord again.",
                self
            ),

            Self::ErrIo(err) => Self::format_io_error(err),

            Self::ErrReqwest(err) => format!(
                "{}\n\nMake sure you're connected to the internet.\n\
             If blocked, GitHub may be restricted — try a VPN.",
                err
            ),

            _ => self.to_string(),
        }
    }

    fn format_io_error(err: &std::io::Error) -> String {
        #[cfg(target_os = "macos")]
        {
            if err.kind() == std::io::ErrorKind::PermissionDenied {
                return format!(
                    "{}\n\nGrant 'App Management' permissions in System Settings.",
                    err
                );
            }
        }

        format!("I/O error: {}", err)
    }
}

impl Error {
    pub fn is_permission_denied(&self) -> bool {
        matches!(
            self,
            Error::ErrIo(err) if err.kind() == std::io::ErrorKind::PermissionDenied
        )
    }

    pub fn is_windows_moved_dir(&self) -> bool {
        matches!(self, Error::ErrWindowsMovedDirectory)
    }
}
