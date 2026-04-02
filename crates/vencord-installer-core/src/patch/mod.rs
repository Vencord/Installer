pub mod patch_mod;

use crate::{Error, paths::branch::DiscordLocation};
use std::path::PathBuf;

#[cfg(target_os = "linux")]
use std::os::linux::fs::MetadataExt;

#[cfg(target_os = "linux")]
unsafe extern "C" {
    fn geteuid() -> u32;
}

#[derive(Debug, Clone)]
pub enum FileOperation {
    Move {
        from: PathBuf,
        to: PathBuf,
    },
    Copy {
        from: PathBuf,
        to: PathBuf,
    },
    Remove {
        path: PathBuf,
    },
    #[cfg(target_os = "linux")]
    Cmd {
        string: String,
    },
}

impl FileOperation {
    #[cfg(target_os = "linux")]
    fn to_shell_command(&self) -> String {
        match self {
            FileOperation::Move { from, to } => {
                format!("mv '{}' '{}'", from.display(), to.display())
            }
            FileOperation::Copy { from, to } => {
                format!("cp '{}' '{}'", from.display(), to.display())
            }
            FileOperation::Remove { path } => {
                format!("rm -f '{}'", path.display())
            }
            FileOperation::Cmd { string } => string.clone(),
        }
    }
}

pub fn rename(from: &PathBuf, to: &PathBuf, opt: &mut Vec<FileOperation>) {
    opt.push(FileOperation::Move {
        from: from.clone(),
        to: to.clone(),
    });
}

pub fn copy(from: &PathBuf, to: &PathBuf, opt: &mut Vec<FileOperation>) {
    opt.push(FileOperation::Copy {
        from: from.clone(),
        to: to.clone(),
    });
}

pub fn remove(path: &PathBuf, opt: &mut Vec<FileOperation>) {
    opt.push(FileOperation::Remove { path: path.clone() });
}

#[cfg(target_os = "linux")]
pub fn cmd(command: &str, opt: &mut Vec<FileOperation>) {
    opt.push(FileOperation::Cmd {
        string: command.to_owned(),
    });
}

pub async fn execute(
    operations: &[FileOperation],
    _location: &DiscordLocation,
) -> Result<(), Error> {
    log::debug!("Running operations: {:#?}", operations);
    if operations.is_empty() {
        return Ok(());
    }

    let mut _needs_elevated = false;

    #[cfg(target_os = "linux")]
    {
        for op in operations {
            let path_to_check = match op {
                FileOperation::Move { from, .. } => from,
                FileOperation::Copy { to, .. } => to,
                FileOperation::Remove { path } => path,
                _ => continue,
            };

            if let Some(parent) = path_to_check.parent() {
                if std::fs::metadata(parent).is_ok() {
                    let metadata = std::fs::metadata(parent).map_err(|e| Error::from(e))?;
                    if metadata.permissions().readonly()
                        || unsafe { geteuid() } != 0 && metadata.st_uid() == 0
                    {
                        _needs_elevated = true;
                        break;
                    }
                }
            }
        }
    }

    // If we need elevated permissions, execute all operations with pkexec
    // in a single command, so it only prompts once, only for linux as well
    if _needs_elevated {
        #[cfg(target_os = "linux")]
        {
            log::warn!("Detected protected directory, using pkexec for atomic batch...");
            let commands: Vec<String> = operations.iter().map(|op| op.to_shell_command()).collect();
            let combined_command = commands.join(" && ");

            let status = tokio::process::Command::new("pkexec")
                .arg("sh")
                .arg("-c")
                .arg(&combined_command)
                .status()
                .await?;

            if status.success() {
                return Ok(());
            } else {
                return Err(Error::ErrPermissionDenied);
            }
        }
    } else {
        for operation in operations {
            match operation {
                FileOperation::Move { from, to } => {
                    tokio::fs::rename(from, to)
                        .await
                        .map_err(|e| Error::from(e))?;
                }
                FileOperation::Copy { from, to } => {
                    tokio::fs::copy(from, to)
                        .await
                        .map_err(|e| Error::from(e))?;

                    #[cfg(target_os = "linux")]
                    unsafe {
                        if geteuid() == 0 && !_location.is_flatpak {
                            crate::paths::locations::copy_ownership_permissions(&to)
                                .await
                                .ok();
                        }
                    }
                }
                FileOperation::Remove { path } => {
                    if path.exists() {
                        if path.is_dir() {
                            tokio::fs::remove_dir_all(path).await?
                        } else {
                            tokio::fs::remove_file(path).await?
                        }
                    }
                }
                #[cfg(target_os = "linux")]
                FileOperation::Cmd { string } => {
                    tokio::process::Command::new("sh")
                        .arg("-c")
                        .arg(string)
                        .status()
                        .await
                        .ok();
                }
            }
        }
    }

    Ok(())
}
