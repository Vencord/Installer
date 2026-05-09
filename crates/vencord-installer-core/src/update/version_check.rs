use std::path::Path;

use reqwest::Client;
use serde_json::Value;

use crate::{Error, USER_AGENT};

/// Try to get JSON from a URL, return None if it fails.
async fn try_get_json(client: &Client, url: &str) -> Result<Value, Error> {
    let json = client
        .get(url)
        .header("User-Agent", USER_AGENT)
        .send()
        .await?
        .json()
        .await?;

    Ok(json)
}

/// Check the latest version from a URL, if it fails, try the fallback URL.
pub async fn check_latest_version(url: &str, fallback_url: &str) -> Result<Value, Error> {
    let client = Client::new();

    if let Ok(result) = try_get_json(&client, url).await {
        return Ok(result);
    }

    try_get_json(&client, fallback_url).await
}

/// Get name from Github api, then seperate the hash from the title. For example, "DevBuild 48a9aef".
pub async fn check_hash_from_release(url: &str, fallback_url: &str) -> Result<String, Error> {
    let json = check_latest_version(url, fallback_url).await?;
    let name = json["name"]
        .as_str()
        .ok_or(Error::ErrOther("Failed to get name from JSON"))?;
    let hash = name
        .split_whitespace()
        .last()
        .ok_or(Error::ErrOther("Failed to get hash from name"))?;
    log::info!("Found hash from release: {}", hash);
    Ok(hash.to_owned())
}

/// Check the local version from a directory by reading the preload.js file.
pub async fn check_local_version(dir: &Path) -> Result<String, Error> {
    let regex = r"// Vencord ([0-9a-zA-Z\.-]+)";

    let main_js = if dir.is_dir() {
        dir.join("preload.js")
    } else {
        dir.to_path_buf()
    };

    let main_js = tokio::fs::read_to_string(main_js).await?;
    let re = regex::Regex::new(regex)?;
    let captures = re
        .captures(&main_js)
        .ok_or(Error::ErrOther("Failed to capture regex"))?;
    let version = captures
        .get(1)
        .ok_or(Error::ErrOther("Failed to get regex group"))?
        .as_str();

    log::info!("Found local hash from preload.js: {}", version);

    Ok(version.to_owned())
}
