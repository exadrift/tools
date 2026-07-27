package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/exadrift/go/tui"
	"github.com/exadrift/tools/kubex/internal/display"
)

var Version = ""

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

	shellBin := os.Getenv("SHELL")
	if shellBin == "" {
		shellBin = "/bin/bash"
	}
	cmd := exec.Command(shellBin)
	if err := shell.Start(app, cmd); err != nil {
		log.Fatal(err)
	}

	shell.CaptureInput("alias k='kubectl'\nclear\n")

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
