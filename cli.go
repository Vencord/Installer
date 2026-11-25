//go:build cli

/*
 * SPDX-License-Identifier: GPL-3.0
 * Vencord Installer, a cross platform gui/cli app for installing Vencord
 * Copyright (c) 2023 Vendicated and Vencord contributors
 */

package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
	"vencordinstaller/buildinfo"

	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
)

var discords []any
var interactive = false
var ansiSupported = false

var (
	styleBold       = color.New(color.Bold)
	styleFaint      = color.New(color.Faint)
	styleCyan       = color.New(color.FgCyan)
	styleBoldCyan   = color.New(color.FgCyan, color.Bold)
	styleGreen      = color.New(color.FgGreen)
	styleRed        = color.New(color.FgRed)
	styleYellow     = color.New(color.FgYellow)
	styleBoldYellow = color.New(color.FgYellow, color.Bold)
)

func selectArrowIcon() string {
	if runtime.GOOS == "windows" {
		return ">"
	}
	return "▸"
}

func checkIcon() string {
	if runtime.GOOS == "windows" {
		return "v"
	}
	return "✔"
}

func crossIcon() string {
	if runtime.GOOS == "windows" {
		return "x"
	}
	return "✗"
}

func isValidBranch(branch string) bool {
	switch branch {
	case "", "stable", "ptb", "canary", "auto":
		return true
	default:
		return false
	}
}

func die(msg string) {
	Log.Error(msg)
	exitFailure()
}

func main() {
	ansiSupported = initAnsiColor()

	// Used by log.go init func
	flag.Bool("debug", false, "Enable debug info")

	var helpFlag = flag.Bool("help", false, "View usage instructions")
	var versionFlag = flag.Bool("version", false, "View the program version")
	var updateSelfFlag = flag.Bool("update-self", false, "Update me to the latest version")
	var installFlag = flag.Bool("install", false, "Install Vencord")
	var updateFlag = flag.Bool("repair", false, "Repair Vencord")
	var uninstallFlag = flag.Bool("uninstall", false, "Uninstall Vencord")
	var installOpenAsarFlag = flag.Bool("install-openasar", false, "Install OpenAsar")
	var uninstallOpenAsarFlag = flag.Bool("uninstall-openasar", false, "Uninstall OpenAsar")
	var locationFlag = flag.String("location", "", "The location of the Discord install to modify")
	var branchFlag = flag.String("branch", "", "The branch of Discord to modify [auto|stable|ptb|canary]")
	flag.Parse()

	if *helpFlag {
		flag.Usage()
		return
	}

	if *versionFlag {
		fmt.Println(
			styleBoldCyan.Sprint("Vencord Installer Cli"),
			styleGreen.Sprint(buildinfo.InstallerTag),
			styleFaint.Sprint("("+buildinfo.InstallerGitHash+")"),
		)
		fmt.Println("Copyright (C) 2023 Vendicated and Vencord contributors")
		fmt.Println("License GPLv3+: GNU GPL version 3 or later <https://gnu.org/licenses/gpl.html>.")
		return
	}

	InitGithubDownloader()

	if *updateSelfFlag {
		if !<-SelfUpdateCheckDoneChan {
			die("Can't update self because checking for updates failed")
		}
		if err := UpdateSelf(); err != nil {
			Log.Error("Failed to update self:", err)
			exitFailure()
		}
		exitSuccess()
	}

	if *locationFlag != "" && *branchFlag != "" {
		die("The 'location' and 'branch' flags are mutually exclusive.")
	}

	if !isValidBranch(*branchFlag) {
		die("The 'branch' flag must be one of the following: [auto|stable|ptb|canary]")
	}

	if *installFlag || *updateFlag {
		if !<-GithubDoneChan {
			die("Not " + Ternary(*installFlag, "installing", "updating") + " as fetching release data failed. If this issue persists, see https://vencord.dev/support")
		}
	}

	install, uninstall, update, installOpenAsar, uninstallOpenAsar := *installFlag, *uninstallFlag, *updateFlag, *installOpenAsarFlag, *uninstallOpenAsarFlag
	switches := []*bool{&install, &update, &uninstall, &installOpenAsar, &uninstallOpenAsar}
	if !SliceContainsFunc(switches, func(b *bool) bool { return *b }) {
		interactive = true

		go func() {
			<-SelfUpdateCheckDoneChan
			if IsSelfOutdated {
				Log.Warn("Your installer is outdated.")
				Log.Warn("To update, select the 'Update Vencord Installer' option to update, or run with --update-self")
			}
		}()

		choices := []string{
			"Install Vencord",
			"Repair Vencord",
			"Uninstall Vencord",
			"Install OpenAsar",
			"Uninstall OpenAsar",
			"View Help Menu",
			"Update Vencord Installer",
			"Quit",
		}
		_, choice, err := runSelect("What would you like to do?", choices)
		handlePromptError(err)

		switch choice {
		case "View Help Menu":
			flag.Usage()
			return
		case "Quit":
			return
		case "Update Vencord Installer":
			if err := UpdateSelf(); err != nil {
				Log.Error("Failed to update self:", err)
				exitFailure()
			}
			exitSuccess()
		}

		*switches[SliceIndex(choices, choice)] = true
	}

	var discord *DiscordInstall
	var err error
	var errSilent error
	if install {
		discord = PromptDiscord("patch", *locationFlag, *branchFlag)
		errSilent = discord.patch()
	} else if uninstall {
		discord = PromptDiscord("unpatch", *locationFlag, *branchFlag)
		errSilent = discord.unpatch()
	} else if update {
		Log.Info("Downloading latest Vencord files...")
		err = installLatestBuilds()
		Log.Info("Done!")
		if err == nil {
			discord = PromptDiscord("repair", *locationFlag, *branchFlag)
			errSilent = discord.patch()
		}
	} else if installOpenAsar {
		discord = PromptDiscord("patch", *locationFlag, *branchFlag)
		if !discord.IsOpenAsar() {
			err = discord.InstallOpenAsar()
		} else {
			die("OpenAsar already installed")
		}
	} else if uninstallOpenAsar {
		discord = PromptDiscord("patch", *locationFlag, *branchFlag)
		if discord.IsOpenAsar() {
			err = discord.UninstallOpenAsar()
		} else {
			die("OpenAsar not installed")
		}
	}

	if err != nil {
		if discord != nil {
			discord.ClearPendingRelaunch()
		}
		Log.Error(err)
		exitFailure()
	}
	if errSilent != nil {
		if discord != nil {
			discord.ClearPendingRelaunch()
		}
		exitFailure()
	}

	if discord != nil {
		handleCliPostPatchRelaunch(discord, interactive || (*locationFlag == "" && *branchFlag == ""))
	}

	exitSuccess()
}

func isStdinTerminal() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func handleCliPostPatchRelaunch(di *DiscordInstall, canPrompt bool) {
	name := branchDisplayName(di.branch)

	if runtime.GOOS == "windows" {
		relaunched, err := di.RelaunchKilled()
		if err != nil {
			Log.Warn("Failed to relaunch", name+":", err)
		} else if relaunched {
			Log.Info("Relaunched", name)
		}
		return
	}

	if !canPrompt || !isStdinTerminal() {
		return
	}

	running := di.FindRunning()
	if running == nil {
		return
	}

	_, choice, err := runSelect(
		name+" is currently running. Would you like to relaunch it?",
		[]string{"Yes", "No"},
	)
	handlePromptError(err)

	if choice == "Yes" {
		Log.Info("Relaunching", name+"...")
		if err := di.Relaunch(running); err != nil {
			Log.Warn("Failed to relaunch", name+":", err)
		}
	}
}

func exit(status int) {
	if runtime.GOOS == "windows" && IsDoubleClickRun() && interactive {
		fmt.Print("Press Enter to exit")
		var b byte
		_, _ = fmt.Scanf("%v", &b)
	}
	os.Exit(status)
}

func exitSuccess() {
	if runtime.GOOS == "windows" {
		color.HiGreen("Success!")
	} else {
		color.HiGreen("✔ Success!")
	}
	exit(0)
}

func exitFailure() {
	if runtime.GOOS == "windows" {
		color.HiRed("Failed! If this issue persists, see https://vencord.dev/support")
	} else {
		color.HiRed("✗ Failed! If this issue persists, see https://vencord.dev/support")
	}
	exit(1)
}

func handlePromptError(err error) {
	if errors.Is(err, promptui.ErrInterrupt) {
		exit(0)
	}

	Log.FatalIfErr(err)
}

func PromptDiscord(action, dir, branch string) *DiscordInstall {
	discords = FindDiscords()

	if branch == "auto" {
		for _, b := range []string{"stable", "canary", "ptb"} {
			for _, discord := range discords {
				install := discord.(*DiscordInstall)
				if install.branch == b {
					return install
				}
			}
		}
		die("No Discord install found. Before proceeding, make sure Discord is installed. snap is not supported!")
	}

	if branch != "" {
		for _, discord := range discords {
			install := discord.(*DiscordInstall)
			if install.branch == branch {
				return install
			}
		}
		die("Discord " + branch + " not found")
	}

	if dir != "" {
		if discord := ParseDiscord(dir, branch); discord != nil {
			return discord
		}

		if discord := ParseDiscordNew(dir, branch, strings.Contains(dir, "com.discordapp")); discord != nil {
			return discord
		}

		die(dir + " is not a valid Discord install. Hint: snap is not supported")
	}

	idx := promptDiscordSelect(action, discords)

	if idx < len(discords) {
		selected := discords[idx].(*DiscordInstall)
		discords = FindDiscords()
		if refreshed, _ := findMatchingInstall(discords, selected); refreshed != nil {
			return refreshed
		}
		return selected
	}

	for {
		custom, err := runPrompt("Custom Discord Location")
		handlePromptError(err)

		if di := ParseDiscord(custom, ""); di != nil {
			return di
		}

		if di := ParseDiscordNew(custom, "", strings.Contains(custom, "com.discordapp")); di != nil {
			return di
		}

		Log.Error("Invalid Discord install!")
	}
}

type discordSelectItem struct {
	Active   string
	Inactive string
	Selected string
	Details  string
}

func truncateEnd(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 0 {
		return ""
	}
	if maxLen <= 3 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-3]) + "..."
}

func truncateMiddle(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return truncateEnd(s, maxLen)
	}
	remaining := maxLen - 3
	left := remaining / 2
	right := remaining - left
	return string(runes[:left]) + "..." + string(runes[len(runes)-right:])
}

func findWrapBreak(runes []rune, limit int) int {
	if limit >= len(runes) {
		return len(runes)
	}

	minBreak := limit / 2
	for i := limit - 1; i >= minBreak; i-- {
		if runes[i] == '/' || runes[i] == '\\' || runes[i] == ' ' {
			return i + 1
		}
	}

	return limit
}

func wrapDetailLine(label, value string, maxWidth int, valueStyle *color.Color) string {
	prefix := "  " + label + ": "
	continuationIndent := "    "
	runes := []rune(value)

	styleVal := func(s string) string {
		if valueStyle != nil {
			return valueStyle.Sprint(s)
		}
		return s
	}

	if len([]rune(prefix))+len(runes) <= maxWidth {
		return "  " + styleFaint.Sprint(label+":") + " " + styleVal(value)
	}

	var lines []string
	firstLimit := maxWidth - len([]rune(prefix))
	if firstLimit >= 8 {
		end := findWrapBreak(runes, firstLimit)
		lines = append(lines, "  "+styleFaint.Sprint(label+":")+" "+styleVal(string(runes[:end])))
		runes = runes[end:]
	} else {
		lines = append(lines, styleFaint.Sprint(truncateEnd("  "+label+":", maxWidth)))
	}

	contLimit := maxWidth - len([]rune(continuationIndent))
	if contLimit < 1 {
		contLimit = 1
	}

	for len(runes) > 0 {
		end := findWrapBreak(runes, contLimit)
		lines = append(lines, continuationIndent+styleVal(string(runes[:end])))
		runes = runes[end:]
	}

	return strings.Join(lines, "\n")
}

func selectLayout(itemCount int, reserveHeight int) (maxWidth, maxLabelWidth, maxItemWidth, size int) {
	width, height := getTerminalSize()

	maxWidth = width - 1
	if maxWidth < 20 {
		maxWidth = 20
	}

	maxLabelWidth = maxWidth - 4
	if maxLabelWidth < 10 {
		maxLabelWidth = 10
	}

	maxItemWidth = maxWidth - 4
	if maxItemWidth < 10 {
		maxItemWidth = 10
	}

	maxVisible := height - reserveHeight
	if maxVisible < 5 {
		maxVisible = 5
	}

	size = itemCount
	if size > maxVisible {
		size = maxVisible
	}

	return maxWidth, maxLabelWidth, maxItemWidth, size
}

func buildSelectTemplates(helpText string, labelField, activeField, inactiveField, selectedField string) *promptui.SelectTemplates {
	templates := &promptui.SelectTemplates{
		Help: fmt.Sprintf("{{ %q }}", styleFaint.Sprint(helpText)),
	}

	if ansiSupported {
		templates.Label = fmt.Sprintf("%s {{ %s | bold }}: ", styleCyan.Sprint("?"), labelField)
		templates.Active = fmt.Sprintf("%s {{ %s }}", styleBoldCyan.Sprint(selectArrowIcon()), activeField)
		templates.Inactive = fmt.Sprintf("  {{ %s }}", inactiveField)
		templates.Selected = fmt.Sprintf("%s {{ %s | faint }}", styleGreen.Sprint(checkIcon()), selectedField)
	} else {
		templates.Label = fmt.Sprintf("? {{ %s }}: ", labelField)
		templates.Active = fmt.Sprintf("> {{ %s }}", activeField)
		templates.Inactive = fmt.Sprintf("  {{ %s }}", inactiveField)
		templates.Selected = fmt.Sprintf("  {{ %s }}", selectedField)
	}

	return templates
}

func runSelect(label string, items []string) (int, string, error) {
	maxWidth, maxLabelWidth, maxItemWidth, size := selectLayout(len(items), 6)

	displayItems := SliceMap(items, func(item string) string {
		return truncateEnd(item, maxItemWidth)
	})

	helpText := truncateEnd("Use arrow keys to navigate, Enter to confirm", maxWidth)
	activeField := "."
	if ansiSupported {
		activeField = ". | cyan | bold"
	}

	sel := &promptui.Select{
		Label:     truncateEnd(label, maxLabelWidth),
		Items:     displayItems,
		Size:      size,
		Templates: buildSelectTemplates(helpText, ".", activeField, ".", "."),
	}

	Log.Pause()
	idx, _, err := sel.Run()
	Log.Resume()

	if err != nil {
		return 0, "", err
	}

	return idx, items[idx], nil
}

func runPrompt(label string) (string, error) {
	width, _ := getTerminalSize()
	maxLabelWidth := width - 5
	if maxLabelWidth < 10 {
		maxLabelWidth = 10
	}

	templates := &promptui.PromptTemplates{}
	if ansiSupported {
		templates.Prompt = fmt.Sprintf("%s {{ . | bold }}: ", styleCyan.Sprint("?"))
		templates.Valid = fmt.Sprintf("%s {{ . | bold }}: ", styleGreen.Sprint(checkIcon()))
		templates.Invalid = fmt.Sprintf("%s {{ . | bold }}: ", styleRed.Sprint(crossIcon()))
		templates.Success = "{{ . | faint }}: "
	} else {
		templates.Prompt = "? {{ . }}: "
		templates.Valid = "? {{ . }}: "
		templates.Invalid = "! {{ . }}: "
		templates.Success = "{{ . }}: "
	}

	prompt := &promptui.Prompt{
		Label:     truncateEnd(label, maxLabelWidth),
		Templates: templates,
	}

	Log.Pause()
	res, err := prompt.Run()
	Log.Resume()

	return res, err
}

func formatDiscordSelectItem(install *DiscordInstall, maxItemWidth, maxDetailWidth int) (discordSelectItem, bool) {
	branch := branchDisplayName(install.branch)
	sep := " - "
	badge := ""
	if install.isPatched {
		badge = " [Vencord Installed]"
	}

	full := branch + sep + install.path + badge
	if len([]rune(full)) <= maxItemWidth {
		return discordSelectItem{
			Active:   styleBoldCyan.Sprint(branch) + styleCyan.Sprint(sep+install.path) + styleGreen.Sprint(badge),
			Inactive: styleBold.Sprint(branch) + styleFaint.Sprint(sep+install.path) + styleGreen.Sprint(badge),
			Selected: full,
		}, false
	}

	availPath := maxItemWidth - len([]rune(branch)) - len([]rune(sep)) - len([]rune(badge))
	var active, inactive, selected string
	suffixCut := false
	if availPath >= 10 {
		shortPath := truncateMiddle(install.path, availPath)
		active = styleBoldCyan.Sprint(branch) + styleCyan.Sprint(sep+shortPath) + styleGreen.Sprint(badge)
		inactive = styleBold.Sprint(branch) + styleFaint.Sprint(sep+shortPath) + styleGreen.Sprint(badge)
		selected = branch + sep + shortPath + badge
	} else {
		truncated := truncateEnd(full, maxItemWidth)
		active = styleBoldCyan.Sprint(truncated)
		inactive = truncated
		selected = truncated
		suffixCut = install.isPatched
	}

	details := wrapDetailLine("Path", install.path, maxDetailWidth, nil)
	if suffixCut {
		details += "\n" + wrapDetailLine("Status", "Vencord Installed", maxDetailWidth, styleGreen)
	}

	return discordSelectItem{
		Active:   active,
		Inactive: inactive,
		Selected: selected,
		Details:  details,
	}, true
}

func promptDiscordSelect(action string, installs []any) int {
	maxWidth, maxLabelWidth, maxItemWidth, _ := selectLayout(len(installs)+1, 6)

	anyTruncated := false
	items := make([]discordSelectItem, 0, len(installs)+1)
	for _, d := range installs {
		install := d.(*DiscordInstall)
		item, truncated := formatDiscordSelectItem(install, maxItemWidth, maxWidth)
		if truncated {
			anyTruncated = true
		}
		items = append(items, item)
	}

	customLabel := truncateEnd("Custom Location", maxItemWidth)
	items = append(items, discordSelectItem{
		Active:   styleBoldCyan.Sprint(customLabel),
		Inactive: customLabel,
		Selected: customLabel,
	})

	reserveHeight := 6
	if anyTruncated {
		reserveHeight = 10
	}
	_, _, _, size := selectLayout(len(items), reserveHeight)

	helpText := truncateEnd("Use arrow keys to navigate, Enter to confirm", maxWidth)
	templates := buildSelectTemplates(helpText, ".", ".Active", ".Inactive", ".Selected")
	if anyTruncated {
		templates.Details = "\n{{ .Details }}"
	}

	sel := &promptui.Select{
		Label:     truncateEnd("Select Discord install to "+action, maxLabelWidth),
		Items:     items,
		Size:      size,
		Templates: templates,
	}

	Log.Pause()
	idx, _, err := sel.Run()
	Log.Resume()
	handlePromptError(err)

	return idx
}

func InstallLatestBuilds() error {
	if IsDevInstall {
		return nil
	}

	return installLatestBuilds()
}

func HandleScuffedInstall() {
	_, _ = styleBoldYellow.Println("Hold On!")
	_, _ = styleYellow.Println("You have a broken Discord Install.")
	_, _ = styleYellow.Println("Please reinstall Discord before proceeding!")
	_, _ = styleYellow.Println("Otherwise, Vencord will likely not work.")
}
