package tui

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/creack/pty"
	"github.com/exadrift/go/tui/internal/terminal"
	"github.com/exadrift/vt10x"
)

type Shell struct {
	*Box
	term         vt10x.Terminal
	ptyFile      *os.File
	renderChan   chan string
	scrollOffest int
}

func NewShell() *Shell {
	return &Shell{
		Box:        NewBox().EnableScrollHandle(true),
		term:       vt10x.New(vt10x.WithSize(40, 25)),
		renderChan: make(chan string, 1000),
	}
}

func (s *Shell) Start(app *Application, cmd *exec.Cmd) error {
	ptyFile, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("unable to strat pseudo tty: %w", err)
	}

	s.ptyFile = ptyFile

	go func() {
		buf := make([]byte, 4096)

		for {
			n, err := ptyFile.Read(buf)
			if err != nil {
				if errors.Is(err, syscall.EIO) {
					// if we've received an EOF, exit
					app.Exit(nil)
					return
				}

				app.Exit(err)
				return
			}

			// Update terminal state outside the tview event loop.
			_, _ = s.term.Write(buf[:n])

			// Enqueue a redraw request
			app.RequestRedrawComponent(s)
		}
	}()

	return nil
}

func (s *Shell) Render(mode RenderMode, focusItem Widget) {
	contentDims := s.GetContentDimensions()
	termWidth, termHeight := s.term.Size()

	if contentDims.Width != termWidth || contentDims.Height != termHeight {
		s.term.Resize(contentDims.Width, contentDims.Height)
		termWidth, termHeight = s.term.Size()
		if err := pty.Setsize(s.ptyFile, &pty.Winsize{
			Cols: uint16(termWidth),
			Rows: uint16(termHeight),
		}); err != nil {
			panic(err)
		}
	}

	historyLength := s.term.HistoryBufferLength()

	zeroScroll := historyLength - contentDims.Height
	if zeroScroll+s.scrollOffest < 0 {
		s.scrollOffest = 0 - zeroScroll
	}

	s.scrollWindow.scrollPosition = zeroScroll + s.scrollOffest
	if s.scrollOffest == 0 {
		s.RenderWithScroll(mode, focusItem, historyLength, -1, nil)
		for y, ansiRow := range s.term.AnsiRows() {
			terminal.SetCursorPos(contentDims.Left, contentDims.Top+y)
			fmt.Print(ansiRow)
		}
	} else {
		hist := s.term.History(0 + s.scrollOffest)
		s.RenderWithScroll(mode, focusItem, historyLength, -1, func(index int) string {
			// index here is going to be based from the beginning of history, so we need to account for that by subtracting the scroll position
			// the history buffer width could be different from the current terminal width, and thus we must constrain the width
			return ConstrainAnsiFullWidth(hist[index-s.scrollWindow.scrollPosition], contentDims.Width)
		})
	}

	if s == focusItem {
		// if shell is in focus, place the cursor at the location
		cur := s.term.Cursor()
		terminal.SetCursorPos(contentDims.Left+cur.X, contentDims.Top+cur.Y)
		terminal.ShowCursor()
	}
}

func (s *Shell) CaptureInput(r string) string {
	switch r {
	case CtrlPgUp:
		s.scrollOffest -= s.dimensions.Height / 2
	case CtrlPgDn:
		s.scrollOffest += s.dimensions.Height / 2
		if s.scrollOffest > 0 {
			s.scrollOffest = 0
		}
	default:
		s.scrollOffest = 0
		_, _ = s.ptyFile.Write([]byte(r))
	}

	return ""
}

func (s *Shell) AbsorbsInput(input string) bool {
	switch input {
	case CtrlC:
		// we don't want the ctrl+c to cause an abort
		return true
	default:
		return false
	}
}
