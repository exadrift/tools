package tui

import (
	"os"
	"os/signal"
	"slices"
	"sync"
	"syscall"

	"github.com/exadrift/go/tui/internal/terminal"
	"golang.org/x/term"
)

type Application struct {
	errorCond    error
	redrawChan   chan Widget
	inputHandler func(string) string
	inFocus      Widget
	ctrlCExit    bool
	termFd       int
	onStart      func() error
	onExit       func()
	sigChan      chan os.Signal
	closeLock    sync.Mutex
	closeChan    chan struct{}
	wg           *sync.WaitGroup
	root         Widget
}

func WithApplicationOptionInputHandler(handleInput func(string) string) *Option {
	return &Option{
		optionType: ApplicationOptionInputHandler,
		data:       handleInput,
	}
}

func WithApplicationOptionOnStart(onStart func() error) *Option {
	return &Option{
		optionType: ApplicationOptionWithOnStart,
		data:       onStart,
	}
}

func WithApplicationOptionOnExit(onExit func()) *Option {
	return &Option{
		optionType: ApplicationOptionWithOnExit,
		data:       onExit,
	}
}

func WithApplicationOptionExitSignals(signals ...syscall.Signal) *Option {
	exitSignals := make([]syscall.Signal, len(signals))
	copy(exitSignals, signals)
	return &Option{
		optionType: ApplicationOptionExitSignals,
		data:       exitSignals,
	}
}

// New constructs and returns a new Application with a root widget specified.
func New(root Widget, options ...Option) *Application {
	var exitSignals []os.Signal

	app := &Application{
		wg:         &sync.WaitGroup{},
		root:       root,
		sigChan:    make(chan os.Signal, 1),
		closeChan:  make(chan struct{}, 1),
		redrawChan: make(chan Widget, 1000),
	}

	for _, option := range options {
		switch option.optionType {
		case ApplicationOptionWithOnStart:
			app.onStart = option.data.(func() error)
		case ApplicationOptionWithOnExit:
			app.onExit = option.data.(func())
		case ApplicationOptionExitSignals:
			exitSignals = option.data.([]os.Signal)
		case ApplicationOptionInputHandler:
			app.inputHandler = option.data.(func(string) string)
		default:
			panic("unknown application option")
		}
	}

	if exitSignals == nil {
		exitSignals = []os.Signal{syscall.SIGINT, syscall.SIGTERM}
	}
	if slices.Contains(exitSignals, os.Signal(syscall.SIGINT)) {
		app.ctrlCExit = true
	}
	signal.Notify(app.sigChan, exitSignals...)

	return app
}

func (a *Application) Exit(err error) {
	a.closeLock.Lock()
	defer a.closeLock.Unlock()

	if err != nil {
		a.errorCond = err
	}

	select {
	case _, ok := <-a.closeChan:
		if !ok {
			return
		}

	default:
		close(a.closeChan)
	}
}

// GetWaitGroup returns the underlying wait group, allowing adjacent threads to adjust the application's wait group,
// allowing for clean thread syncrhonization on exit
func (a *Application) GetWaitGroup() *sync.WaitGroup {
	return a.wg
}

func (a *Application) SetFocus(obj Widget) *Application {
	a.inFocus = obj
	return a
}

func (a *Application) GetFocalWidgets() []Widget {
	focalWidgets := &FocalWidgets{}
	a.root.GetFocalWidgets(a.root, focalWidgets)
	return focalWidgets.Widgets
}

func (a *Application) findFocusNext() Widget {
	focalWidgets := a.GetFocalWidgets()
	for i, widget := range focalWidgets {
		if widget == a.inFocus {
			if i < len(focalWidgets)-1 {
				return focalWidgets[i+1]
			}

			break
		}
	}
	return focalWidgets[0]
}

func (a *Application) findFocusPrev() Widget {
	focalWidgets := a.GetFocalWidgets()
	for i, widget := range focalWidgets {
		if widget == a.inFocus {
			if i > 0 {
				return focalWidgets[i-1]
			}

			break
		}
	}
	return focalWidgets[len(focalWidgets)-1]
}

// handleInput directs input to the widget with focus
func (a *Application) handleInput(input string) string {
	if a.inFocus == nil {
		return input
	}
	return a.inFocus.CaptureInput(input)
}

func (a *Application) renderAll() {
	a.renderWidgets(RenderModeAll, a.root.Collect(a.root)...)
}

func (a *Application) renderAllRefocus() {
	a.renderWidgets(RenderModeBorder, a.root.Collect(a.root)...)
}

func (a *Application) renderFocused() {
	a.renderWidgets(RenderModeContent, a.inFocus)
}

func (a *Application) RequestRedrawComponent(component Widget) {
	a.redrawChan <- component
}

func (a *Application) renderWidgets(renderMode RenderMode, widgets ...Widget) {
	terminal.HideCursor()
	for _, w := range widgets {
		if w == a.inFocus {
			continue
		}

		w.Render(renderMode, a.inFocus)
	}

	if a.inFocus != nil {
		a.inFocus.Render(renderMode, a.inFocus)
	}
}

// Start starts the application loop
func (a *Application) Start() error {
	defer a.wg.Wait()

	// this exit will signal prior to the waitgroup wait function, allowing any blocking threads
	// an opportunity to gracefully exit
	defer a.Exit(nil)
	// main application loop

	termFd, restore, err := terminal.Start()
	if err != nil {
		return err
	}
	a.termFd = termFd
	defer restore()
	defer terminal.Clear()

	if a.onExit != nil {
		defer a.onExit()
	}

	if a.onStart != nil {
		if err := a.onStart(); err != nil {
			return err
		}
	}

	inputChan := make(chan string, 100)
	byteBuf := make([]byte, 100)
	go func() {
		defer close(inputChan)
		for {
			nBytes, err := os.Stdin.Read(byteBuf)
			if err != nil {
				return
			}

			inputChan <- string(byteBuf[:nBytes])
		}
	}()

	termSizeChan := make(chan os.Signal, 100)
	signal.Notify(termSizeChan, syscall.SIGWINCH)

	width, height, err := term.GetSize(a.termFd)
	if err != nil {
		return err
	}
	a.root.SetDimensions(0, 0, width, height)
	a.renderAll()

	// application event loop
	for {
		select {
		case <-a.sigChan:
			// process received a signal to exit
			return nil
		case <-a.closeChan:
			// application is being closed internally
			return a.errorCond
		case <-termSizeChan:
			// resize event occurred
			width, height, err := term.GetSize(a.termFd)
			if err != nil {
				return err
			}
			a.root.SetDimensions(0, 0, width, height)
			a.renderAll()
		case widget := <-a.redrawChan:
			widget.Render(RenderModeContent, a.inFocus)
		case c := <-inputChan:
			switch c {
			case Tab:
				if a.inFocus != nil {
					a.SetFocus(a.findFocusNext())
					a.renderAllRefocus()
				}
			case ShiftTab:
				if a.inFocus != nil {
					a.SetFocus(a.findFocusPrev())
					a.renderAllRefocus()
				}
			default:
				// this is to allow sending control+c to the shell
				if c == CtrlC && a.ctrlCExit {
					if a.inFocus == nil || !a.inFocus.AbsorbsInput(CtrlC) {
						return nil
					}
				}

				if a.inputHandler != nil {
					c = a.inputHandler(c)
					if c == RenderFullCode {
						a.renderAll()
						continue
					}
				}

				c = a.handleInput(c)
				if c == RenderFullCode {
					a.renderAll()
				} else {
					// Render just the in-focus thing
					a.renderFocused()
				}
			}
		}
	}
}
