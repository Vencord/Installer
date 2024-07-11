export interface DiscordInstall {
    path: string;
    app_path: string;
    branch: "Stable" | "Canary" | "PTB";
    is_patched: boolean;
    is_flatpak: boolean;
    custom?: boolean;
}
