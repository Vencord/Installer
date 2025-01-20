use serde::{Serialize, Deserialize};

#[derive(Serialize, Deserialize, Debug)]
pub enum ErrorCode {
    /// Cannot modify a program while its in use or another program is using it
    ErrWindowsFileLock,
    /// Some windows bullshit where they moved the directory somewhere i don't know
    ErrWindowsMovedDirectory,
    /// macOS error when the user didn't give the program App Management permissions
    ErrMacMissingPermissions,
    /// User somehow has put discord in ~/Desktop, causing confusion
    ErrMacDiscordInDesktop,
    /// User has ran the patch function on an already patched location
    ErrAlreadyPatched,
    /// User has ran the unpatch function on an already unpatched location
    ErrAlreadyUnpatched,
    /// Network error
    ErrNetwork,
    /// Unknown error
    ErrUnknown,
}

pub type InstallerResult<T> = std::result::Result<T, InstallerError>;

#[derive(Serialize, Deserialize, Debug, thiserror::Error)]
pub struct InstallerError {
    pub code: ErrorCode,
    pub message: String,
}

impl std::fmt::Display for InstallerError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{} ({:?})", self.message, self.code)
    }
}

impl From<std::io::Error> for InstallerError {
    fn from(value: std::io::Error) -> Self {
        InstallerError {
            message: value.to_string(),
            code: match (value.raw_os_error(), std::env::consts::OS) {
                (Some(32), "windows") => ErrorCode::ErrWindowsFileLock,
                (Some(1), "macos") => ErrorCode::ErrMacMissingPermissions,
                _ => ErrorCode::ErrUnknown,
            },
        }
    }
}

impl From<Box<dyn std::error::Error>> for InstallerError {
    fn from(value: Box<dyn std::error::Error>) -> Self {
        InstallerError {
            message: value.to_string(),
            code: ErrorCode::ErrUnknown,
        }
    }
}

impl From<reqwest::Error> for InstallerError {
    fn from(value: reqwest::Error) -> Self {
        InstallerError {
            message: value.to_string(),
            code: ErrorCode::ErrNetwork,
        }
    }
}

impl From<serde_json::Error> for InstallerError {
    fn from(error: serde_json::Error) -> Self {
        InstallerError {
            message: error.to_string(),
            code: ErrorCode::ErrUnknown,
        }
    }
}

impl From<String> for InstallerError {
    fn from(value: String) -> Self {
        InstallerError {
            message: value,
            code: ErrorCode::ErrUnknown,
        }
    }
}

pub fn error_with_code(message: &str, code: ErrorCode) -> InstallerResult<()> {
    Err(InstallerError {
        message: message.to_string(),
        code,
    })
}