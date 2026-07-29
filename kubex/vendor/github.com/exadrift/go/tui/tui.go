package tui

import "fmt"

type Dimensions struct {
	Top    int
	Left   int
	Width  int
	Height int
}

type Widget interface {
	GetChildren() []Widget
	GetDimensions() *Dimensions
	Render(mode RenderMode, inFocus Widget)
	Collect(me Widget) []Widget
	SetDimensions(left int, top int, width int, height int)
	GetBox() *Box
	CaptureInput(r string) string
	GetFocalWidgets(me Widget, focalWidgets *FocalWidgets)
	CanHaveFocus() bool
	AbsorbsInput(input string) bool
	ResetScrollPosition()
}

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
)

type Option struct {
	optionType OptionType
	data       any
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

func Pad(value string, length int) string {
	return fmt.Sprintf("%-*s", length, value)
}
