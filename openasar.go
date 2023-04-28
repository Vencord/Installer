/*
 * SPDX-License-Identifier: GPL-3.0
 * Vencord Installer, a cross platform gui/cli app for installing Vencord
 * Copyright (c) 2023 Vendicated and Vencord contributors
 */

package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	path "path/filepath"
	"strconv"
)

const OpenAsarDownloadLink = "https://github.com/GooseMod/OpenAsar/releases/download/nightly/app.asar"

func FindAsarFile(dir string) (*os.File, error) {
	for _, file := range []string{"app.asar", "_app.asar"} {
		f, err := os.Open(path.Join(dir, file))
		if err != nil {
			continue
		}
		stats, err := f.Stat()
		if err == nil && !stats.IsDir() {
			return f, nil
		}
		_ = f.Close()
	}
	return nil, errors.New("Install at " + dir + " has no asar file")
}

func (di *DiscordInstall) IsOpenAsar() (retBool bool) {
	if di.isOpenAsar != nil {
		return *di.isOpenAsar
	}

	defer func() {
		fmt.Println("Checking if", di.path, "is using OpenAsar:", retBool)
		di.isOpenAsar = &retBool
	}()

	for _, version := range di.versions {
		fmt.Println(version, path.Join(version, ".."))
		asarFile, err := FindAsarFile(path.Join(version, ".."))
		if err != nil {
			fmt.Println(err)
			continue
		}

		b, err := io.ReadAll(asarFile)
		_ = asarFile.Close()
		if err != nil {
			fmt.Println(err)
			continue
		}

		if bytes.Contains(b, []byte("OpenAsar")) {
			return true
		}
	}

	return false
}

func (di *DiscordInstall) InstallOpenAsar() error {
	for _, version := range di.versions {
		dir := path.Join(version, "..")
		asarFile, err := FindAsarFile(dir)
		if err != nil {
			return err
		}
		_ = asarFile.Close()

		if err = os.Rename(asarFile.Name(), path.Join(dir, "app.asar.original")); err != nil {
			return err
		}

		res, err := http.Get(OpenAsarDownloadLink)
		if err != nil {
			return err
		} else if res.StatusCode >= 300 {
			return errors.New("Failed to fetch OpenAsar - " + strconv.Itoa(res.StatusCode) + ": " + res.Status)
		}

		outFile, err := os.Create(asarFile.Name())
		if err != nil {
			return err
		}

		if _, err = io.Copy(outFile, res.Body); err != nil {
			return err
		}
	}

	di.isOpenAsar = Ptr(true)
	return nil
}

func (di *DiscordInstall) UninstallOpenAsar() error {
	for _, version := range di.versions {
		dir := path.Join(version, "..")
		originalAsar := path.Join(dir, "app.asar.original")
		if !ExistsFile(originalAsar) {
			return errors.New("No app.asar.original. Reinstall Discord")
		}

		asarFile, err := FindAsarFile(dir)
		if err != nil {
			return err
		}
		_ = asarFile.Close()

		if err = os.Rename(originalAsar, asarFile.Name()); err != nil {
			return err
		}
	}

	di.isOpenAsar = Ptr(false)
	return nil
}
