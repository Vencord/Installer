#![allow(rustdoc::missing_crate_level_docs)]

mod commands;

use clap::Parser;
use commands::Cli;
use vencord_installer_core::Error;

#[cfg(any(target_os = "macos"))]
unsafe extern "C" {
    unsafe fn geteuid() -> u32;
}

#[tokio::main]
async fn main() -> Result<(), Error> {
    env_logger::Builder::from_env(env_logger::Env::default().default_filter_or("info")).init();

    // macOS doesn't need root permissions
    #[cfg(any(target_os = "macos"))]
    if unsafe { geteuid() } == 0 {
        return Err(Error::ErrInvalidArguments(
            "Please run this program without root, and make sure your terminal has Developer Tool permissions and Full Disk Access!",
        ));
    }

    let cli = Cli::parse();
    commands::patch::execute(cli.args).await?;

    Ok(())
}
