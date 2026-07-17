package terminal

import (
	"errors"
	"log"
	"os"
	"os/exec"
	"syscall"

	"github.com/creack/pty"
	"github.com/gdamore/tcell/v2"
	"github.com/hinshun/vt10x"
	"github.com/rivo/tview"
)

type Terminal struct {
	*tview.Box

	app *tview.Application

	vt   vt10x.Terminal
	pty  *os.File
	cols int
	rows int
}

func New(app *tview.Application, cols, rows int) *Terminal {
	term := vt10x.New(vt10x.WithSize(cols, rows))

	return &Terminal{
		Box: tview.NewBox().
			SetBorder(true).
			SetTitle("terminal"),
		app:  app,
		vt:   term,
		cols: cols,
		rows: rows,
	}
}

func (t *Terminal) Start(cmd *exec.Cmd) error {
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return err
	}

	if _, err = ptmx.Write([]byte("alias k='kubectl'\nclear\n")); err != nil {
		return err
	}

	t.pty = ptmx

	go func() {
		buf := make([]byte, 4096)

		for {
			n, err := ptmx.Read(buf)
			if err != nil {
				if errors.Is(err, syscall.EIO) {
					// if we've received an EOF, exit
					t.app.Stop()
					return
				}

				log.Fatal(err)
			}

			// Update terminal state outside the tview event loop.
			_, _ = t.vt.Write(buf[:n])

			// Ask tview to redraw.
			t.app.QueueUpdateDraw(func() {})
		}
	}()

	return nil
}

func convertColor(c vt10x.Color) tcell.Color {
	switch c {
	case vt10x.Black:
		return tcell.ColorBlack
	case vt10x.Red:
		return tcell.ColorMaroon
	case vt10x.Green:
		return tcell.ColorGreen
	case vt10x.Yellow:
		return tcell.ColorOlive
	case vt10x.Blue:
		return tcell.ColorNavy
	case vt10x.Magenta:
		return tcell.ColorPurple
	case vt10x.Cyan:
		return tcell.ColorTeal
	case vt10x.LightGrey:
		return tcell.ColorSilver
	case vt10x.DarkGrey:
		return tcell.ColorGray
	case vt10x.LightRed:
		return tcell.ColorRed
	case vt10x.LightGreen:
		return tcell.ColorLime
	case vt10x.LightYellow:
		return tcell.ColorYellow
	case vt10x.LightBlue:
		return tcell.ColorBlue
	case vt10x.LightMagenta:
		return tcell.ColorFuchsia
	case vt10x.LightCyan:
		return tcell.ColorAqua
	case vt10x.White:
		return tcell.ColorWhite
	default:
		return tcell.ColorDefault
	}
}

func (t *Terminal) Draw(screen tcell.Screen) {
	t.DrawForSubclass(screen, t)

	x, y, w, h := t.GetInnerRect()

	// Keep vt10x matching the widget size.
	if w != t.cols || h != t.rows {
		t.cols = w
		t.rows = h

		t.vt.Resize(w, h)

		if t.pty != nil {
			if err := pty.Setsize(t.pty, &pty.Winsize{
				Cols: uint16(w),
				Rows: uint16(h),
			}); err != nil {
				panic(err)
			}
		}
	}

	termW, termH := t.vt.Size()

	maxW := min(w, termW)
	maxH := min(h, termH)

	for row := 0; row < maxH; row++ {
		for col := 0; col < maxW; col++ {
			cell := t.vt.Cell(col, row)

			style := tcell.StyleDefault.
				Foreground(convertColor(cell.FG)).
				Background(convertColor(cell.BG))

			screen.SetContent(
				x+col,
				y+row,
				cell.Char,
				nil,
				style,
			)
		}
	}

	cur := t.vt.Cursor()
	if t.HasFocus() {
		screen.ShowCursor(x+cur.X, y+cur.Y)
	} else {
		screen.HideCursor()
	}
}

func (t *Terminal) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return t.WrapInputHandler(func(ev *tcell.EventKey, setFocus func(p tview.Primitive)) {
		var err error

		switch ev.Key() {
		case tcell.KeyRune:
			_, err = t.pty.Write([]byte(string(ev.Rune())))

		case tcell.KeyEnter:
			_, err = t.pty.Write([]byte("\r"))

		case tcell.KeyTab:
			_, err = t.pty.Write([]byte("\t"))

		case tcell.KeyBackspace, tcell.KeyBackspace2:
			_, err = t.pty.Write([]byte{0x7f})

		case tcell.KeyUp:
			_, err = t.pty.Write([]byte("\x1b[A"))

		case tcell.KeyDown:
			_, err = t.pty.Write([]byte("\x1b[B"))

		case tcell.KeyLeft:
			_, err = t.pty.Write([]byte("\x1b[D"))

		case tcell.KeyRight:
			_, err = t.pty.Write([]byte("\x1b[C"))

		case tcell.KeyHome:
			_, err = t.pty.Write([]byte("\x1b[H"))

		case tcell.KeyEnd:
			_, err = t.pty.Write([]byte("\x1b[F"))

		case tcell.KeyPgUp:
			_, err = t.pty.Write([]byte("\x1b[5~"))

		case tcell.KeyPgDn:
			_, err = t.pty.Write([]byte("\x1b[6~"))

		case tcell.KeyDelete:
			_, err = t.pty.Write([]byte("\x1b[3~"))

		case tcell.KeyInsert:
			_, err = t.pty.Write([]byte("\x1b[2~"))

		case tcell.KeyCtrlC:
			_, err = t.pty.Write([]byte{3})

		case tcell.KeyCtrlD:
			_, err = t.pty.Write([]byte{4})

		case tcell.KeyCtrlZ:
			_, err = t.pty.Write([]byte{26})

		case tcell.KeyCtrlR:
			_, err = t.pty.Write([]byte{0x12})

		case tcell.KeyCtrlV:
			_, err = t.pty.Write([]byte("| vi -\n"))
		}

		if err != nil {
			panic(err)
		}
	})
}
