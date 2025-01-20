use std::path::{Path, PathBuf};

use crate::handler::handler::InstallerResult;

/// Prepares a dist directory by downloading the specified files, including a creating a `package.json` file.
/// 
/// # Arguments
/// 
/// * `dir_name` - The name of your config directory.
/// * `url_path` - The URL path to download the files from.
/// * `user_agent` - The user agent to use for the request.
/// * `dist_files` - The list of files to download from the url path.
/// 
/// # Returns
/// 
/// Returns `Ok(())` if the directory was prepared successfully, otherwise an InstallerResult error.
pub async fn prepare_dist_directory<P: AsRef<Path>, S: Into<String>, I: IntoIterator<Item = String>>(
    path: P,
    url_path: S,
    user_agent: S,
    dist_files: I,
) -> InstallerResult<()> {
    let url_path = url_path.into();
    let user_agent = user_agent.into();

    // Use the provided path directly instead of constructing from dir_name
    let downloads_dist_path = path.as_ref();
    let package_path = downloads_dist_path.join("package.json");

    tokio::fs::write(package_path, r#"{}"#).await?;

    for file in dist_files {
        download_file(
            &format!("{}/{}", url_path, file),
            downloads_dist_path.join(&file),
            &user_agent
        ).await?;
    }

    Ok(())
}

/// Downloads a file from a given URL and saves it to a given path.
/// 
/// # Arguments
/// 
/// * `url` - The URL to download the file from.
/// * `path` - The path to save the downloaded file to.
/// * `user_agent` - The user agent to use for the request.
/// 
/// # Returns
/// 
/// Returns `Ok(())` if the file was downloaded successfully, otherwise an InstallerResult error.
pub async fn download_file(url: &str, path: PathBuf, user_agent: &str) -> InstallerResult<()> {
    let client = reqwest::Client::new();

    let response = client.get(url).header("User-Agent", user_agent).send().await?;

    let mut dest = tokio::fs::File::create(&path).await?;

    tokio::io::copy(&mut response.bytes().await?.as_ref(), &mut dest).await?;
        
    Ok(())
}
