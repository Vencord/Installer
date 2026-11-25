//go:build cli

/*
 * SPDX-License-Identifier: GPL-3.0
 * Vencord Installer, a cross platform gui/cli app for installing Vencord
 * Copyright (c) 2023 Vendicated and Vencord contributors
 */

package main

import (
	"errors"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatih/color"
)

var (
	errCanceled = errors.New("canceled")
	errBack     = errors.New("back")
)

type selfUpdateCheckedMsg struct {
	outdated bool
}

func waitForSelfUpdateCmd() tea.Cmd {
	return func() tea.Msg {
		<-SelfUpdateCheckDoneChan
		return selfUpdateCheckedMsg{outdated: IsSelfOutdated}
	}
}

func runProgram(model tea.Model) (tea.Model, error) {
	Log.Pause()
	defer Log.Resume()

	return tea.NewProgram(model).Run()
}

type selectItem struct {
	active   string
	inactive string
	selected string
	details  string
}

type selectModel struct {
	label       string
	count       int
	buildItems  func(maxItemWidth, maxWidth int) (items []selectItem, hasDetails bool)
	cursor      int
	scroll      int
	width       int
	height      int
	allowBack   bool
	yesNo       bool
	watchUpdate bool
	outdated    bool
	done        bool
	canceled    bool
	wentBack    bool
}

func newSelectModel(
	label string,
	count int,
	buildItems func(maxItemWidth, maxWidth int) ([]selectItem, bool),
	allowBack, yesNo, watchUpdate bool,
) selectModel {
	w, h := getTerminalSize()
	return selectModel{
		label:       label,
		count:       count,
		buildItems:  buildItems,
		width:       w,
		height:      h,
		allowBack:   allowBack,
		yesNo:       yesNo,
		watchUpdate: watchUpdate,
		outdated:    watchUpdate && IsSelfOutdated,
	}
}

func (m selectModel) Init() tea.Cmd {
	if m.watchUpdate && !m.outdated {
		return waitForSelfUpdateCmd()
	}
	return nil
}

func (m selectModel) visibleCount(hasDetails bool) int {
	reserve := 4
	if m.outdated {
		reserve += 2
	}
	if hasDetails {
		reserve += 4
	}
	maxVisible := m.height - reserve
	if maxVisible < 3 {
		maxVisible = 3
	}
	if m.count < maxVisible {
		return m.count
	}
	return maxVisible
}

func (m *selectModel) clampScroll(visible int) {
	if visible <= 0 || m.count <= visible {
		m.scroll = 0
		return
	}
	if m.cursor < m.scroll {
		m.scroll = m.cursor
	}
	if m.cursor >= m.scroll+visible {
		m.scroll = m.cursor - visible + 1
	}
	if maxScroll := m.count - visible; m.scroll > maxScroll {
		m.scroll = maxScroll
	}
}

func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case selfUpdateCheckedMsg:
		m.outdated = msg.outdated

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.canceled = true
			return m, tea.Quit
		case "esc":
			if m.allowBack {
				m.wentBack = true
				return m, tea.Quit
			}
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < m.count-1 {
				m.cursor++
			}
		case "y", "Y":
			if m.yesNo && m.count >= 2 {
				m.cursor = 0
				m.done = true
				return m, tea.Quit
			}
		case "n", "N":
			if m.yesNo && m.count >= 2 {
				m.cursor = 1
				m.done = true
				return m, tea.Quit
			}
		case "enter":
			if m.count > 0 {
				m.done = true
				return m, tea.Quit
			}
		}
	}

	m.clampScroll(m.visibleCount(false))
	return m, nil
}

func (m selectModel) View() string {
	if m.canceled || m.wentBack {
		return ""
	}

	maxWidth := m.width - 1
	if maxWidth < 20 {
		maxWidth = 20
	}
	maxItemWidth := maxWidth - 4
	if maxItemWidth < 10 {
		maxItemWidth = 10
	}

	items, hasDetails := m.buildItems(maxItemWidth, maxWidth)
	if m.done {
		return fmt.Sprintf("%s %s\n", styleGreen.Sprint(checkIcon()), styleFaint.Sprint(items[m.cursor].selected))
	}

	visible := m.visibleCount(hasDetails)
	m.clampScroll(visible)
	end := m.scroll + visible

	help := "Use arrow keys to navigate, Enter to confirm"
	if m.yesNo {
		help = "Use arrow keys or y/n, Enter to confirm"
	} else if m.allowBack {
		help = "Use arrow keys to navigate, Enter to confirm, Esc to go back"
	}

	var b strings.Builder
	if m.outdated {
		notice := "Installer outdated: select 'Update Vencord Installer' or run with --update-self"
		b.WriteString(styleYellow.Sprint(truncateEnd(notice, maxWidth)) + "\n")
	}

	b.WriteString(styleFaint.Sprint(truncateEnd(help, maxWidth)) + "\n")
	b.WriteString(fmt.Sprintf("%s %s:\n", styleCyan.Sprint("?"), styleBold.Sprint(truncateEnd(m.label, maxItemWidth))))

	for i := m.scroll; i < end; i++ {
		page := " "
		if i == m.scroll && m.scroll > 0 {
			page = "^"
		} else if i == end-1 && end < len(items) {
			page = "v"
		}

		if i == m.cursor {
			b.WriteString(fmt.Sprintf("%s %s %s\n", styleFaint.Sprint(page), styleBoldCyan.Sprint(selectArrowIcon()), items[i].active))
		} else {
			b.WriteString(fmt.Sprintf("%s   %s\n", styleFaint.Sprint(page), items[i].inactive))
		}
	}

	if hasDetails && items[m.cursor].details != "" {
		b.WriteString("\n" + items[m.cursor].details + "\n")
	}

	return b.String()
}

type customLocationModel struct {
	value    []rune
	width    int
	errMsg   string
	result   *DiscordInstall
	done     bool
	canceled bool
	wentBack bool
}

func newCustomLocationModel() customLocationModel {
	w, _ := getTerminalSize()
	return customLocationModel{width: w}
}

func (m customLocationModel) Init() tea.Cmd {
	return nil
}

func (m customLocationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			m.canceled = true
			return m, tea.Quit
		case tea.KeyEsc:
			m.wentBack = true
			return m, tea.Quit
		case tea.KeyBackspace:
			if len(m.value) > 0 {
				m.value = m.value[:len(m.value)-1]
				m.errMsg = ""
			}
		case tea.KeyCtrlU:
			m.value = nil
			m.errMsg = ""
		case tea.KeySpace:
			m.value = append(m.value, ' ')
			m.errMsg = ""
		case tea.KeyRunes:
			for _, r := range msg.Runes {
				if r != '\r' && r != '\n' {
					m.value = append(m.value, r)
				}
			}
			m.errMsg = ""
		case tea.KeyEnter:
			raw := strings.Trim(strings.TrimSpace(string(m.value)), `"'`)
			if raw == "" {
				m.errMsg = "Please enter a path to a Discord installation"
				return m, nil
			}
			if di := ParseDiscord(raw, ""); di != nil {
				m.result = di
				m.done = true
				return m, tea.Quit
			}
			if di := ParseDiscordNew(raw, "", strings.Contains(raw, "com.discordapp")); di != nil {
				m.result = di
				m.done = true
				return m, tea.Quit
			}
			m.errMsg = "Invalid Discord install! Hint: snap is not supported"
		}
	}

	return m, nil
}

func (m customLocationModel) View() string {
	if m.canceled || m.wentBack {
		return ""
	}

	maxWidth := m.width - 1
	if maxWidth < 20 {
		maxWidth = 20
	}

	if m.done {
		return fmt.Sprintf("%s %s\n", styleGreen.Sprint(checkIcon()), styleFaint.Sprint(truncateMiddle(m.result.path, maxWidth-4)))
	}

	label := truncateEnd("Custom Discord Location", maxWidth-14)
	fieldWidth := maxWidth - len([]rune("? "+label+": ")) - 1
	if fieldWidth < 5 {
		fieldWidth = 5
	}

	shown := m.value
	if len(shown) > fieldWidth {
		shown = shown[len(shown)-fieldWidth:]
	}

	cursor := "_"
	if ansiSupported {
		cursor = color.New(color.ReverseVideo).Sprint(" ")
	}

	var b strings.Builder
	b.WriteString(styleFaint.Sprint(truncateEnd("Enter path to Discord install, Esc to go back", maxWidth)) + "\n")
	b.WriteString(fmt.Sprintf("%s %s: %s%s\n", styleCyan.Sprint("?"), styleBold.Sprint(label), string(shown), cursor))
	if m.errMsg != "" {
		b.WriteString(fmt.Sprintf("  %s %s\n", styleRed.Sprint(crossIcon()), styleRed.Sprint(truncateEnd(m.errMsg, maxWidth-4))))
	}
	return b.String()
}
