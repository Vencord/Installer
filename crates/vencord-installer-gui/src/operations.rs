use tokio::sync::mpsc;

use vencord_installer_core::{
    Error, OPENASAR_URL, download, get_dist_path, patch::patch_mod::Installer,
    paths::branch::DiscordLocation,
};

#[derive(Debug, Clone)]
pub enum AppOperation {
    Install(DiscordLocation),
    Uninstall(DiscordLocation),
    Repair(DiscordLocation),
    InstallOpenAsar(DiscordLocation),
    UninstallOpenAsar(DiscordLocation),
    OpenLink(String),
}

#[derive(Debug, Clone)]
pub enum AppMessage {
    OperationSuccess,
    OperationError(String, bool, bool),
}

pub struct AppActions {
    operation_rx: mpsc::UnboundedReceiver<AppOperation>,
    message_tx: mpsc::UnboundedSender<AppMessage>,
}

impl AppActions {
    pub fn new(
        operation_rx: mpsc::UnboundedReceiver<AppOperation>,
        message_tx: mpsc::UnboundedSender<AppMessage>,
    ) -> Self {
        Self {
            operation_rx,
            message_tx,
        }
    }

    pub async fn run(mut self) {
        while let Some(operation) = self.operation_rx.recv().await {
            let message = match self.handle_operation(operation).await {
                Ok(()) => AppMessage::OperationSuccess,
                Err(err) => AppMessage::OperationError(
                    err.format_error(),
                    err.is_windows_moved_dir(),
                    err.is_permission_denied(),
                ),
            };

            let _ = self.message_tx.send(message);
        }
    }

    async fn handle_operation(&self, operation: AppOperation) -> Result<(), Error> {
        match operation {
            AppOperation::Install(location) => Self::install(location).await,
            AppOperation::Uninstall(location) => Self::uninstall(location).await,
            AppOperation::Repair(location) => Self::repair(location).await,
            AppOperation::InstallOpenAsar(location) => Self::install_openasar(location).await,
            AppOperation::UninstallOpenAsar(location) => Self::uninstall_openasar(location).await,
            AppOperation::OpenLink(url) => {
                open::that(url).map_err(|e| Error::ErrIo(e))?;
                Ok(())
            }
        }
    }

    async fn install(location: DiscordLocation) -> Result<(), Error> {
        if std::env::var("VENCORD_DEV_INSTALL").map_or(true, |v| v != "1") {
            download().await?;
        }

        Installer::new(location.clone(), Some(get_dist_path(None)))
            .patch()
            .await
    }

    async fn uninstall(location: DiscordLocation) -> Result<(), Error> {
        Installer::new(location.clone(), None).unpatch().await
    }

    async fn repair(location: DiscordLocation) -> Result<(), Error> {
        let mut installer = Installer::new(location.clone(), Some(get_dist_path(None)));
        installer.repair().await?;

        if location.patched {
            installer.patch().await?;
        }

        if location.openasar {
            installer.patch_openasar(OPENASAR_URL).await?;
        }

        Ok(())
    }

    async fn install_openasar(location: DiscordLocation) -> Result<(), Error> {
        Installer::new(location, Some(get_dist_path(None)))
            .patch_openasar(OPENASAR_URL)
            .await
    }

    async fn uninstall_openasar(location: DiscordLocation) -> Result<(), Error> {
        Installer::new(location, None).unpatch_openasar().await
    }
}
