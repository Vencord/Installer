package main

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/fatih/color"
)

type Level = int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

var levelNames = map[Level]string{
	LevelDebug: "DEBUG",
	LevelInfo:  "INFO",
	LevelWarn:  "WARN",
	LevelError: "ERROR",
	LevelFatal: "FATAL",
}

var levelColors = map[Level]*color.Color{
	LevelDebug: color.New(color.FgCyan),
	LevelInfo:  color.New(color.FgBlue),
	LevelWarn:  color.New(color.FgYellow),
	LevelError: color.New(color.FgRed),
	LevelFatal: color.New(color.FgHiRed),
}

var Log Handler
var LogLevel = LevelInfo

func init() {
	debug := SliceContainsFunc(os.Args, func(s string) bool {
		return s == "-debug" || s == "--debug"
	})

	if debug {
		LogLevel = LevelDebug
	}
}

type Handler struct {
	mu       sync.Mutex
	paused   int
	buffered []string
}

func (h *Handler) Pause() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.paused++
}

func (h *Handler) Resume() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.paused > 0 {
		h.paused--
	}

	if h.paused == 0 {
		h.flushLocked()
	}
}

func (h *Handler) flushLocked() {
	for _, line := range h.buffered {
		_, _ = fmt.Fprint(os.Stderr, line)
	}
	h.buffered = nil
}

func (h *Handler) Log(level Level, a ...any) {
	if level < LogLevel {
		return
	}

	levelName := levelNames[level]
	var prefix any = levelColors[level].Sprint(levelName + strings.Repeat(" ", len("error")-len(levelName)))
	line := fmt.Sprintln(Prepend(a, prefix)...)

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.paused > 0 && level < LevelFatal {
		h.buffered = append(h.buffered, line)
		return
	}

	h.flushLocked()
	_, _ = fmt.Fprint(os.Stderr, line)
}

func (h *Handler) Debug(a ...any) {
	h.Log(LevelDebug, a...)
}

func (h *Handler) Info(a ...any) {
	h.Log(LevelInfo, a...)
}

func (h *Handler) Warn(a ...any) {
	h.Log(LevelWarn, a...)
}

func (h *Handler) Error(a ...any) {
	h.Log(LevelError, a...)
}

func (h *Handler) Fatal(a ...any) {
	h.Log(LevelFatal, a...)
	os.Exit(1)
}

func (h *Handler) FatalIfErr(err error) {
	if err != nil {
		h.Fatal(err)
	}
}
