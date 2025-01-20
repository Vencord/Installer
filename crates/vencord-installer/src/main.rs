#![allow(rustdoc::missing_crate_level_docs)]

#[cfg(any(target_os = "macos", target_os = "linux"))]
use {
    std::process::exit,
    logger_rs::error
};

use vencord_installer::cli::arguments;

#[cfg(any(target_os = "macos", target_os = "linux"))]
extern "C" {
    fn geteuid() -> u32;
}

fn main() {
    // call this when theres 1 or more arguments being passed when doing the gui
    cli();
}

fn cli() {
    let matches = arguments::args_build().get_matches();

    // Linux needs root, mainly for places that are containerized
    #[cfg(not(any(target_os = "windows", target_os = "macos")))]
    if unsafe { geteuid() } != 0 {
        error!("Please run this program using `sudo -E`!");
        exit(1);
    }

    // macOS don't need root
    #[cfg(any(target_os = "macos"))]
    if unsafe { geteuid() } == 0 {
        error!("Please run this program without root!");
        exit(1);
    }

    arguments::arg_conflicts(&matches);
    arguments::arg_commands(&matches);
}