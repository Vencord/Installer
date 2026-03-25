#[cfg(feature = "generate_asar")]
pub mod asar;
pub mod patch;
pub mod paths;
pub mod update;

pub const USER_AGENT: &str = "VencordInstaller (https://github.com/Vencord/Installer)";
pub const RELEASE_URL: &str = "https://api.github.com/repos/Vendicated/Vencord/releases/latest";
pub const RELEASE_URL_FALLBACK: &str = "https://vencord.dev/releases/vencord";
pub const RELEASE_TAG_DOWNLOAD: &str =
    "https://github.com/Vendicated/Vencord/releases/download/devbuild";
pub const OPENASAR_URL: &str =
    "https://github.com/GooseMod/OpenAsar/releases/download/nightly/app.asar";

pub fn get_dist_path(name: Option<&str>) -> std::path::PathBuf {
    let name = name.unwrap_or("Vencord");

    if let Ok(path) = std::env::var("VENCORD_USER_DATA_DIR") {
        std::path::PathBuf::from(path).join("dist")
    } else {
        paths::locations::get_data_path(name)
    }
}

pub async fn download() -> Result<(), Error> {
    let latest_version =
        update::version_check::check_hash_from_release(RELEASE_URL, Some(RELEASE_URL_FALLBACK))
            .await;

    let local_version =
        update::version_check::check_local_version(&get_dist_path(None), None).await;

    if latest_version.is_some() && latest_version != local_version {
        update::download::prepare_dist_directory(
            &get_dist_path(None),
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
}

impl Error {
    pub fn format_error(&self) -> String {
        match self {
            Self::ErrWindowsMovedDirectory => {
                format!(
                    "{}\n\nSometimes Discord decides to install to the wrong location for some reason!\n\
                 You need to fix this before patching, otherwise Vencord will likely not work.\n\n\
                 Use the below button to jump there and delete any folder called Discord or Squirrel.\n\
                 If the folder is now empty, feel free to go back a step and delete that folder too.\n\
                 Then see if Discord still starts. If not, reinstall it.",
                    self.to_string()
                )
            }
            Self::ErrIo(err) => {
                #[cfg(target_os = "macos")]
                {
                    match err.kind() {
                        std::io::ErrorKind::PermissionDenied => {
                            format!(
                                "{}\n\nOn macOS, you need to grant 'App Management' permissions in System Preferences > Security & Privacy.",
                                err
                            )
                        }
                        _ => format!("I/O error: {}", err),
                    }
                }
                #[cfg(not(target_os = "macos"))]
                {
                    format!("I/O error: {}", err)
                }
            }
            Self::ErrReqwest(err) => format!(
                "{}\n\nMake sure you're connected to the internet!\n\nIf it's still blocked, github may be blocked in your country or your isp, if thats the case, setup or use a VPN to bypass it.",
                err
            ),
            _ => self.to_string(),
        }
    }
}
