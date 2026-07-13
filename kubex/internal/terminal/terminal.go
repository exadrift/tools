package terminal

import (
	"os"
	"os/exec"

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

	ptmx.Write([]byte("alias k='kubectl'\nclear\n"))

	t.pty = ptmx

	go func() {
		buf := make([]byte, 4096)

		for {
			n, err := ptmx.Read(buf)
			if err != nil {
				return
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
		return tcell.ColorRed
	case vt10x.Green:
		return tcell.ColorGreen
	case vt10x.Yellow:
		return tcell.ColorYellow
	case vt10x.Blue:
		return tcell.ColorBlue
	case vt10x.Magenta:
		return tcell.ColorDarkMagenta
	case vt10x.Cyan:
		return tcell.ColorTeal
	case vt10x.White:
		return tcell.ColorWhite
	case vt10x.DefaultFG:
		return tcell.ColorDefault
	default:
		return tcell.ColorDefault
	}
}

func (t *Terminal) Draw(screen tcell.Screen) {
	t.Box.DrawForSubclass(screen, t)

	x, y, w, h := t.GetInnerRect()

	// Keep vt10x matching the widget size.
	if w != t.cols || h != t.rows {
		t.cols = w
		t.rows = h

		t.vt.Resize(w, h)

		if t.pty != nil {
			pty.Setsize(t.pty, &pty.Winsize{
				Cols: uint16(w),
				Rows: uint16(h),
			})
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
}

func (t *Terminal) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return t.WrapInputHandler(func(ev *tcell.EventKey, setFocus func(p tview.Primitive)) {

		switch ev.Key() {

		case tcell.KeyRune:
			t.pty.Write([]byte(string(ev.Rune())))

		case tcell.KeyEnter:
			t.pty.Write([]byte("\r"))

		case tcell.KeyTab:
			t.pty.Write([]byte("\t"))

		case tcell.KeyBackspace, tcell.KeyBackspace2:
			t.pty.Write([]byte{0x7f})

		case tcell.KeyUp:
			t.pty.Write([]byte("\x1b[A"))

		case tcell.KeyDown:
			t.pty.Write([]byte("\x1b[B"))

		case tcell.KeyLeft:
			t.pty.Write([]byte("\x1b[D"))

		case tcell.KeyRight:
			t.pty.Write([]byte("\x1b[C"))
		}
	})
}
