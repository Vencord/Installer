use std::path::{Path, PathBuf};

use crate::Error;

use super::DiscordBranch;

// #[readonly::make]
#[derive(Debug, PartialEq, Clone)]
pub struct DiscordLocation {
    /// The full path to the Discord installation
    pub path: PathBuf,
    /// The branch of the Discord installation
    pub branch: DiscordBranch,
    /// Whether the Discord installation has been patched with Vencord
    pub is_vencord: bool,
    /// Whether the Discord installation has been patched with openasar
    pub is_openasar: bool,
    /// If its a flatpak installation: https://flatpak.org/
    pub is_flatpak: bool,
    /// If the installation is in a scuffed location on Windows.
    pub is_scuffed: bool,
    /// Arch package, needs special care: https://aur.archlinux.org/packages/discord_arch_electron
    pub is_system_electron: bool,
}

impl Default for DiscordLocation {
    fn default() -> Self {
        Self {
            path: PathBuf::new(),
            branch: DiscordBranch::default(),
            is_vencord: false,
            is_openasar: false,
            is_flatpak: false,
            is_scuffed: false,
            is_system_electron: false,
        }
    }
}

impl DiscordLocation {
    pub fn from_path<P: AsRef<Path>>(full_path: P) -> Option<Self> {
        let full_path = full_path.as_ref().to_path_buf();

        let mut discord = DiscordLocation {
            path: full_path.clone(),
            branch: DiscordBranch::from_path(full_path),
            ..Default::default()
        };

        // Order matters here, system electron package is jank and needs special treatment.
        #[cfg(target_os = "linux")]
        {
            discord.is_system_electron = !discord.resources_dir().exists();
        }

        if !(discord.asar_path().exists()
            || discord.asar_patched_path().exists()
            || discord.asar_openasar_path().exists())
        {
            return None;
        }

        #[cfg(target_os = "linux")]
        {
            discord.is_flatpak = discord.path.to_string_lossy().contains("flatpak");
        }

        // _app.asar(.unpacked)
        discord.is_vencord = discord.asar_patched_path().exists();
        // app.asar.original
        discord.is_openasar = discord.asar_openasar_path().exists();

        // Sometimes Discord likes installing to an improper location for some reason
        // we need to make the user properly uninstall and reinstall Discord so its
        // in the correct location so we can patch it.
        #[cfg(target_os = "windows")]
        {
            discord.is_scuffed = check_for_scuffed_windows_location(&discord.path);
        }

        Some(discord)
    }

    pub fn resources_dir(&self) -> PathBuf {
        #[cfg(target_os = "macos")]
        return self.path.join("Contents").join("Resources");
        #[cfg(not(target_os = "macos"))]
        return self.path.join("resources");
    }

    pub fn asar_path(&self) -> PathBuf {
        #[cfg(target_os = "linux")]
        if !self.is_system_electron {
            self.resources_dir().join("app.asar")
        } else {
            self.path.join("app.asar.unpacked")
        }

        #[cfg(not(target_os = "linux"))]
        self.resources_dir().join("app.asar")
    }

    pub fn asar_patched_path(&self) -> PathBuf {
        #[cfg(target_os = "linux")]
        if !self.is_system_electron {
            self.resources_dir().join("_app.asar")
        } else {
            self.path.join("_app.asar.unpacked")
        }

        #[cfg(not(target_os = "linux"))]
        self.resources_dir().join("_app.asar")
    }

    pub fn asar_openasar_path(&self) -> PathBuf {
        #[cfg(target_os = "linux")]
        if !self.is_system_electron {
            self.resources_dir().join("app.asar.original")
        } else {
            self.path.join("app.asar.original")
        }

        #[cfg(not(target_os = "linux"))]
        self.resources_dir().join("app.asar.original")
    }

    pub fn find_asars(&self) -> Result<(PathBuf, Vec<PathBuf>), Error> {
        let asars = [
            self.asar_openasar_path(),
            self.asar_patched_path(),
            self.asar_path(),
        ];

        let existing: Vec<_> = asars.into_iter().filter(|p| p.exists()).collect();

        let original = existing
            .iter()
            .max_by_key(|path| std::fs::metadata(path).map(|m| m.len()).unwrap_or(0))
            .cloned()
            .ok_or(Error::ErrPleaseReinstallDiscord)?;

        let fake = existing
            .into_iter()
            .filter(|path| path != &original)
            .collect();

        Ok((original, fake))
    }

    #[cfg(target_os = "windows")]
    pub fn find_process_id(&self) -> u32 {
        use windows::Win32::System::Diagnostics::ToolHelp::*;

        let target = format!("{}.exe", self.branch.to_string().replace(' ', ""));

        unsafe {
            let snapshot = match CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS, 0) {
                Ok(s) => s,
                Err(_) => return 0,
            };

            let mut entry = PROCESSENTRY32W::default();
            entry.dwSize = std::mem::size_of::<PROCESSENTRY32W>() as u32;

            if Process32FirstW(snapshot, &mut entry).is_err() {
                return 0;
            }

            loop {
                let exe = String::from_utf16_lossy(
                    &entry.szExeFile[..entry
                        .szExeFile
                        .iter()
                        .position(|&c| c == 0)
                        .unwrap_or(entry.szExeFile.len())],
                );

                if exe.eq_ignore_ascii_case(&target) {
                    return entry.th32ProcessID;
                }

                if Process32NextW(snapshot, &mut entry).is_err() {
                    return 0;
                }
            }
        }
    }

    #[cfg(target_os = "windows")]
    pub fn kill_process(&self) -> Result<(), Error> {
        use std::{thread, time::Duration};
        use windows::Win32::System::Threading::*;

        let pid = self.find_process_id();

        if pid == 0 {
            return Ok(());
        }

        unsafe {
            let handle = OpenProcess(PROCESS_TERMINATE, false, pid)
                .map_err(|_| Error::ErrDiscordOpened)?;

            let _ = TerminateProcess(handle, 1);
            let _ = windows::Win32::Foundation::CloseHandle(handle);
        }

        loop {
            if self.find_process_id() == 0 {
                break;
            }

            thread::sleep(Duration::from_millis(100));
        }

        Ok(())
    }

}

#[cfg(target_os = "windows")]
fn check_for_scuffed_windows_location<P: AsRef<Path>>(path: P) -> bool {
    let path = path.as_ref();

    let Some(file_name) = path.file_name() else {
        return false;
    };

    let Some(program_data) = crate::paths::get_program_data_path() else {
        return false;
    };

    program_data
        .join(file_name)
        .exists()
}
