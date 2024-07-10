export interface DiscordInstall {
    path: string;
    branch: "Stable" | "Canary" | "PTB";
    is_patched: boolean;
    is_flatpak: boolean;
    custom?: boolean;
}
