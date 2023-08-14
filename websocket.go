//go:build web

package main

import (
	"encoding/json"
	"fmt"
	"github.com/sqweek/dialog"
	"net/http"
)

import "github.com/gorilla/websocket"

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return r.Header.Get("Origin") == "https://vencord.dev" || true
	},
}

type DiscordData struct {
	Branch     string `json:"branch"`
	Path       string `json:"path"`
	IsPatched  bool   `json:"isPatched"`
	IsOpenAsar bool   `json:"isOpenAsar"`
}

const (
	OpError               = "ERROR"
	OpOk                  = "OK"
	OpListInstalls        = "LIST_INSTALLS"
	OpChooseCustomInstall = "CHOOSE_CUSTOM_INSTALL"
	OpPatch               = "PATCH"
	OpUnpatch             = "UNPATCH"
	OpRepair              = "REPAIR"
	OpInstallOpenAsar     = "INSTALL_OPENASAR"
	OpUninstallOpenAsar   = "UNINSTALL_OPENASAR"
)

type Payload struct {
	Nonce string          `json:"nonce"`
	Op    string          `json:"op"`
	Data  json.RawMessage `json:"data"`
}

type PayloadWithMessage struct {
	Payload
	Message string `json:"message"`
}

func launch(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	write := func(payload any) {
		b, err := json.Marshal(payload)
		if err != nil {
			fmt.Println("Failed to marshal payload", err)
			return
		}
		err = conn.WriteMessage(websocket.TextMessage, b)
		if err != nil {
			fmt.Println("Failed to send message", err)
		}
	}

	reply := func(nonce string, data any) {
		b, err := json.Marshal(data)
		if err != nil {
			fmt.Println("Failed to marshal payload", err)
			return
		}

		write(Payload{
			Nonce: nonce,
			Op:    OpOk,
			Data:  b,
		})
	}

	replyError := func(nonce string, message string) {
		write(PayloadWithMessage{
			Payload: Payload{
				Nonce: nonce,
				Op:    OpError,
			},
			Message: message,
		})
	}

	defer conn.Close()
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("Error while reading message:", err)
			break
		}

		var payload Payload
		err = json.Unmarshal(msg, &payload)
		if err != nil {
			replyError("", "Invalid data")
			continue
		}
		if payload.Nonce == "" {
			replyError("", "Missing Nonce")
			continue
		}

		switch payload.Op {
		case OpListInstalls:
			discords := FindDiscords()
			data := make([]DiscordData, len(discords))
			for i, d := range discords {
				in := d.(*DiscordInstall)
				data[i] = DiscordData{
					Path:       in.path,
					Branch:     in.branch,
					IsPatched:  in.isPatched,
					IsOpenAsar: in.IsOpenAsar(),
				}
			}

			reply(payload.Nonce, data)
		case OpChooseCustomInstall:
			directory, err := dialog.Directory().Title("Discord Install Directory").Browse()
			if err != nil {
				replyError(payload.Nonce, err.Error())
			} else {
				reply(payload.Nonce, directory)
			}
		case OpPatch, OpUnpatch, OpRepair, OpInstallOpenAsar, OpUninstallOpenAsar:
			var path string
			err := json.Unmarshal(payload.Data, &path)
			if err != nil {
				replyError(payload.Nonce, "Expected data to be string")
				break
			}

			discordInstall := ParseDiscord(path, "")
			if discordInstall == nil {
				replyError(payload.Nonce, "Not a valid Discord install: "+path)
				break
			}

			switch payload.Op {
			case OpPatch:
				err = discordInstall.patch()
			case OpUnpatch:
				err = discordInstall.unpatch()
			case OpRepair:
				err := InstallLatestBuilds()
				if err == nil {
					err = discordInstall.patch()
				}
			case OpInstallOpenAsar:
				err = discordInstall.InstallOpenAsar()
			case OpUninstallOpenAsar:
				err = discordInstall.UninstallOpenAsar()
			}

			if err == nil {
				reply(payload.Nonce, nil)
			} else {
				replyError(payload.Nonce, err.Error())
			}
		default:
			replyError(payload.Nonce, "Unknown OP '"+payload.Op+"'")
		}
	}
}

func main() {
	http.HandleFunc("/launch", launch)

	http.ListenAndServe("localhost:18281", nil)
}

func InstallLatestBuilds() error {
	return installLatestBuilds()
}

func HandleScuffedInstall() {
	// TODO
}
