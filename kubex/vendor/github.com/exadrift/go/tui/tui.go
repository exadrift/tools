package tui

import (
	"fmt"
	"math"
	"strings"

	"github.com/exadrift/go/ansi/style"
	"github.com/exadrift/go/tui/internal/terminal"
)

type Dimensions struct {
	Top    int
	Left   int
	Width  int
	Height int
}

type Widget interface {
	GetChildren() []Widget
	GetDimensions() *Dimensions
	Render(contentWindow *ContentWindow, focusItem Widget)
	Collect(me Widget) []Widget
	SetDimensions(left int, top int, width int, height int)
	GetContainer() *Container
	CaptureInput(r string) string
	GetFocalWidgets(me Widget, focalWidgets *FocalWidgets)
	CanHaveFocus() bool
	AbsorbsInput(input string) bool
	ResetScrollPosition()
}

var blanks = []rune(strings.Repeat(" ", 2000))

type FocalWidgets struct {
	Widgets []Widget
}

type OptionType int

const (
	SegmentOptionMinChars OptionType = iota
	SegmentOptionMaxChars
	ApplicationOptionWithOnStart
	ApplicationOptionWithOnExit
	ApplicationOptionExitSignals
	ApplicationOptionInputHandler
	ApplicationOptionKeyBindings
	BusyModal
)

type Option struct {
	optionType OptionType
	data       any
}

type KeyBindings struct {
	FocusNext     string
	FocusPrev     string
	SelectionNext string
	SelectionPrev string
	ScrollUp      string
	ScrollDown    string
	Trigger       string
}

func NewKeyBindings() *KeyBindings {
	return &KeyBindings{
		FocusNext:     Tab,
		FocusPrev:     ShiftTab,
		SelectionNext: DownArrow,
		SelectionPrev: UpArrow,
		ScrollUp:      CtrlPgUp,
		ScrollDown:    CtrlPgDn,
		Trigger:       Enter,
	}
}

// Constrain will constrain the provided value to the provided length, adding ellipsis
// to the end if possible
func Constrain(value string, length int) string {
	if len(value) <= length {
		return value
	}

	newValue := []rune(value[:length])
	last := len(newValue) - 1
	first := last - 1
	if first < 0 {
		first = 0
	}
	for i := first; i <= last; i++ {
		newValue[i] = '.'
	}

	return string(newValue)
}

func ProcessToStyledText(text any) *style.Text {
	switch t := text.(type) {
	case *style.Text:
		return t
	case []rune:
		return style.T(string(t))
	case string:
		return style.T(t)
	default:
		panic(fmt.Sprintf("unknown text type to be processed into stylized text %+v", text))
	}
}

type ContentWindowRenderOptions struct {
	RowCallback func(rowIndex int, yIndex int, width int) *style.Text
	TotalRows   int
	TextBlock   *style.TextBlock

	RowLoadCallback func(cw *ContentWindow) []*style.Text

	Rows []*style.Text
	Row  int
	Col  int
}

type ContentWindow struct {
	ScrollPosition    int
	TotalRows         int
	HasScrollbar      bool
	Options           ContentWindowRenderOptions
	ContentDimensions Dimensions
}

type ContentWindowRenderOptionFunc func(opt *ContentWindowRenderOptions)

func NewContentWindow(scrollPosition int, contentDimensions *Dimensions, opts ...ContentWindowRenderOptionFunc) *ContentWindow {
	cw := &ContentWindow{}

	for _, optFunc := range opts {
		optFunc(&cw.Options)
	}

	cw.ContentDimensions.Left = contentDimensions.Left
	cw.ContentDimensions.Top = contentDimensions.Top
	cw.ContentDimensions.Width = contentDimensions.Width
	cw.ContentDimensions.Height = contentDimensions.Height

	if cw.Options.RowCallback != nil {
		cw.TotalRows = cw.Options.TotalRows
		if cw.TotalRows > contentDimensions.Height {
			cw.HasScrollbar = true
			cw.ContentDimensions.Width--
		}
	} else if cw.Options.TextBlock != nil {
		if !cw.Options.TextBlock.FitsOnPage(contentDimensions.Width, contentDimensions.Height) {
			cw.HasScrollbar = true
			cw.ContentDimensions.Width--
		}
		cw.TotalRows = cw.Options.TextBlock.NumLines(cw.ContentDimensions.Width)
	} else if cw.Options.RowLoadCallback != nil {
		cw.TotalRows = cw.Options.TotalRows
		if cw.TotalRows > contentDimensions.Height {
			cw.HasScrollbar = true
			cw.ContentDimensions.Width--
		}
	} else {
		cw.TotalRows = len(cw.Options.Rows)
	}

	if scrollPosition > cw.TotalRows-cw.ContentDimensions.Height {
		scrollPosition = cw.TotalRows - cw.ContentDimensions.Height
	}

	if scrollPosition < 0 {
		scrollPosition = 0
	}

	cw.ScrollPosition = scrollPosition

	return cw
}

func WithFixedPositionRows(col int, row int, rows ...*style.Text) ContentWindowRenderOptionFunc {
	return func(opt *ContentWindowRenderOptions) {
		opt.Rows = rows
		opt.Col = col
		opt.Row = row
	}
}

func WithContentWindowOptionTextBlock(tb *style.TextBlock) ContentWindowRenderOptionFunc {
	return func(opt *ContentWindowRenderOptions) {
		opt.TextBlock = tb
	}
}

func WithContentWindowRowCallback(totalRows int, callback func(rowIndex int, yIndex int, width int) *style.Text) ContentWindowRenderOptionFunc {
	return func(opt *ContentWindowRenderOptions) {
		opt.TotalRows = totalRows
		opt.RowCallback = callback
	}
}

func WithContentWindowRowLoadCallback(totalRows int, callback func(cw *ContentWindow) []*style.Text) ContentWindowRenderOptionFunc {
	return func(opt *ContentWindowRenderOptions) {
		opt.RowLoadCallback = callback
		opt.TotalRows = totalRows
	}
}

func (cw *ContentWindow) Render(defaultStyles ...*style.Style) {
	if cw.Options.RowCallback != nil {
		emptyRow, _ := style.T().Render(style.WithFixedWidth(cw.ContentDimensions.Width), style.WithStyles(defaultStyles...))
		for containerY := 0; containerY < cw.ContentDimensions.Height; containerY++ {
			terminal.SetCursorPos(cw.ContentDimensions.Left, cw.ContentDimensions.Top+containerY)
			text := cw.Options.RowCallback(cw.ScrollPosition+containerY, containerY, cw.ContentDimensions.Width)
			if text != nil {
				str, _ := text.Render(style.WithStyles(defaultStyles...))
				fmt.Print(str)
				continue
			}
			fmt.Print(emptyRow)
		}
		return
	}

	if cw.Options.TextBlock != nil {
		rows := cw.Options.TextBlock.Render(cw.ContentDimensions.Width, cw.ContentDimensions.Height, cw.ScrollPosition, defaultStyles...)
		for containerY, row := range rows {
			terminal.SetCursorPos(cw.ContentDimensions.Left, cw.ContentDimensions.Top+containerY)
			fmt.Print(row)
		}
		return
	}

	if cw.Options.RowLoadCallback != nil {
		rows := cw.Options.RowLoadCallback(cw)
		for rowY, row := range rows {
			terminal.SetCursorPos(cw.ContentDimensions.Left, cw.ContentDimensions.Top+rowY)
			str, _ := row.Render(style.WithFixedWidth(cw.ContentDimensions.Width), style.WithStyles(defaultStyles...))
			fmt.Print(str)
		}
	}

	for rowY, row := range cw.Options.Rows {
		terminal.SetCursorPos(cw.Options.Col, cw.Options.Row+rowY)
		str, _ := row.Render(style.WithStyles(defaultStyles...))
		fmt.Print(str)
	}
}

func (cw *ContentWindow) GetScrollWindow() (startRow, endRow int) {
	startFraction := float64(cw.ScrollPosition) / float64(cw.TotalRows)
	endFraction := float64(cw.ScrollPosition+cw.ContentDimensions.Height) / float64(cw.TotalRows)
	startRow = int(math.Round(startFraction * float64(cw.ContentDimensions.Height-1)))
	endRow = int(math.Round(endFraction * float64(cw.ContentDimensions.Height-1)))

	return
}

func Pad(text []rune, width int) []rune {
	if len(text) == width {
		return text
	}

	if len(text) > width {
		return text[:width]
	}

	return append(text, blanks[:width-len(text)]...)
}
