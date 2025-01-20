/// The branch of a Discord installation
#[derive(Debug, PartialEq, Clone)]
pub enum DiscordBranch {
    Stable,
    PTB,
    Canary,
    Development
}

impl DiscordBranch {
    /// Returns the DiscordBranch from the given path.
    pub fn from_path(path: &str) -> Self {
        match path {
            p if p.contains("Canary") => DiscordBranch::Canary,
            p if p.contains("Development") => DiscordBranch::Development,
            p if p.contains("PTB") => DiscordBranch::PTB,
            _ => DiscordBranch::Stable,
        }
    }
}

impl std::fmt::Display for DiscordBranch {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", match self {
            DiscordBranch::Canary => "Canary",
            DiscordBranch::Development => "Development",
            DiscordBranch::PTB => "PTB",
            DiscordBranch::Stable => "Stable"
        })
    }
}

#[derive(Debug, PartialEq, Clone)]
pub struct DiscordLocation {
    /// The name of the Discord installation, e.g. "Discord.app"
    pub name: String,
    /// The full path to the Discord installation
    pub path: String,
    /// The branch of the Discord installation
    pub branch: DiscordBranch,
    /// Whether the Discord installation has been patched with mod
    pub patched: bool,
    /// Whether the Discord installation has been patched with openasar
    pub openasar: bool,
    /// If its a flatpak installation: https://flatpak.org/
    pub is_flatpak: bool, 
    /// Arch package, needs special care: https://aur.archlinux.org/packages/discord_arch_electron
    pub is_system_electron: bool,
}