use std::path::Path;

/// The branch of a Discord installation
#[derive(Debug, PartialEq, Clone)]
pub enum DiscordBranch {
    Stable,
    PTB,
    Canary,
    Development,
}

impl DiscordBranch {
    /// Returns the DiscordBranch from the given path.
    pub fn from_path<P: AsRef<Path>>(path: P) -> Self {
        let path_str = path
            .as_ref()
            .to_str()
            .unwrap_or_default()
            .to_ascii_lowercase();

        match path_str.as_str() {
            p if p.contains("canary") => DiscordBranch::Canary,
            p if p.contains("development") => DiscordBranch::Development,
            p if p.contains("ptb") => DiscordBranch::PTB,
            _ => DiscordBranch::Stable,
        }
    }
}

impl Default for DiscordBranch {
    fn default() -> Self {
        DiscordBranch::Stable
    }
}

impl std::fmt::Display for DiscordBranch {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(
            f,
            "{}",
            match self {
                DiscordBranch::Canary => "Discord Canary",
                DiscordBranch::Development => "Discord Development",
                DiscordBranch::PTB => "Discord PTB",
                DiscordBranch::Stable => "Discord",
            }
        )
    }
}
