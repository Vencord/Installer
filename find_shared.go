package main

import "path"

func isResourcesFolderValid(dir string) bool {
	return ExistsFile(path.Join(dir, "app.asar"))
}

func isResourcesFolderPatched(dir string) bool {
	return ExistsFile(path.Join(dir, "_app.asar"))
}
