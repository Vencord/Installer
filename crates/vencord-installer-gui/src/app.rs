use std::{
    error::Error,
    rc::Rc,
    sync::{Arc, Mutex},
};
use tokio::sync::mpsc;

use vencord_installer_core::paths::{
    DiscordBranch as CoreDiscordBranch, DiscordLocation as CoreDiscordLocation,
    get_discord_locations,
};

use crate::operations::{AppActions, AppMessage, AppOperation};

slint::include_modules!();

type CustomLocations = Arc<Mutex<Vec<CoreDiscordLocation>>>;

struct CallbackCtx {
    tx: mpsc::UnboundedSender<AppOperation>,
    app_weak: slint::Weak<AppWindow>,
}

impl CallbackCtx {
    fn dispatch(&self, op: LoadingOp, path: slint::SharedString, app_op: AppOperation) {
        if let Some(app) = self.app_weak.upgrade() {
            let pm = app.global::<PageManager>();
            pm.set_loading_op(op);
            pm.set_loading_path(path);
        }
        self.tx.send(app_op).ok();
    }
}

#[derive(Clone, Copy)]
enum ErrorAction {
    OpenAppData,
    OpenAppManagement,
}

impl ErrorAction {
    fn label(self) -> &'static str {
        match self {
            Self::OpenAppData => "Take me There",
            Self::OpenAppManagement => "App Management",
        }
    }

    fn execute(self) {
        match self {
            #[cfg(target_os = "windows")]
            Self::OpenAppData => {
                if let Some(path) = vencord_installer_core::paths::get_program_data_path() {
                    open::that_in_background(path);
                }
            }
            #[cfg(target_os = "macos")]
            Self::OpenAppManagement => {
                open::that_in_background(
                    "x-apple.systempreferences:com.apple.preference.security?Privacy_AppBundles",
                );
            }
            _ => {}
        }
    }
}

pub struct VencordInstallerApp {
    app: AppWindow,
    app_weak: slint::Weak<AppWindow>,
    operation_tx: mpsc::UnboundedSender<AppOperation>,
    message_rx: Option<mpsc::UnboundedReceiver<AppMessage>>,
    custom_locations: CustomLocations,
}

impl VencordInstallerApp {
    pub async fn new() -> Result<Self, Box<dyn Error>> {
        let app = AppWindow::new()?;
        let app_weak = app.as_weak();

        let (operation_tx, operation_rx) = mpsc::unbounded_channel();
        let (message_tx, message_rx) = mpsc::unbounded_channel();

        tokio::spawn(AppActions::new(operation_rx, message_tx).run());

        let mut gui_app = Self {
            app,
            app_weak,
            operation_tx,
            message_rx: Some(message_rx),
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
        let ctx = Rc::new(CallbackCtx {
            tx: self.operation_tx.clone(),
            app_weak: self.app_weak.clone(),
        });

        let app_weak = self.app_weak.clone();
        let custom_locations = self.custom_locations.clone();
        callbacks.on_refresh_locations(move || {
            if let Some(app) = app_weak.upgrade() {
                Self::refresh_locations(&app, &custom_locations.lock().unwrap());
            }
        });

        let c = ctx.clone();
        callbacks.on_do_install(move |loc| {
            let loc: CoreDiscordLocation = (&loc).into();
            if !loc.is_vencord {
                c.dispatch(
                    LoadingOp::Install,
                    loc.path.to_string_lossy().as_ref().into(),
                    AppOperation::Install(loc),
                );
            }
        });

        let c = ctx.clone();
        callbacks.on_do_uninstall(move |loc| {
            let loc: CoreDiscordLocation = (&loc).into();
            if loc.is_vencord {
                c.dispatch(
                    LoadingOp::Uninstall,
                    loc.path.to_string_lossy().as_ref().into(),
                    AppOperation::Uninstall(loc),
                );
            }
        });

        let c = ctx.clone();
        callbacks.on_do_o_install(move |loc| {
            let loc: CoreDiscordLocation = (&loc).into();
            if !loc.is_openasar {
                c.dispatch(
                    LoadingOp::OpenAsarInstall,
                    loc.path.to_string_lossy().as_ref().into(),
                    AppOperation::InstallOpenAsar(loc),
                );
            }
        });

        let c = ctx.clone();
        callbacks.on_do_o_uninstall(move |loc| {
            let loc: CoreDiscordLocation = (&loc).into();
            if loc.is_openasar {
                c.dispatch(
                    LoadingOp::OpenAsarUninstall,
                    loc.path.to_string_lossy().as_ref().into(),
                    AppOperation::UninstallOpenAsar(loc),
                );
            }
        });

        let c = ctx.clone();
        callbacks.on_do_repair(move |loc| {
            c.dispatch(
                LoadingOp::Repair,
                loc.path.clone(),
                AppOperation::Repair((&loc).into()),
            );
        });

        let c = ctx.clone();
        callbacks.on_do_open_link(move |url| {
            c.tx.send(AppOperation::OpenLink(url.to_string())).ok();
        });

        let app_weak_folder = self.app_weak.clone();
        let custom_locations_folder = self.custom_locations.clone();
        callbacks.on_do_open_folder_dialog(move || {
            #[cfg(target_os = "macos")]
            let picked = rfd::FileDialog::new()
                .set_title("Select Discord Bundle")
                .pick_file();
            #[cfg(not(target_os = "macos"))]
            let picked = rfd::FileDialog::new()
                .set_title("Select Discord Installation Folder")
                .pick_folder();

            if let Some(path) = picked.as_ref().and_then(|p| p.to_str()) {
                if let Some(location) = CoreDiscordLocation::from_path(path) {
                    custom_locations_folder.lock().unwrap().push(location);
                    if let Some(app) = app_weak_folder.upgrade() {
                        VencordInstallerApp::refresh_locations(
                            &app,
                            &custom_locations_folder.lock().unwrap(),
                        );
                    }
                }
            }
        });
    }

    fn refresh_discord_locations(&self) {
        Self::refresh_locations(&self.app, &self.custom_locations.lock().unwrap());
    }

    fn refresh_locations(app: &AppWindow, custom_locations: &[CoreDiscordLocation]) {
        let mut seen = std::collections::HashSet::new();

        let locations: Vec<DiscordLocation> = get_discord_locations()
            .into_iter()
            .chain(custom_locations.iter().cloned())
            .filter(|loc| seen.insert(loc.path.clone()))
            .map(|loc| (&loc).into())
            .collect();

        app.global::<DiscordLocationAdapter>()
            .set_locations(std::rc::Rc::new(slint::VecModel::from(locations)).into());

        app.global::<PageManager>().set_current_page_index(0);
    }

    fn start_message_handler(&mut self) {
        let app_weak = self.app_weak.clone();
        let custom_locations = self.custom_locations.clone();
        let mut message_rx = self
            .message_rx
            .take()
            .expect("message handler already started");

        tokio::spawn(async move {
            while let Some(message) = message_rx.recv().await {
                Self::handle_message(message, &app_weak, custom_locations.clone());
            }
        });
    }

    fn handle_message(
        message: AppMessage,
        app_weak: &slint::Weak<AppWindow>,
        custom_locations: CustomLocations,
    ) {
        Self::invoke_ui_update(app_weak.clone(), move |app| {
            let pm = app.global::<PageManager>();
            pm.set_loading_op(LoadingOp::None);
            pm.set_loading_path("".into());
            Self::refresh_locations(app, &custom_locations.lock().unwrap());
        });

        if let AppMessage::OperationError {
            message: error,
            show_appdata,
            show_permission_help,
        } = message
        {
            Self::show_error_dialog(error, show_appdata, show_permission_help);
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

    fn show_error_dialog(error: String, show_appdata: bool, show_permission_help: bool) {
        let action: Option<ErrorAction> = if show_appdata {
            Some(ErrorAction::OpenAppData)
        } else if show_permission_help {
            Some(ErrorAction::OpenAppManagement)
        } else {
            None
        };

        let buttons = match action {
            Some(a) => rfd::MessageButtons::OkCancelCustom(a.label().to_owned(), "Ok".to_owned()),
            None => rfd::MessageButtons::Ok,
        };

        let result = rfd::MessageDialog::new()
            .set_title("Operation Failed")
            .set_description(&error)
            .set_buttons(buttons)
            .set_level(rfd::MessageLevel::Error)
            .show();

        if let Some(action) = action {
            if result == rfd::MessageDialogResult::Custom(action.label().to_owned()) {
                action.execute();
            }
        }
    }
}

// MARK: - Type conversions

impl From<&CoreDiscordLocation> for DiscordLocation {
    fn from(core: &CoreDiscordLocation) -> Self {
        Self {
            path: core.path.to_string_lossy().as_ref().into(),
            branch: (&core.branch).into(),
            is_vencord: core.is_vencord,
            is_openasar: core.is_openasar,
            is_flatpak: core.is_flatpak,
            is_scuffed: core.is_scuffed,
            is_system_electron: core.is_system_electron,
        }
    }
}

impl From<&DiscordLocation> for CoreDiscordLocation {
    fn from(loc: &DiscordLocation) -> Self {
        Self {
            path: std::path::PathBuf::from(loc.path.to_string()),
            branch: (&loc.branch).into(),
            is_vencord: loc.is_vencord,
            is_openasar: loc.is_openasar,
            is_flatpak: loc.is_flatpak,
            is_scuffed: loc.is_scuffed,
            is_system_electron: loc.is_system_electron,
        }
    }
}

impl From<&CoreDiscordBranch> for DiscordBranch {
    fn from(b: &CoreDiscordBranch) -> Self {
        match b {
            CoreDiscordBranch::Stable => Self::Stable,
            CoreDiscordBranch::PTB => Self::PTB,
            CoreDiscordBranch::Canary => Self::Canary,
            CoreDiscordBranch::Development => Self::Development,
        }
    }
}

impl From<&DiscordBranch> for CoreDiscordBranch {
    fn from(b: &DiscordBranch) -> Self {
        match b {
            DiscordBranch::Stable => Self::Stable,
            DiscordBranch::PTB => Self::PTB,
            DiscordBranch::Canary => Self::Canary,
            DiscordBranch::Development => Self::Development,
        }
    }
}
