#![warn(mismatched_lifetime_syntaxes)]
#![warn(clippy::pedantic)]
#![allow(clippy::module_name_repetitions)]

pub mod dl;
mod error;
pub mod patch;
pub mod paths;

pub use error::Error;
