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

	versionBox := gtk.NewBox(gtk.OrientationVertical, 0)
	attachVersionLabel(versionBox, "Installer", fmt.Sprintf("%s (%s)", buildinfo.InstallerTag, buildinfo.InstallerGitHash))
	attachVersionLabel(versionBox, "Local Vencord", InstalledHash)

	if GithubError == nil {
		if IsDevInstall {
			versionBox.Append(gtk.NewLabel("Not updating Vencord due to being in DevMode"))
		} else {
			versionLabel := attachVersionLabel(versionBox, "Latest Vencord", LatestHash)
			go func() {
				<-GithubDoneChan
				glib.IdleAdd(func() {
					versionLabel.SetLabel(LatestHash)
				})
			}()
		}
	} else {
		versionBox.Append(gtk.NewLabel("Failed to fetch latest version from github: " + GithubError.Error()))
	}

	// header.Append(title)
	header.Append(fileLocationBox)
	header.Append(versionBox)

	return header
}

func attachVersionLabel(parent *gtk.Box, name, version string) *gtk.Label {
	box := gtk.NewBox(gtk.OrientationHorizontal, 0)
	box.AddCSSClass("version-row")

	nameLabel := gtk.NewLabel(name + " Version:")
	nameLabel.AddCSSClass("version-name")
	versionLabel := gtk.NewLabel(version)
	versionLabel.AddCSSClass("version-value")

	box.Append(nameLabel)
	box.Append(versionLabel)

	parent.Append(box)
	return versionLabel
}

func createInstaller() gtk.Widgetter {
	installer := gtk.NewBox(gtk.OrientationVertical, 10)

	title := gtk.NewLabel("Please select an install to patch")
	title.SetXAlign(0)
	title.SetYAlign(0)
	title.AddCSSClass("installer-title")

	var firstButton *gtk.CheckButton
	radioBox := gtk.NewBox(gtk.OrientationVertical, 3)
	for _, di := range discords {
		install := di.(*DiscordInstall)

		radio := gtk.NewCheckButton()
		if firstButton == nil {
			firstButton = radio
			radio.SetActive(true)
		} else {
			radio.SetGroup(firstButton)
		}

		labelBox := gtk.NewBox(gtk.OrientationHorizontal, 3)
		labelBox.SetParent(radio)

		icon, _ := loadAssetPng(install.branch + ".png")
		labelBox.Append(gtk.NewImageFromPixbuf(icon))
		labelBox.Append(gtk.NewLabel(install.branch))

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

	installer.Append(title)
	installer.Append(radioBox)
	installer.Append(createInstallerButtons())

	return installer
}

func createInstallerButtons() gtk.Widgetter {
	installButton := gtk.NewButtonWithLabel("Install")
	installButton.SetCSSClasses([]string{"button", "button-positive"})
	repairButton := gtk.NewButtonWithLabel("Repair")
	repairButton.SetCSSClasses([]string{"button", "button-neutral"})
	uninstallButton := gtk.NewButtonWithLabel("Uninstall")
	uninstallButton.SetCSSClasses([]string{"button", "button-negative"})
	openAsarButton := gtk.NewButtonWithLabel("Install OpenAsar")
	openAsarButton.SetCSSClasses([]string{"button", "button-positive"})

	grid := gtk.NewGrid()
	grid.SetRowSpacing(8)
	grid.SetColumnSpacing(8)
	grid.Attach(installButton, 0, 0, 1, 1)
	grid.Attach(repairButton, 1, 0, 1, 1)
	grid.Attach(uninstallButton, 0, 1, 1, 1)
	grid.Attach(openAsarButton, 1, 1, 1, 1)

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
