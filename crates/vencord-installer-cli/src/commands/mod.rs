pub mod patch;

use clap::Parser;

#[derive(Debug, Parser)]
#[command(
    name = "Vencord Installer CLI",
    author,
    version,
    about = "Command-line tool for installing Vencord and OpenAsar to Discord.",
    disable_help_subcommand = true
)]
pub struct Cli {
    #[command(flatten)]
    pub args: patch::PatchArgs,
}
