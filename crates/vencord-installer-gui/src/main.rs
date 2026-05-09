#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod app;
mod operations;

use std::error::Error;

#[tokio::main]
async fn main() -> Result<(), Box<dyn Error>> {
    env_logger::Builder::from_env(env_logger::Env::default().default_filter_or("debug")).init();

    // force the renderer to software
    // it should work on most systems
    unsafe { 
        std::env::set_var("SLINT_BACKEND", "software"); 
    }

    let app = app::VencordInstallerApp::new().await?;
    app.run().map_err(Into::into)
}
