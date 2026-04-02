use std::{
    error::Error,
    sync::{Arc, Mutex},
};
use tokio::sync::mpsc;

use vencord_installer_core::paths::{
    branch::{DiscordBranch as CoreDiscordBranch, DiscordLocation as CoreDiscordLocation},
    locations::{get_discord_locations, get_program_data_path},
    shared::get_custom_discord_location,
};

use crate::operations::{AppActions, AppMessage, AppOperation};

slint::include_modules!();

pub struct VencordInstallerApp {
    app: AppWindow,
    app_weak: slint::Weak<AppWindow>,
    operation_tx: mpsc::UnboundedSender<AppOperation>,
    message_rx: mpsc::UnboundedReceiver<AppMessage>,
    custom_locations: Arc<Mutex<Vec<CoreDiscordLocation>>>,
}

impl VencordInstallerApp {
    pub async fn new() -> Result<Self, Box<dyn Error>> {
        let app = AppWindow::new()?;
        let app_weak = app.as_weak();

        let (operation_tx, operation_rx) = mpsc::unbounded_channel();
        let (message_tx, message_rx) = mpsc::unbounded_channel();

        let actions = AppActions::new(operation_rx, message_tx);
        tokio::spawn(actions.run());

        let mut gui_app = Self {
            app,
            app_weak: app_weak.clone(),
            operation_tx,
            message_rx,
            custom_locations: Arc::new(Mutex::new(Vec::new())),
        };

        gui_app.initialize().await?;
        gui_app.start_message_handler();
        Ok(gui_app)
    }

    pub fn run(self) -> Result<(), slint::PlatformError> {
        self.app.run()
    }

    async fn initialize(&self) -> Result<(), Box<dyn Error>> {
        self.app
            .global::<AppInfo>()
            .set_version(env!("CARGO_PKG_VERSION").into());

        self.setup_callbacks();
        self.refresh_discord_locations();
        Ok(())
    }

    fn setup_callbacks(&self) {
        let callbacks = self.app.global::<RustCallbacks>();
        let app_weak = self.app_weak.clone();
        let custom_locations = self.custom_locations.clone();
        let tx = self.operation_tx.clone();

        callbacks.on_refresh_locations({
            let app_weak = app_weak.clone();
            let custom_locations = custom_locations.clone();
            move || {
                if let Some(app) = app_weak.upgrade() {
                    Self::refresh_locations(&app, &custom_locations.lock().unwrap());
                }
            }
        });

        let tx_install = tx.clone();
        callbacks.on_do_install(move |loc| {
            let loc: CoreDiscordLocation = (&loc).into();
            if !loc.patched {
                tx_install.send(AppOperation::Install(loc)).ok();
            }
        });

        let tx_uninstall = tx.clone();
        callbacks.on_do_uninstall(move |loc| {
            let loc: CoreDiscordLocation = (&loc).into();
            if loc.patched {
                tx_uninstall.send(AppOperation::Uninstall(loc)).ok();
            }
        });

        let tx_o_install = tx.clone();
        callbacks.on_do_o_install(move |loc| {
            let loc: CoreDiscordLocation = (&loc).into();
            if !loc.openasar {
                tx_o_install.send(AppOperation::InstallOpenAsar(loc)).ok();
            }
        });

        let tx_o_uninstall = tx.clone();
        callbacks.on_do_o_uninstall(move |loc| {
            let loc: CoreDiscordLocation = (&loc).into();
            if loc.openasar {
                tx_o_uninstall
                    .send(AppOperation::UninstallOpenAsar(loc))
                    .ok();
            }
        });

        let tx_repair = tx.clone();
        callbacks.on_do_repair(move |loc| {
            tx_repair.send(AppOperation::Repair((&loc).into())).ok();
        });

        let tx_open_link = tx.clone();
        callbacks.on_do_open_link(move |url| {
            tx_open_link
                .send(AppOperation::OpenLink(url.to_string()))
                .ok();
        });

        let app_weak_folder = self.app_weak.clone();
        let custom_locations_folder = self.custom_locations.clone();
        callbacks.on_do_open_folder_dialog(move || {
            #[cfg(target_os = "macos")]
            let dialog_result = rfd::FileDialog::new()
                .set_title("Select Discord Bundle")
                .pick_file();

            #[cfg(not(target_os = "macos"))]
            let dialog_result = rfd::FileDialog::new()
                .set_title("Select Discord Installation Folder")
                .pick_folder();

            if let Some(selected) = dialog_result {
                if let Some(path) = selected.to_str() {
                    if let Some(location) = get_custom_discord_location(&path) {
                        custom_locations_folder.lock().unwrap().push(location);

                        if let Some(app) = app_weak_folder.upgrade() {
                            VencordInstallerApp::refresh_locations(
                                &app,
                                &custom_locations_folder.lock().unwrap(),
                            );
                        }
                    }
                }
            }
        });
    }

    fn refresh_discord_locations(&self) {
        Self::refresh_locations(&self.app, &self.custom_locations.lock().unwrap());
    }

    fn refresh_locations(app: &AppWindow, custom_locations: &[CoreDiscordLocation]) {
        let mut all_locations: Vec<DiscordLocation> = Vec::new();
        let mut seen_paths = std::collections::HashSet::new();

        if let Some(core_locations) = get_discord_locations() {
            for loc in &core_locations {
                if seen_paths.insert(loc.path.clone()) {
                    all_locations.push(loc.into());
                }
            }
        }

        for loc in custom_locations {
            if seen_paths.insert(loc.path.clone()) {
                all_locations.push(loc.into());
            }
        }

        let locations_model = std::rc::Rc::new(slint::VecModel::from(all_locations));
        app.global::<DiscordLocationAdapter>()
            .set_locations(locations_model.into());

        app.global::<PageManager>().set_current_page_index(0);
    }

    fn start_message_handler(&mut self) {
        let app_weak = self.app_weak.clone();
        let custom_locations = self.custom_locations.clone();
        let mut message_rx = std::mem::replace(&mut self.message_rx, mpsc::unbounded_channel().1);

        tokio::spawn(async move {
            while let Some(message) = message_rx.recv().await {
                VencordInstallerApp::handle_message(message, &app_weak, custom_locations.clone());
            }
        });
    }

    fn handle_message(
        message: AppMessage,
        app_weak: &slint::Weak<AppWindow>,
        custom_locations: Arc<Mutex<Vec<CoreDiscordLocation>>>,
    ) {
        Self::invoke_ui_update(app_weak.clone(), move |app| {
            Self::refresh_locations(app, &custom_locations.lock().unwrap());
        });

        if let AppMessage::OperationError(error, show_open_appdata) = message {
            Self::show_error_dialog(error, show_open_appdata);
        }
    }

    fn invoke_ui_update<F>(app_weak: slint::Weak<AppWindow>, f: F)
    where
        F: FnOnce(&AppWindow) + Send + 'static,
    {
        slint::invoke_from_event_loop(move || {
            if let Some(app) = app_weak.upgrade() {
                f(&app);
            }
        })
        .ok();
    }

    fn show_error_dialog(error: String, show_open_appdata: bool) {
        let result = rfd::MessageDialog::new()
            .set_title("Operation Failed")
            .set_description(&error)
            .set_buttons(if show_open_appdata {
                rfd::MessageButtons::OkCancelCustom("Take me There".to_owned(), "Ok".to_owned())
            } else {
                rfd::MessageButtons::Ok
            })
            .set_level(rfd::MessageLevel::Error)
            .show();

        if show_open_appdata
            && result == rfd::MessageDialogResult::Custom("Take me There".to_owned())
        {
            open::that_in_background(get_program_data_path());
        }
    }
}

// MARK: - Type conversions
impl From<&CoreDiscordLocation> for DiscordLocation {
    fn from(core: &CoreDiscordLocation) -> Self {
        Self {
            name: core.name.clone().into(),
            path: core.path.clone().into(),
            branch: convert_branch_to_slint(&core.branch),
            patched: core.patched,
            openasar: core.openasar,
            is_flatpak: core.is_flatpak,
            is_system_electron: core.is_system_electron,
        }
    }
}

impl From<&DiscordLocation> for CoreDiscordLocation {
    fn from(slint_location: &DiscordLocation) -> Self {
        Self {
            name: slint_location.name.to_string(),
            path: slint_location.path.to_string(),
            branch: convert_branch_to_core(&slint_location.branch),
            patched: slint_location.patched,
            openasar: slint_location.openasar,
            is_flatpak: slint_location.is_flatpak,
            is_system_electron: slint_location.is_system_electron,
        }
    }
}

fn convert_branch_to_slint(core_branch: &CoreDiscordBranch) -> DiscordBranch {
    match core_branch {
        CoreDiscordBranch::Stable => DiscordBranch::Stable,
        CoreDiscordBranch::PTB => DiscordBranch::PTB,
        CoreDiscordBranch::Canary => DiscordBranch::Canary,
        CoreDiscordBranch::Development => DiscordBranch::Development,
    }
}

fn convert_branch_to_core(slint_branch: &DiscordBranch) -> CoreDiscordBranch {
    match slint_branch {
        DiscordBranch::Stable => CoreDiscordBranch::Stable,
        DiscordBranch::PTB => CoreDiscordBranch::PTB,
        DiscordBranch::Canary => CoreDiscordBranch::Canary,
        DiscordBranch::Development => CoreDiscordBranch::Development,
    }
}
