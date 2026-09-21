package tui

import (
	"strings"
	"sync"
	"time"

	"github.com/exadrift/go/ansi/style"
)

var LoaderImages = []string{"⣷", "⣯", "⣟", "⡿", "⢿", "⣻", "⣽", "⣾"}

type Loader struct {
	*Container
	Label         *style.Text
	lock          sync.Mutex
	isBusyChan    chan struct{}
	borderStyles  style.Styles
	spinnerStyles style.Styles
}

func NewLoader() *Loader {
	l := &Loader{
		Container: NewContainer(),
	}
	l.SetBackgroundStyle(nil)
	l.SetFocusedBackgroundStyle(nil)
	return l
}

func (l *Loader) SetLabel(label any) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.Label = ProcessToStyledText(label)
}

func (l *Loader) SetBorderStyles(styles ...*style.Style) *Loader {
	l.borderStyles = styles
	return l
}

func (l *Loader) SetSpinnerStyles(styles ...*style.Style) *Loader {
	l.spinnerStyles = styles
	return l
}

func (l *Loader) Render(contentWindow *ContentWindow, focusItem Widget) {
	l.lock.Lock()
	defer l.lock.Unlock()

	dimensions := l.GetDimensions()
	width := l.Label.Len()

	rows := []*style.Text{
		style.T(style.S(strings.Repeat(" ", width+6), l.borderStyles...)),
		style.T(" ", style.S(strings.Repeat("█", width+4), l.borderStyles...), " "),
		style.T(style.S(" █", l.borderStyles...), strings.Repeat(" ", width+2), style.S("█ ", l.borderStyles...)),
		style.T(style.S(" █ ", l.borderStyles...), l.Label, style.S(" █ ", l.borderStyles...)),
		style.T(style.S(" █", l.borderStyles...), strings.Repeat(" ", width+2), style.S("█ ", l.borderStyles...)),
		style.T(" ", style.S(strings.Repeat("█", width+4), l.borderStyles...), " "),
		style.T(strings.Repeat(" ", width+6)),
	}

	cw := NewContentWindow(
		0,
		dimensions,
		WithFixedPositionRows(
			int((float64(l.dimensions.Width)/2.)-(float64(width)/2.)),
			int((float64(l.dimensions.Height)/2.)-(float64(len(rows))/2.)),
			rows...,
		),
	)

	l.Container.Render(cw, focusItem)
}

// SetBusy sets the busy status on the application component.  If becoming busy, a thread will be started with a UI
// timer to set render events, if becoming not busy, the timer will stop
func (l *Loader) Show(label any) {
	l.isBusyChan = make(chan struct{}, 1)
	a := appSingleton

	// Start the load timer thread
	go func() {
		loaderFrame := 0
		ticker := time.NewTicker(LoaderTickInterval)
		defer ticker.Stop()
		for {
			select {
			case <-l.isBusyChan:
				a.RequestRedrawComponent(RedrawRequest{
					Widget: nil,
				})
				return
			case <-ticker.C:
				// update the label and send a redraw request
				l.SetLabel(style.T(style.S(LoaderImages[loaderFrame], l.spinnerStyles...), " ", label))
				a.RequestRedrawComponent(RedrawRequest{
					Widget: l,
				})
				loaderFrame++
				if loaderFrame >= len(LoaderImages) {
					loaderFrame = 0
				}
			}
		}
	}()
}

func (l *Loader) Hide() {
	close(l.isBusyChan)
}
