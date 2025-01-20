# libvencord

Various utilities made for [Vencord Installer](https://github.com/Vencord/Installer), which includes utilities to patch and unpatch mods for Discord. 

Including support for installing [OpenAsar](https://github.com/GooseMod/OpenAsar).

### Supported environments
- Linux
- Windows
- macOS

## Usage in project

### Creating your data directory
```rs
use libvencord::update::download::prepare_dist_directory;

const RELEASE_TAG_DOWNLOAD: &str = "https://github.com/Vendicated/Vencord/releases/download/devbuild";

// Github requires a user-agent when downloading from their 
// services, so we make this required for when downloading 
// files, for this we use the package name, however you can
// use something else
const USER_AGENT: &str = "(libvencord) (https://github.com/Vencord/Installer/crates/libvencord)";

fn main() {
    let _ = prepare_dist_directory(
        &get_data_path("Vencord"), // this will get us the appropriate dist path 
        RELEASE_TAG_DOWNLOAD,
        USER_AGENT,
        [
            "patcher.js".to_string(),
            "preload.js".to_string(),
            "renderer.js".to_string(),
            "renderer.css".to_string(),
        ],
    );
}
```

This function is meant for preparing a dist directory for your mod files, for instance, it will download `https://github.com/Vendicated/Vencord/releases/download/devbuild/patcher.js` along with other included files and put them in your dist directory.

It will also create a `package.json`, without this, node will walk up the file tree and search for a package.json in the parent folders. This might lead to issues if the user for example has `~/package.jso` with type: "module" in it.

### Finding the discord locations

```rs
use libvencord::paths::locations::get_discord_locations;

// ...

fn main() {
    // ...
    let locations = get_discord_locations();
    println!("{:#?}", locations)
}

```

This will look through all available locations discord is in on your specific OS, and tell you some details about the instance.

**Output**

```
Some(
    [
        DiscordLocation {
            name: "Discord.app",
            path: "/Applications/Discord.app",
            branch: Stable,
            patched: false,
            openasar: false,
            is_flatpak: false,
            is_system_electron: false,
        },
        DiscordLocation {
            name: "Discord Canary.app",
            path: "/Applications/Discord Canary.app",
            branch: Canary,
            patched: false,
            openasar: false,
            is_flatpak: false,
            is_system_electron: false,
        },
        DiscordLocation {
            name: "Discord PTB.app",
            path: "/Applications/Discord PTB.app",
            branch: PTB,
            patched: false,
            openasar: false,
            is_flatpak: false,
            is_system_electron: false,
        },
    ],
)
```

Here's the output of the code we've provided, this example shows all the Discord locations found in the `/Applications` directory, if there were any Discord locations inside of `~/Applications` it would show in the output. Do note that it does not account for custom names that people may name their Discord applications, this does not raise any significance to us as no one really does this.

There's another thing worth detailing about the output, it has information on if it's patched or if it has OpenAsar, which should tell you that this library also supports the installation of [OpenAsar](https://github.com/GooseMod/OpenAsar)! Via our features

```toml
libvencord = { ... features = ["openasar"] }
```

### Patching discord

You now have the Discord instance you want to patch or unpatch to, heres some code that will help with that:

```rs
use libvencord::{
    patch::patch_mod::Installer, 
    paths::locations::get_data_path
};

// ...

fn main() {
    // ...
    let installer = Installer::new();

     // This in particular creates an app.asar with our downloaded patcher.js file
    let _ = installer.write_app_asar(
        &get_data_path("Vencord").join("app.asar").to_string_lossy(), 
        &get_data_path("Vencord").join("patcher.js").to_string_lossy()
    );

    let _ = installer.patch(
        location, 
        &get_data_path("Vencord").join("app.asar").to_string_lossy()
    );
}

```

Since in this example we're using [Vencord](https://github.com/Vendicated/Vencord) we have to write our custom `app.asar` within our program to a specified location, however writing is not required as mods can include their own custom `app.asar` elsewhere, such as a link on Github or a cdn.

After writing the custom asar, we attempt to patch a location with the created asar, this process involves renaming and replacing the original `app.asar` within Discord, the original asar is renamed to `_app.asar` so when unpatching, we can return the original asar to its rightful place.

`_app.asar` is also used to detect if the Discord installation has been patched with [Vencord](https://github.com/Vendicated/Vencord) or any other client mod using this library.

This function will also only run if the install is unpatched.

### Unpatching discord

You want to unpatch Discord from its mods, what do you do?

```rs
use libvencord::patch::patch_mod::Installer;

// ...

fn main() {
    // ...
    let installer = Installer::new();

    let _ = installer.unpatch(
        location, 
    );
}

```

It's as simple as this, your client mod will completely be removed. However, just like the install function this will only run properly if the location you're using is patched by the mod.

## Additional Info

We have some built in handling just in case anything goes wrong, use that if you can! 

If theres an instance that a specific error when installing the mod that needs to be known, feel free to let us know.

## License
<!-- TODO: this -->
