package tui

import (
	"fmt"
	"sync"
	"time"

	"github.com/exadrift/go/tui/internal/terminal"
)

var LoaderImages = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type Loader struct {
	*Box
	Label      string
	lock       sync.Mutex
	isBusyChan chan struct{}
}

func NewLoader() *Loader {
	return &Loader{
		Box: NewBox(),
	}
}

func (l *Loader) SetLabel(label string) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.Label = label
}

func (l *Loader) Render(mode RenderMode, focusItem Widget) {
	l.lock.Lock()
	defer l.lock.Unlock()

	width := len(l.Label) + 4
	height := 7

	left := int((float64(l.dimensions.Width) / 2.) - (float64(width) / 2.))
	top := int((float64(l.dimensions.Height) / 2.) - (float64(height) / 2.))

	curY := top

	terminal.SetCursorPos(left, curY)
	fmt.Print(StyleReset)
	for i := 0; i < width; i++ {
		fmt.Print(" ")
	}
	curY++

	terminal.SetCursorPos(left, curY)
	fmt.Print(StyleReset)
	fmt.Print(" ")
	fmt.Print(StyleFg(Blue))
	for i := 1; i < width-1; i++ {
		fmt.Print("█")
	}
	fmt.Print(StyleReset)
	fmt.Print(" ")
	curY++

	terminal.SetCursorPos(left, curY)
	fmt.Print(StyleReset)
	fmt.Print(" ")
	fmt.Print(StyleFg(Blue))
	fmt.Print("█")
	fmt.Print(StyleReset)
	for i := 2; i < width-2; i++ {
		fmt.Print(" ")
	}
	fmt.Print(StyleFg(Blue))
	fmt.Print("█")
	fmt.Print(StyleReset)
	fmt.Print(" ")
	curY++

	terminal.SetCursorPos(left, curY)
	fmt.Print(StyleReset)
	fmt.Print(" ")
	fmt.Print(StyleFg(Blue))
	fmt.Print("█")
	fmt.Print(StyleFg(Yellow))
	fmt.Printf(" %s ", l.Label)
	fmt.Print(StyleFg(Blue))
	fmt.Print("█")
	fmt.Print(StyleReset)
	fmt.Print(" ")
	curY++

	terminal.SetCursorPos(left, curY)
	fmt.Print(StyleReset)
	fmt.Print(" ")
	fmt.Print(StyleFg(Blue))
	fmt.Print("█")
	fmt.Print(StyleReset)
	for i := 2; i < width-2; i++ {
		fmt.Print(" ")
	}
	fmt.Print(StyleFg(Blue))
	fmt.Print("█")
	fmt.Print(StyleReset)
	fmt.Print(" ")
	curY++

	terminal.SetCursorPos(left, curY)
	fmt.Print(StyleReset)
	fmt.Print(" ")
	fmt.Print(StyleFg(Blue))
	for i := 1; i < width-1; i++ {
		fmt.Print("█")
	}
	fmt.Print(StyleReset)
	fmt.Print(" ")
	curY++

	terminal.SetCursorPos(left, curY)
	fmt.Print(StyleReset)
	for i := 0; i < width; i++ {
		fmt.Print(" ")
	}
}

// SetBusy sets the busy status on the application component.  If becoming busy, a thread will be started with a UI
// timer to set render events, if becoming not busy, the timer will stop
func (l *Loader) Show(label string) {
	l.isBusyChan = make(chan struct{}, 1)
	a := appSingleton
	a.loader.Label = label

	// Start the load timer thread
	go func() {
		loaderFrame := 0
		ticker := time.NewTicker(LoaderTickInterval)
		defer ticker.Stop()
		for {
			select {
			case <-l.isBusyChan:
				a.RequestRedrawComponent(RedrawRequest{
					Widget:     nil,
					RenderMode: RenderModeAll,
				})
				return
			case <-ticker.C:
				// update the label and send a redraw request
				a.loader.SetLabel(fmt.Sprintf("%s %s", LoaderImages[loaderFrame], label))
				a.RequestRedrawComponent(RedrawRequest{
					Widget:     a.loader,
					RenderMode: RenderModeAll,
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
