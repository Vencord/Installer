use std::path::Path;

use serde_json::Value;
use reqwest::Client;

/// Try to get JSON from a URL, return None if it fails.
/// 
/// # Arguments
/// 
/// * `client` - The reqwest client to use for the request.
/// * `url` - The URL to get the JSON from.
/// * `user_agent` - The user agent to use for the request.
/// 
/// # Returns
/// 
/// Returns the JSON value if the request was successful, otherwise None.
async fn try_get_json(client: &Client, url: &str, user_agent: &str) -> Option<Value> {
    match client
        .get(url)
        .header("User-Agent", user_agent)
        .send()
        .await {
            Ok(response) => response.json().await.ok(),
            Err(_) => None
    }
}

/// Check the latest version from a URL, if it fails, try the fallback URL.
/// 
/// # Arguments
/// 
/// * `url` - The URL to get the latest version from.
/// * `fallback_url` - The fallback URL to get the latest version from if the first URL fails.
/// * `user_agent` - The user agent to use for the request.
/// 
/// # Returns
/// 
/// Returns the JSON value if the request was successful, otherwise None.
pub async fn check_latest_version(url: &str, fallback_url: Option<&str>, user_agent: &str) -> Option<Value> {
    let client = Client::new();
    
    if let Some(result) = try_get_json(&client, url, user_agent).await {
        Some(result)
    } else if let Some(fallback) = fallback_url {
        try_get_json(&client, fallback, user_agent).await
    } else {
        None
    }
}

/// Get name from Github api, then seperate the hash from the title. For example, "DevBuild 48a9aef".
/// 
/// # Arguments
/// 
/// * `url` - The URL to get the latest version from.
/// * `fallback_url` - The fallback URL to get the latest version from if the first URL fails.
/// * `user_agent` - The user agent to use for the request.
/// 
/// # Returns
/// 
/// Returns the hash if the request was successful, otherwise None.
pub async fn check_hash_from_release(url: &str, fallback_url: Option<&str>, user_agent: &str) -> Option<String> {
    if let Some(json) = check_latest_version(url, fallback_url, user_agent).await {
        let name = json["name"].as_str().unwrap();
        let hash = name.split_whitespace().last().unwrap();
        Some(hash.to_owned())
    } else {
        None
    }
}

// r"// Vencord (\w+)" example
// this needs application support path to dist directory, or a asar file
pub async fn check_local_version(dir: &Path, regex: &str) -> Option<String> {
    let main_js = if dir.is_dir() {
        dir.join("preload.js")
    } else {
        dir.to_path_buf()
    };
    
    let main_js = tokio::fs::read_to_string(main_js).await.ok()?;
    let re = regex::Regex::new(regex).ok()?;
    let captures = re.captures(&main_js)?;
    let version = captures.get(1)?.as_str();

    Some(version.to_owned())
}
