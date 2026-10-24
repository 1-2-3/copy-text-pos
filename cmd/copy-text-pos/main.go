//go:build windows

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"copytextpos/internal/app"
	"copytextpos/internal/version"
	"copytextpos/internal/winapi"
)

func main() {
	configPath := flag.String("config", "", "path to config.json (default: %APPDATA%\\CopyTextPos\\config.json)")
	debug := flag.Bool("debug", false, "log to stderr as well as the log file")
	showVersion := flag.Bool("version", false, "print version information and exit")
	flag.Parse()

	winapi.SetDPIAware()

	if *showVersion {
		fmt.Println(version.Full())
		return
	}
	if *configPath != "" {
		if abs, err := filepath.Abs(*configPath); err == nil {
			*configPath = abs
		}
	}

	mutex, err := winapi.CreateSingleInstanceMutex(`Local\CopyTextPos.Singleton`)
	if err != nil {
		messageBoxError(err.Error())
		return
	}
	defer mutex.Close()
	if mutex.Already {
		// A second launch acts as "open settings" for the running instance.
		for i := 0; i < 25; i++ {
			hwnd := winapi.FindWindow("CopyTextPos.HostWindow", "")
			if hwnd != 0 {
				winapi.SendMessage(hwnd, winapi.WmAppShowSettings, 0, 0)
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
		return
	}

	application, err := app.New(*configPath, *debug)
	if err != nil {
		messageBoxError(err.Error())
		return
	}
	if err := application.Run(); err != nil {
		messageBoxError(err.Error())
	}
}

func messageBoxError(message string) {
	_, _ = os.Stderr.WriteString(message + "\n")
	winapi.MessageBox("Copy Text Pos", message, 0x10) // MB_ICONERROR
}
