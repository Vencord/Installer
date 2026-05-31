use tokio::sync::mpsc;

use vencord_installer_core::{
    Error, download,
    patch::Installer,
    paths::{DiscordLocation, get_data_path},
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
    OperationError {
        message: String,
        show_appdata: bool,
        show_permission_help: bool,
    },
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
                Err(err) => AppMessage::OperationError {
                    message: err.format_error(),
                    show_appdata: err.is_windows_moved_dir(),
                    show_permission_help: err.is_permission_denied(),
                },
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
                open::that(url).map_err(Error::ErrIo)?;
                Ok(())
            }
        }
    }

    async fn install(location: DiscordLocation) -> Result<(), Error> {
        if std::env::var("VENCORD_DEV_INSTALL").map_or(true, |v| v != "1") {
            download().await?;
        }

        Installer::new(location, get_data_path())?.patch().await
    }

    async fn uninstall(location: DiscordLocation) -> Result<(), Error> {
        Installer::new(location, get_data_path())?.unpatch().await
    }

    async fn repair(location: DiscordLocation) -> Result<(), Error> {
        let is_vencord = location.is_vencord;
        let is_openasar = location.is_openasar;
        let mut installer = Installer::new(location, get_data_path())?;
        installer.repair().await?;

        if is_vencord {
            installer.patch().await?;
        }

        if is_openasar {
            installer.patch_openasar().await?;
        }

        Ok(())
    }

    async fn install_openasar(location: DiscordLocation) -> Result<(), Error> {
        Installer::new(location, get_data_path())?
            .patch_openasar()
            .await
    }

    async fn uninstall_openasar(location: DiscordLocation) -> Result<(), Error> {
        Installer::new(location, get_data_path())?
            .unpatch_openasar()
            .await
    }
}
