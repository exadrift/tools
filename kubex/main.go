package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/exadrift/go/tui"
	"github.com/exadrift/tools/kubex/internal/display"
	"github.com/exadrift/tools/kubex/internal/kubectl"
)

var Version = ""

var shellList = []string{
	"/bin/zsh",
	"/bin/bash",
	"/bin/sh",
}

func findShell() (string, error) {
	shellBin := os.Getenv("SHELL")
	if shellBin != "" {
		return shellBin, nil
	}

	for _, shell := range shellList {
		_, err := os.Stat(shell)
		if err == nil {
			return shell, nil
		}
	}

	return "", fmt.Errorf("unable to locate command shell binary")
}

func main() {
	for _, arg := range os.Args {
		if arg == "--help" {
			fmt.Println("kubex - kubernetes explorer")
			fmt.Println("help:")
			fmt.Println()
			fmt.Println("  k                   - invoke kubectl (terminal alias)")
			fmt.Println("  <tab> / <shift-tab> - change focus through panes right / left")
			fmt.Println("  <ctrl> + <p>        - execute command at prompt and send output to vi")
			os.Exit(0)
		}

		if arg == "--version" {
			fmt.Printf("%s\n", Version)
			os.Exit(0)
		}
	}

	if !kubectl.KubectlInPath() {
		log.Fatalf("unable to locate kubectl, please make sure it's in your execution path")
	}

	shellBin, err := findShell()
	if err != nil {
		log.Fatal(err)
	}

	contextMenu := tui.NewMenu()
	contextMenu.EnableBorder(true).SetTitle("context")

	namespaceMenu := tui.NewMenu()
	namespaceMenu.EnableBorder(true).SetTitle("namespace")

	if err := display.InitializeDisplay(contextMenu, namespaceMenu); err != nil {
		log.Fatal(err)
	}

	shell := tui.NewShell()
	shell.EnableBorder(true).SetTitle("terminal")

	layout := tui.NewFlexLayout(
		tui.OrientationHorizontal,
		tui.NewSegment(1, contextMenu),
		tui.NewSegment(1, namespaceMenu),
		tui.NewSegment(3, shell),
	)

	app := tui.New(layout).SetFocus(shell)

	contextMenu.SetSelectHandler(func(selectedIndex int, selectedItem string) {
		if err := display.UpdateContextSelection(selectedItem, namespaceMenu); err != nil {
			log.Fatal(err)
		}

		app.SetFocus(namespaceMenu)
	})

	namespaceMenu.SetSelectHandler(func(selectedIndex int, selectedItem string) {
		if err := display.UpdateNamespaceSelection(selectedItem); err != nil {
			log.Fatal(err)
		}

		app.SetFocus(shell)
	})

	cmd := exec.Command(shellBin)
	if err := shell.Start(app, cmd); err != nil {
		log.Fatal(err)
	}

	shell.CaptureInput("alias k='kubectl'\nclear\n")

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
