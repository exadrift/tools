package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/exadrift/tools/kubex/internal/display"
	"github.com/exadrift/tools/kubex/internal/terminal"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
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

	gridView := tview.NewGrid()
	gridView.SetRows(0).SetColumns(30, 30, 0).SetBorder(false)

	contextTable := tview.NewTable()
	contextTable.SetBorder(true).SetTitleAlign(tview.AlignLeft).SetTitle("context")
	contextTable.SetSelectable(true, true)

	namespaceTable := tview.NewTable()
	namespaceTable.SetBorder(true).SetTitleAlign(tview.AlignLeft).SetTitle("namespace")
	namespaceTable.SetSelectable(true, true)

	if err := display.InitializeDisplay(contextTable, namespaceTable); err != nil {
		log.Fatal(err)
	}

	app := tview.NewApplication()

	term := terminal.New(app, 80, 25)

	contextTable.SetSelectedFunc(func(row, column int) {
		if err := display.UpdateContextSelection(row, namespaceTable); err != nil {
			log.Fatal(err)
		}

		app.SetFocus(namespaceTable)
	})

	namespaceTable.SetSelectedFunc(func(row, column int) {
		if err := display.UpdateNamespaceSelection(row); err != nil {
			log.Fatal(err)
		}

		app.SetFocus(term)
	})

	gridView.AddItem(contextTable, 0, 0, 1, 1, 0, 0, true)
	gridView.AddItem(namespaceTable, 0, 1, 1, 1, 0, 0, true)
	gridView.AddItem(term, 0, 2, 1, 1, 0, 0, true)

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}
	cmd := exec.Command(shell)
	if err := term.Start(cmd); err != nil {
		log.Fatal(err)
	}

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyBacktab:
			if contextTable.HasFocus() {
				app.SetFocus(term)
			} else if namespaceTable.HasFocus() {
				app.SetFocus(contextTable)
			} else {
				app.SetFocus(namespaceTable)
			}
		case tcell.KeyTab:
			if contextTable.HasFocus() {
				app.SetFocus(namespaceTable)
			} else if namespaceTable.HasFocus() {
				app.SetFocus(term)
			} else {
				app.SetFocus(contextTable)
			}
		default:
			return event
		}

		return nil
	})

	if err := app.SetRoot(gridView, true).SetFocus(term).Run(); err != nil {
		log.Fatal(err)
	}
}
