package tui

type Text struct {
	*Box
	Contents string
}

func NewText(contents string) *Text {
	return &Text{
		Box:      NewBox(),
		Contents: contents,
	}
}

func (t *Text) CaptureInput(r string) string {
	switch r {
	case UpArrow:
		t.scrollWindow.ScrollUp()
	case DownArrow:
		t.scrollWindow.ScrollDown()
	default:
		return r
	}

	return ""
}

func (t *Text) Render(mode RenderMode, focusItem Widget) {
	dimensions := t.GetContentDimensions()
	lines := WrapTextBasic(t.Contents, dimensions.Width)

	t.RenderWithScroll(mode, focusItem, len(lines), -1, func(index int) string {
		return Pad(lines[index], dimensions.Width)
	})
}
