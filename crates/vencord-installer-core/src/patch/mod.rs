pub mod patch_mod;

use crate::{Error, paths::branch::DiscordLocation};
use std::path::PathBuf;

#[cfg(any(target_os = "linux"))]
unsafe extern "C" {
    unsafe fn geteuid() -> u32;
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
                format!("rm '{}'", path.display())
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
    let mut needs_elevated = false;

    log::debug!("Running operations: {:#?}", operations);

    for operation in operations {
        let result = match operation {
            FileOperation::Move { from, to } => tokio::fs::rename(from, to).await,
            FileOperation::Copy { from, to } => {
                tokio::fs::copy(from, to).await.map(|_| ())?;
                // users on linux running with sudo need special treatment
                #[cfg(target_os = "linux")]
                unsafe {
                    if geteuid() == 0 {
                        if !_location.is_flatpak {
                            crate::paths::locations::copy_ownership_permissions(&to)
                                .await
                                .ok();
                        }
                    }
                }
                Ok(())
            }
            FileOperation::Remove { path } => tokio::fs::remove_file(path).await,
            #[cfg(target_os = "linux")]
            FileOperation::Cmd { string } => {
                tokio::process::Command::new("sh")
                    .arg("-c")
                    .arg(string)
                    .status()
                    .await
                    .ok();
                Ok(())
            }
        };

        if let Err(e) = result {
            if e.kind() == std::io::ErrorKind::PermissionDenied {
                needs_elevated = true;
                break;
            } else {
                return Err(Error::from(e));
            }
        }
    }

    // If we need elevated permissions, execute all operations with pkexec
    // in a single command, so it only prompts once, only for linux as well
    if needs_elevated {
        #[cfg(target_os = "linux")]
        {
            log::warn!("Permission was denied, attempting to use pkexec instead...");

            let commands: Vec<String> = operations.iter().map(|op| op.to_shell_command()).collect();

            let combined_command = commands.join(" && ");

            let status = tokio::process::Command::new("pkexec")
                .arg("sh")
                .arg("-c")
                .arg(&combined_command)
                .status()
                .await?;

            if status.success() {
                Ok(())
            } else {
                Err(Error::ErrPermissionDenied)
            }
        }
        #[cfg(not(target_os = "linux"))]
        {
            Err(Error::ErrPermissionDenied)
        }
    } else {
        Ok(())
    }
}
