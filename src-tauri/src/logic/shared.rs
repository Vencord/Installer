use serde::Serialize;

#[derive(Serialize, Clone, Eq, PartialEq, PartialOrd, Ord)]
pub enum DiscordBranch {
    Stable,
    Canary,
    PTB,
}

impl DiscordBranch {
    pub fn from_filename(s: &str) -> Self {
        let lower = s.to_lowercase();

        if lower.ends_with("ptb") {
            return DiscordBranch::PTB;
        } else if lower.ends_with("canary") {
            return DiscordBranch::Canary;
        } else {
            return DiscordBranch::Stable;
        }
    }
}

#[derive(Serialize)]
pub struct DiscordInstall {
    pub path: String,
    pub app_path: Option<String>,
    pub branch: DiscordBranch,
    pub is_patched: bool,
    pub is_flatpak: bool,
}
