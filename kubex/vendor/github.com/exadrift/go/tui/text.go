package tui

import (
	"fmt"

	"github.com/exadrift/go/tui/internal/terminal"
)

type Text struct {
	*Box
	Contents       string
	scrollPosition int
}

func NewText(contents string) *Text {
	return &Text{
		Box:      NewBox(),
		Contents: contents,
	}
}

func (t *Text) Render(mode RenderMode, focusItem Widget) {
	t.Box.Render(mode, focusItem)

	dimensions := t.GetContentDimensions()

	lines := WrapTextBasic(t.Contents, dimensions.Width)
	curRow := 0
	for i := t.scrollPosition; i < len(lines); i++ {
		if curRow >= dimensions.Height {
			break
		}
		terminal.SetCursorPos(dimensions.Left, dimensions.Top+curRow)
		fmt.Print(lines[i])

		curRow++
	}
}
