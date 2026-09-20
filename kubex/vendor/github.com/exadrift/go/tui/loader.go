package tui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/exadrift/go/ansi/style"
)

var LoaderImages = []string{"⣷", "⣯", "⣟", "⡿", "⢿", "⣻", "⣽", "⣾"}

type Loader struct {
	*Container
	Label      *style.Text
	lock       sync.Mutex
	isBusyChan chan struct{}
}

func NewLoader() *Loader {
	return &Loader{
		Container: NewContainer(),
	}
}

func (l *Loader) SetLabel(label any) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.Label = ProcessToStyledText(label)
}

func (l *Loader) Render(contentWindow *ContentWindow, focusItem Widget) {
	l.backgroundStyle = nil
	l.focusBackgroundStyle = nil
	l.lock.Lock()
	defer l.lock.Unlock()

	dimensions := l.GetDimensions()
	width := l.Label.Len()

	rendered, _ := l.Label.Render()
	rows := []*style.Text{
		style.T(strings.Repeat(" ", width+6)),
		style.T(" ", strings.Repeat("█", width+4), " "),
		style.T(" █", strings.Repeat(" ", width+2), "█ "),
		style.T(" █ ", rendered, " █ "),
		style.T(" █", strings.Repeat(" ", width+2), "█ "),
		style.T(" ", strings.Repeat("█", width+4), " "),
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
func (l *Loader) Show(label string) {
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
				l.SetLabel(fmt.Sprintf("%s %s", LoaderImages[loaderFrame], label))
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
