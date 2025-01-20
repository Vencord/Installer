use std::io::{self, Write};
use std::process::exit;

use colored::{ColoredString, Colorize};
use libvencord::paths::{
    branch::{DiscordBranch, DiscordLocation},
    locations::get_discord_locations
};
use logger_rs::{error, info, input};

// sigh
fn make_colored_branch_string(branch: &DiscordBranch) -> ColoredString {
    match branch {
        DiscordBranch::Stable => "Stable".to_string().truecolor(93, 107, 243).bold(),
        DiscordBranch::PTB => "PTB".to_string().truecolor(67, 150, 226).bold(),
        DiscordBranch::Canary => "Canary".to_string().truecolor(251, 183, 71).bold(),
        DiscordBranch::Development => "Development".to_string().bold()
    }
}

pub fn select_location(prompt: &str) -> DiscordLocation {
    info!("{}", prompt);
    let locations = get_discord_locations().unwrap_or_default();

    let filtered_locations: Vec<DiscordLocation> = locations
        .into_iter()
        .collect();
    
    if filtered_locations.is_empty() {
        error!("No matching Discord locations found!");
        exit(1);
    }
    
    for (index, location) in filtered_locations.iter().enumerate() {
        let mut instance = Vec::new();
        instance.push(make_colored_branch_string(&location.branch));
        if location.is_flatpak { instance.push("Flatpak".to_string().white().bold()); }

        let mut tags = Vec::new();
        if location.patched { tags.push("Vencord".to_string().truecolor(255, 192, 203).bold()); }
        if location.openasar { tags.push("OpenAsar".to_string().white().bold()); }

        let tags_str = if !tags.is_empty() {
            format!("+ {}", tags.iter().map(|s| s.to_string()).collect::<Vec<String>>().join(", "))
        } else {
            String::new()
        };

        println!("{}: {} {}\n└── {}", 
            index,
            instance.iter().map(|s| s.to_string()).collect::<Vec<String>>().join(", "),
            tags_str,
            location.path.to_string().truecolor(168, 168, 168)
        );
    }

    input!("Enter your choice [press Enter for default: 0]: ");
    io::stdout().flush().unwrap();

    let mut input = String::new();
    io::stdin().read_line(&mut input).unwrap();
    let choice: usize = loop {
        if input.trim().is_empty() {
            break 0;
        } else if let Ok(num) = input.trim().parse() {
            break num;
        } else {
            error!("You've chose something else thats not a number, please fix that!");
            input!("Enter your choice [press Enter for default: 0]: ");
            io::stdout().flush().unwrap();
            input.clear();
            io::stdin().read_line(&mut input).unwrap();
        }
    };

    if let Some(selected_location) = filtered_locations.get(choice) {
        selected_location.clone().clone()
    } else {
        error!("You can't choose something that doesn't exist! :c");
        exit(1);
    }
}