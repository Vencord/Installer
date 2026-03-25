use std::collections::HashMap;
use std::path::Path;

use serde::Serialize;
use tokio::{fs::File, io::AsyncWriteExt};

use crate::Error;

#[derive(Serialize)]
struct AsarEntry {
    size: i32,
    offset: String,
}

pub(crate) async fn write_app_asar(path: &Path, entries: &[(&str, &String)]) -> Result<(), Error> {
    let files = make_asar_files(entries);

    let header = serde_json::to_string(&HashMap::from([("files".to_string(), files)]))?;
    let aligned_size = (header.len() as u32 + 3) & !3;

    let mut file = File::create(path).await?;

    for size in [
        4u32,
        aligned_size + 8,
        aligned_size + 4,
        header.len() as u32,
    ] {
        file.write_all(&(size as i32).to_le_bytes()).await?;
    }

    file.write_all(format!("{:<width$}", header, width = aligned_size as usize).as_bytes())
        .await?;

    for (_, content) in entries {
        file.write_all(content.as_bytes()).await?;
    }

    Ok(())
}

fn make_asar_files(entries: &[(&str, &String)]) -> HashMap<String, AsarEntry> {
    let mut files = HashMap::new();
    let mut current_offset = 0;

    for (name, content) in entries {
        let size = content.len() as i32;
        files.insert(
            name.to_string(),
            AsarEntry {
                size,
                offset: current_offset.to_string(),
            },
        );
        current_offset += size as usize;
    }

    files
}
