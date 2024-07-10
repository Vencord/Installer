pub mod shared;

#[cfg(target_os = "linux")]
mod linux;

#[cfg(target_os = "linux")]
pub use linux::discord;

#[cfg(windows)]
mod windows;

#[cfg(windows)]
pub use super::windows::discord;
