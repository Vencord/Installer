#[cfg(not(feature = "gui"))]
pub mod cli;

pub const RELEASE_URL: &str = "https://api.github.com/repos/Vendicated/Vencord/releases/latest";
pub const RELEASE_URL_FALLBACK: &str = "https://vencord.dev/releases/vencord";
pub const RELEASE_TAG_DOWNLOAD: &str = "https://github.com/Vendicated/Vencord/releases/download/devbuild";
pub const OPENASAR_URL: &str = "https://github.com/GooseMod/OpenAsar/releases/download/nightly/app.asar";
pub const USER_AGENT: &str = "VencordInstaller (https://github.com/Vencord/Installer)";