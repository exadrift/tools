package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/exadrift/go/ansi/keys"
	"github.com/exadrift/go/ansi/style"
	"github.com/exadrift/go/tui"
	"github.com/exadrift/tools/kubex/internal/config"
	"github.com/exadrift/tools/kubex/internal/kubectl"
)

var Version = "v0.0.0"

var shellList = []string{
	"/bin/zsh",
	"/bin/bash",
	"/bin/sh",
}

type Namespaces struct {
	Selected string
	All      []string
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
	fmt.Println("loading...")
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
			for _, keyCombo := range keys.AllKeys {
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
	contextMenu.SetTitle("context")

	namespaceMenu := tui.NewMenu()
	namespaceMenu.SetTitle("namespace")

	shell := tui.NewShell()
	shell.SetTitle("terminal")

	topText := style.B(
		style.T(
			style.S("kubex ", style.White.Fg()),
			style.S(Version, style.White.Fg()),
			style.S(" - ctrl+c to exit (ctrl+d to exit shell)", style.Black.Fg()),
		),
	)
	topBar := tui.NewText(topText)
	topBar.SetFocusable(false)
	topBar.SetBackgroundStyle(style.FromRgb(111, 25, 224).Bg())

	selectableLayout := tui.NewFlexLayout(
		tui.OrientationHorizontal,
		1,
		tui.NewSegment(1, contextMenu),
		tui.NewSegment(1, namespaceMenu),
		tui.NewSegment(3, shell),
	)

	layout := tui.NewFlexLayout(
		tui.OrientationVertical,
		1,
		tui.NewSegment(1, topBar, tui.WithSegmentOptionMinChars(1)),
		tui.NewSegment(1000, selectableLayout),
	)

	bindings := tui.NewKeyBindings()
	bindings.FocusNext = cfg.KeyBindings.NavNext.Ansi
	bindings.FocusPrev = cfg.KeyBindings.NavPrev.Ansi
	bindings.SelectionPrev = cfg.KeyBindings.Up.Ansi
	bindings.SelectionNext = cfg.KeyBindings.Down.Ansi
	bindings.ScrollUp = cfg.KeyBindings.ScrollUp.Ansi
	bindings.ScrollDown = cfg.KeyBindings.ScrollDown.Ansi
	app := tui.New(layout, *tui.WithApplicationOptionKeyBindings(bindings)).SetFocus(shell)

	contexts, err := kubectl.GetContexts()
	if err != nil {
		log.Fatal(err)
	}
	curContext, err := kubectl.GetCurrentContext()
	if err != nil {
		log.Fatal(err)
	}
	namespaces, err := kubectl.GetNamespaces()
	if err != nil {
		log.Fatal(err)
	}
	curNamespace, err := kubectl.GetCurrentNamespace(curContext)
	if err != nil {
		log.Fatal(err)
	}

	contextMenu.SetContents(contexts...).SetSelectedItem(curContext)
	namespaceMenu.SetContents(namespaces...).SetSelectedItem(curNamespace)

	contextMenu.SetSelectHandler(
		func(selectedIndex int, selectedItem string) any {
			err := kubectl.SetCurrentContext(selectedItem)
			if err != nil {
				app.Exit(err)
			}

			curNamespace, err := kubectl.GetCurrentNamespace(curContext)
			if err != nil {
				app.Exit(err)
			}

			namespaces, err := kubectl.GetNamespaces()
			if err != nil {
				app.Exit(err)
			}

			return Namespaces{
				Selected: curNamespace,
				All:      namespaces,
			}
		},
		tui.WithBusyModal(
			"switching context...",
			func(a any) {
				ns := a.(Namespaces)
				namespaceMenu.SetContents(ns.All...)
				namespaceMenu.SetSelectedItem(ns.Selected)
				app.SetFocus(namespaceMenu)
			},
		),
	)

	namespaceMenu.SetSelectHandler(
		func(selectedIndex int, selectedItem string) any {
			if err := kubectl.SetCurrentNamespace(selectedItem); err != nil {
				app.Exit(err)
			}
			return nil
		},
		tui.WithBusyModal(
			"switching namespace...",
			func(a any) {
				app.SetFocus(shell)
			},
		),
	)

	cmd := exec.Command(shellBin)
	if err := shell.Start(app, cmd); err != nil {
		log.Fatal(err)
	}

	shell.CaptureInput("alias k='kubectl'\nclear\n")

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
