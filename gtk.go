//go:build gtk

package main

import (
	"embed"
	_ "embed"
	"fmt"
	g "github.com/AllenDang/giu"
	"github.com/diamondburned/gotk4/pkg/gdkpixbuf/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"log"
	"os"
	"strings"
	"vencordinstaller/buildinfo"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

//go:embed gtk.css
var styleCSS string

//go:embed assets/*
var assets embed.FS

var discords []any

func init() {
	LogLevel = LevelDebug
}

type Labellable interface {
	gtk.Widgetter
	SetLabel(label string)
	AddCSSClass(class string)
}

func main() {
	InitGithubDownloader()
	discords = FindDiscords()

	go func() {
		<-SelfUpdateCheckDoneChan
		g.Update()
	}()

	app := gtk.NewApplication("dev.vencord.VesktopInstaller", gio.ApplicationFlagsNone)
	app.ConnectActivate(func() {
		activate(app)
	})

	os.Exit(app.Run(os.Args))
}

func activate(app *gtk.Application) {
	gtk.StyleContextAddProviderForDisplay(
		gdk.DisplayGetDefault(), loadCSS(styleCSS),
		gtk.STYLE_PROVIDER_PRIORITY_APPLICATION,
	)

	window := gtk.NewApplicationWindow(app)
	window.SetTitle("Vencord Installer")
	window.SetDefaultSize(800, 600)

	content := gtk.NewBox(gtk.OrientationVertical, 20)
	content.AddCSSClass("main-content")
	content.Append(createHeader())
	content.Append(createInstaller())

	window.SetChild(content)

	window.Show()
}

func createHeader() gtk.Widgetter {
	header := gtk.NewBox(gtk.OrientationVertical, 10)

	//title := gtk.NewLabel("Vencord Installer")
	//title.AddCSSClass("title")

	fileLocationBox := gtk.NewBox(gtk.OrientationVertical, 0)

	versionGrid := gtk.NewGrid()
	versionGrid.SetHAlign(gtk.AlignStart)
	versionGrid.SetColumnHomogeneous(true)
	versionGrid.SetColumnSpacing(5)

	const VencordTreeBaseUrl = "https://github.com/Vendicated/Vencord/tree/"
	const InstallerTreeBaseUrl = "https://github.com/Vencord/Installer/tree/"
	var installerUrl string
	if buildinfo.InstallerGitHash == buildinfo.VersionUnknown {
		installerUrl = InstallerTreeBaseUrl + "main"
	} else {
		installerUrl = InstallerTreeBaseUrl + buildinfo.InstallerGitHash
	}

	attachVersionLabel(versionGrid, 0, "Installer", fmt.Sprintf("%s (%s)", buildinfo.InstallerTag, buildinfo.InstallerGitHash), installerUrl)
	attachVersionLabel(versionGrid, 1, "Local Vencord", InstalledHash, VencordTreeBaseUrl+InstalledHash)

	if GithubError == nil {
		if IsDevInstall {
			versionGrid.Attach(gtk.NewLabel("Not updating Vencord due to being in DevMode"), 0, 2, 1, 1)
		} else {
			linkButton := attachVersionLabel(versionGrid, 2, "Latest Vencord", LatestHash, VencordTreeBaseUrl+LatestHash)
			go func() {
				<-GithubDoneChan
				glib.IdleAdd(func() {
					linkButton.SetURI(VencordTreeBaseUrl + LatestHash)
					linkButton.SetLabel(LatestHash)
				})
			}()
		}
	} else {
		versionGrid.Attach(gtk.NewLabel("Failed to fetch latest version from github: "+GithubError.Error()), 0, 2, 1, 1)
	}

	// header.Append(title)
	header.Append(fileLocationBox)
	header.Append(versionGrid)

	return header
}

func attachVersionLabel(parent *gtk.Grid, row int, name, version, url string) *gtk.LinkButton {
	nameLabel := gtk.NewLabel(name + " Version:")
	nameLabel.SetXAlign(0)
	nameLabel.AddCSSClass("version-name")

	linkButton := gtk.NewLinkButtonWithLabel(url, version)
	linkButton.AddCSSClass("version-value")
	linkButton.Child().(*gtk.Label).SetXAlign(0)

	parent.Attach(nameLabel, 0, row, 1, 1)
	parent.Attach(linkButton, 1, row, 1, 1)

	return linkButton
}

func createInstaller() gtk.Widgetter {
	installer := gtk.NewBox(gtk.OrientationVertical, 10)

	title := gtk.NewLabel("Please select an install to patch")
	title.SetXAlign(0)
	title.SetYAlign(0)
	title.AddCSSClass("installer-title")

	var firstButton *gtk.CheckButton
	radioBox := gtk.NewBox(gtk.OrientationVertical, 5)
	for _, di := range discords {
		install := di.(*DiscordInstall)

		radio := gtk.NewCheckButton()
		radio.AddCSSClass("branch-radio-button")
		if firstButton == nil {
			firstButton = radio
			radio.SetActive(true)
		} else {
			radio.SetGroup(firstButton)
		}

		labelBox := gtk.NewBox(gtk.OrientationHorizontal, 5)
		labelBox.SetParent(radio)

		icon, _ := loadAssetPng(install.branch + ".png")
		labelBox.Append(gtk.NewImageFromPixbuf(icon))
		//goland:noinspection GoDeprecation strings.Title is useeeful

		branchLabel := gtk.NewLabel(strings.Title(install.branch))
		branchLabel.SetXAlign(0)
		branchLabel.AddCSSClass("branch-radio-name")
		labelBox.Append(branchLabel)

		if install.isPatched {
			starIcon := gtk.NewImageFromIconName("starred-symbolic")
			starIcon.SetTooltipText("This install is patched")
			labelBox.Append(starIcon)
		}

		pathLabel := gtk.NewLabel(install.path)
		pathLabel.SetOpacity(0.5)
		labelBox.Append(pathLabel)

		radioBox.Append(radio)
	}

	customRadio := gtk.NewCheckButton()
	customRadio.AddCSSClass("branch-radio-button")
	if firstButton != nil {
		customRadio.SetGroup(firstButton)
	}
	labelBox := gtk.NewBox(gtk.OrientationHorizontal, 5)
	labelBox.SetParent(customRadio)

	labelBox.Append(gtk.NewImageFromIconName("document-open-symbolic"))
	customLabel := gtk.NewLabel("Custom Location")
	customLabel.SetXAlign(0)
	customLabel.AddCSSClass("branch-radio-name")
	labelBox.Append(customLabel)

	radioBox.Append(customRadio)

	installer.Append(title)
	installer.Append(radioBox)
	installer.Append(createInstallerButtons())

	return installer
}

func createInstallerButtons() gtk.Widgetter {
	installButton := gtk.NewButtonWithLabel("Install")
	installButton.SetHExpand(true)
	installButton.SetCSSClasses([]string{"button", "button-positive"})
	repairButton := gtk.NewButtonWithLabel("Repair")
	repairButton.SetHExpand(true)
	repairButton.SetCSSClasses([]string{"button", "button-neutral"})
	uninstallButton := gtk.NewButtonWithLabel("Uninstall")
	uninstallButton.SetHExpand(true)
	uninstallButton.SetCSSClasses([]string{"button", "button-negative"})
	openAsarButton := gtk.NewButtonWithLabel("Install OpenAsar")
	openAsarButton.SetHExpand(true)
	openAsarButton.SetCSSClasses([]string{"button", "button-positive"})

	grid := gtk.NewGrid()
	grid.SetColumnSpacing(8)
	grid.SetColumnHomogeneous(true)
	grid.Attach(installButton, 0, 0, 1, 1)
	grid.Attach(repairButton, 1, 0, 1, 1)
	grid.Attach(uninstallButton, 2, 0, 1, 1)
	grid.Attach(openAsarButton, 3, 0, 1, 1)

	return grid
}

func InstallLatestBuilds() error {
	return nil
}

func loadCSS(content string) *gtk.CSSProvider {
	prov := gtk.NewCSSProvider()
	prov.ConnectParsingError(func(sec *gtk.CSSSection, err error) {
		// Optional line parsing routine.
		loc := sec.StartLocation()
		lines := strings.Split(content, "\n")
		log.Printf("CSS error (%v) at line: %q", err, lines[loc.Lines()])
	})
	prov.LoadFromData(content)
	return prov
}

func loadAssetPng(path string) (*gdkpixbuf.Pixbuf, error) {
	data, err := assets.ReadFile("assets/" + path)
	if err != nil {
		return nil, fmt.Errorf("Failed to read asset %s: %w", path, err)
	}

	l, err := gdkpixbuf.NewPixbufLoaderWithType("png")
	if err != nil {
		return nil, fmt.Errorf("NewLoaderWithType png: %w", err)
	}
	defer l.Close()

	if err := l.Write(data); err != nil {
		return nil, fmt.Errorf("PixbufLoader.Write: %w", err)
	}

	if err := l.Close(); err != nil {
		return nil, fmt.Errorf("PixbufLoader.Close: %w", err)
	}

	return l.Pixbuf(), nil
}
