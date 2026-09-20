package tui

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/creack/pty"
	"github.com/exadrift/go/ansi/style"
	"github.com/exadrift/go/tui/internal/terminal"
	"github.com/exadrift/vt10x"
)

type Shell struct {
	*Container
	term       vt10x.Terminal
	ptyFile    *os.File
	renderChan chan string

	// offset where 0 is the present moment in time, and anything < 0 is scrolled up by n lines
	scrollOffset int
}

func NewShell() *Shell {
	return &Shell{
		Container:  NewContainer(),
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
			app.RequestRedrawComponent(RedrawRequest{
				Widget: s,
			})
		}
	}()

	return nil
}

func (s *Shell) Render(contentWindow *ContentWindow, focusItem Widget) {
	historyLength := s.term.HistoryBufferLength()
	if historyLength < s.GetContentDimensions().Height {
		historyLength = s.GetContentDimensions().Height
	}

	cw := NewContentWindow(0, s.GetContentDimensions(), WithContentWindowRowLoadCallback(historyLength, func(cw *ContentWindow) []*style.Text {
		termWidth, termHeight := s.term.Size()
		if cw.ContentDimensions.Width != termWidth || cw.ContentDimensions.Height != termHeight {
			s.term.Resize(cw.ContentDimensions.Width, cw.ContentDimensions.Height)
			if err := pty.Setsize(s.ptyFile, &pty.Winsize{
				Cols: uint16(cw.ContentDimensions.Width),
				Rows: uint16(cw.ContentDimensions.Height),
			}); err != nil {
				panic(err)
			}
		}

		// this scrollOffset always should be <= 0
		zeroScroll := historyLength - cw.ContentDimensions.Height
		scrollPosition := zeroScroll + s.scrollOffset
		if scrollPosition < 0 {
			s.scrollOffset -= scrollPosition
			scrollPosition = 0
		}
		cw.ScrollPosition = scrollPosition

		if s.scrollOffset == 0 {
			return s.term.TextRows()
		}

		return s.term.History(s.scrollOffset)
	}))
	s.Container.Render(cw, focusItem)

	if s == focusItem && s.scrollOffset == 0 {
		// if shell is in focus, place the cursor at the location
		cur := s.term.Cursor()
		terminal.SetCursorPos(cw.ContentDimensions.Left+cur.X, cw.ContentDimensions.Top+cur.Y)
		terminal.ShowCursor()
	}
}

func (s *Shell) CaptureInput(r string) string {
	switch r {
	case CtrlPgUp:
		s.scrollOffset -= s.dimensions.Height / 2
	case CtrlPgDn:
		s.scrollOffset += s.dimensions.Height / 2
		if s.scrollOffset > 0 {
			s.scrollOffset = 0
		}
	default:
		s.scrollOffset = 0
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
