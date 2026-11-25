package main

import "path"

func isResourcesFolderValid(dir string) bool {
	return ExistsFile(path.Join(dir, "app.asar"))
}

func isResourcesFolderPatched(dir string) bool {
	return ExistsFile(path.Join(dir, "_app.asar"))
}

func findMatchingInstall(list []any, target *DiscordInstall) (*DiscordInstall, int) {
	if target == nil {
		return nil, -1
	}

	for i, d := range list {
		install := d.(*DiscordInstall)
		if install.path == target.path {
			return install, i
		}
	}

	for i, d := range list {
		install := d.(*DiscordInstall)
		if install.branch == target.branch && install.isFlatpak == target.isFlatpak {
			return install, i
		}
	}

	return nil, -1
}
