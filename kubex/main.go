package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/exadrift/go/ansi"
	"github.com/exadrift/go/tui"
	"github.com/exadrift/tools/kubex/internal/config"
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
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	for _, arg := range os.Args {
		if arg == "--help" {
			fmt.Println("kubex - kubernetes explorer")
			fmt.Println("  --help   - display help")
			fmt.Println("  --keys   - display bindable key combinations (exclude basic keys)")
			fmt.Println("  --config - edit the configuration (key bindings, etc.)")
			fmt.Println()
			fmt.Println("tui help:")
			fmt.Printf("  k                   - invoke kubectl (terminal alias)\n")
			fmt.Printf("  %s / %s - change focus through panes right / left\n", cfg.KeyBindings.NavPrev.HumanName, cfg.KeyBindings.NavNext.HumanName)
			os.Exit(0)
		}

		if arg == "--keys" {
			for _, keyCombo := range ansi.AllKeys {
				if len(keyCombo.HumanName) > 1 {
					fmt.Printf("%s\n", keyCombo.HumanName)
				}
			}
			os.Exit(0)
		}

		if arg == "--config" {
			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = "vi"
			}
			fmt.Printf("attempting to edit configs at %s using %s\n", cfg.Location, editor)

			cmd := exec.Command(editor, cfg.Location)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Run()
			if err != nil {
				log.Fatal(err)
			}
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

	bindings := tui.NewKeyBindings()
	bindings.FocusNext = cfg.KeyBindings.NavNext.Ansi
	bindings.FocusPrev = cfg.KeyBindings.NavPrev.Ansi
	bindings.SelectionPrev = cfg.KeyBindings.Up.Ansi
	bindings.SelectionNext = cfg.KeyBindings.Down.Ansi
	bindings.ScrollUp = cfg.KeyBindings.ScrollUp.Ansi
	bindings.ScrollDown = cfg.KeyBindings.ScrollDown.Ansi
	app := tui.New(layout, *tui.WithApplicationOptionKeyBindings(bindings)).SetFocus(shell)

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
