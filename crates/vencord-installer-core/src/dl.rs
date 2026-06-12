use std::path::PathBuf;

use regex::Regex;
use reqwest::Client;
use tokio::{fs, io};

use crate::Error;

use crate::paths::get_data_path;

static USER_AGENT: &str = "VencordInstaller (https://github.com/Vencord/Installer)";
static RELEASE_URL: &str = "https://api.github.com/repos/Vendicated/Vencord/releases/latest";
static RELEASE_URL_FALLBACK: &str = "https://vencord.dev/releases/vencord";
static RELEASE_TAG_DOWNLOAD: &str =
    "https://github.com/Vendicated/Vencord/releases/download/devbuild";
pub(crate) static OPENASAR_URL: &str =
    "https://github.com/GooseMod/OpenAsar/releases/download/nightly/app.asar";

static RELEASE_LOADER: &str = "patcher.js";
static RELEASE_MAIN: &str = "preload.js";
static RELEASE_PACKAGE_JSON: &str = "package.json";

static RELEASE_ASSETS: &[&str] = &[
    "patcher.js",
    "patcher.js.map",
    "patcher.js.LEGAL.txt",
    "preload.js",
    "preload.js.map",
    "renderer.js",
    "renderer.js.map",
    "renderer.js.LEGAL.txt",
    "renderer.css",
    "renderer.css.map",
];

static RELEASE_HEADER_REGEX: &str = r"// Vencord ([0-9a-zA-Z\.-]+)";

pub async fn download_latest_assets() -> Result<(), Error> {
    let client = Client::new();

    let data_path = get_data_path().ok_or(Error::ErrNoDataPath)?;

    if compare_versions(&client, &data_path).await? {
        // node likes to recursively search for package.json, so just create an empty one
        fs::write(data_path.join(RELEASE_PACKAGE_JSON), r#"{}"#).await?;

        for file in RELEASE_ASSETS {
            download_file(
                &client,
                &format!("{RELEASE_TAG_DOWNLOAD}/{file}"),
                data_path.join(file),
            )
            .await?;
        }
    }

    Ok(())
}

pub async fn download_openasar() -> Result<(), Error> {
    let client = Client::new();

    let data_path = get_data_path().ok_or(Error::ErrNoDataPath)?;

    download_file(&client, OPENASAR_URL, data_path.join("open.asar")).await?;

    Ok(())
}

pub async fn download_file(client: &Client, url: &str, path: PathBuf) -> Result<(), Error> {
    let response = client
        .get(url)
        .header("User-Agent", USER_AGENT)
        .send()
        .await?;

    let mut dest = fs::File::create(&path).await?;

    io::copy(&mut response.bytes().await?.as_ref(), &mut dest).await?;

    Ok(())
}

#[derive(serde::Deserialize)]
struct Release {
    name: String,
}

async fn compare_versions(client: &Client, data_path: &PathBuf) -> Result<bool, Error> {
    let main_js_asset = data_path.join(RELEASE_MAIN);

    if !main_js_asset.exists() {
        return Ok(true);
    }

    let main_js = fs::read_to_string(main_js_asset).await?;

    let Some(local_version) = Regex::new(RELEASE_HEADER_REGEX)?
        .captures(&main_js)
        .and_then(|c| c.get(1))
        .map(|m| m.as_str())
    else {
        return Ok(true);
    };

    let release: Release = client
        .get(RELEASE_URL)
        .header("User-Agent", USER_AGENT)
        .send()
        .await?
        .json()
        .await?;

    let latest_version = release
        .name
        .split_whitespace()
        .last()
        .ok_or(Error::ErrOther("Failed to extract latest version"))?;

    Ok(local_version != latest_version)
}
