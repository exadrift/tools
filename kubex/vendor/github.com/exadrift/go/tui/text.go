package tui

import (
	"github.com/exadrift/go/ansi/style"
)

type Text struct {
	*Container
	Contents *style.TextBlock
}

func NewText(contents *style.TextBlock) *Text {
	return &Text{
		Container: NewContainer(),
		Contents:  contents,
	}
}

func (t *Text) CaptureInput(r string) string {
	switch r {
	case appSingleton.keyBindings.ScrollUp:
		t.ScrollUp()
	case appSingleton.keyBindings.ScrollDown:
		t.ScrollDown()
	default:
		return r
	}

	return ""
}

func (t *Text) Render(contentWindow *ContentWindow, focusItem Widget) {
	cw := NewContentWindow(t.scrollPosition, t.GetContentDimensions(), WithContentWindowOptionTextBlock(t.Contents))
	t.Container.Render(cw, focusItem)
}
