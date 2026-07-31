package tui

import (
	"fmt"

	"github.com/exadrift/go/tui/internal/terminal"
)

type ScrollWindow struct {
	scrollPosition      int
	dimensions          Dimensions
	scrollHandleEnabled bool
}

func NewScrollWindow() *ScrollWindow {
	return &ScrollWindow{}
}

func (sw *ScrollWindow) Reset() {
	sw.scrollPosition = 0
}

func (sw *ScrollWindow) ScrollHandleEnabled(enabled bool) *ScrollWindow {
	sw.scrollHandleEnabled = enabled
	return sw
}

func (sw *ScrollWindow) SetDimensions(left int, top int, width int, height int) {
	sw.dimensions.Left = left
	sw.dimensions.Top = top
	sw.dimensions.Width = width
	sw.dimensions.Height = height
}

func (sw *ScrollWindow) ScrollUp() {
	sw.scrollPosition--
}

func (sw *ScrollWindow) ScrollDown() {
	sw.scrollPosition++
}

func (sw *ScrollWindow) SetFocusPosition(focusIndex int) {
	dimensions := sw.dimensions
	if focusIndex > dimensions.Height-1+sw.scrollPosition {
		sw.scrollPosition = focusIndex - (dimensions.Height - 1)
	} else if focusIndex < sw.scrollPosition {
		sw.scrollPosition = focusIndex
	}
}

func (sw *ScrollWindow) AdjustScrollPostition(contentRows int) {
	dimensions := sw.dimensions

	// adjust the scroll position to the given dimensions, ensuring that we aren't over scrolled
	var maxScrollOffset int
	if contentRows <= dimensions.Height {
		maxScrollOffset = 0
	} else {
		maxScrollOffset = contentRows - dimensions.Height
	}

	if sw.scrollPosition > maxScrollOffset {
		sw.scrollPosition = maxScrollOffset
	} else if sw.scrollPosition < 0 {
		sw.scrollPosition = 0
	}
}

func (sw *ScrollWindow) Render(contentRows int, callback func(index int) string) {
	dimensions := sw.dimensions

	if callback != nil {
		curRow := 0
		for i := sw.scrollPosition; i < contentRows; i++ {
			if curRow >= dimensions.Height {
				break
			}
			terminal.SetCursorPos(dimensions.Left, dimensions.Top+curRow)
			fmt.Print(callback(i))
			curRow++
		}
	}
}
