package tui

import (
	"fmt"

	"github.com/exadrift/go/ansi/style"
	"github.com/exadrift/go/tui/internal/terminal"
)

type Container struct {
	title                []rune
	titleString          string
	titleStyle           style.Styles
	focusTitleStyle      style.Styles
	backgroundStyle      *style.Style
	focusBackgroundStyle *style.Style
	canHaveFocus         bool
	dimensions           Dimensions
	contentDimensions    Dimensions
	scrollPosition       int
	scrollerBackground   string
	scrollerHandle       string
}

func NewContainer() *Container {
	sb, _ := style.T(style.S(" ", style.FromRgb(60, 60, 60).Bg())).Render()
	sh, _ := style.T(style.S(" ", style.FromRgb(100, 100, 100).Bg())).Render()
	return &Container{
		canHaveFocus:         true,
		titleStyle:           style.Styles{style.FromRgb(50, 50, 50).Bg(), style.FromRgb(180, 180, 180).Fg()},
		focusTitleStyle:      style.Styles{style.FromRgb(180, 180, 180).Bg(), style.FromRgb(0, 0, 0).Fg()},
		backgroundStyle:      style.FromRgb(30, 30, 30).Bg(),
		focusBackgroundStyle: style.FromRgb(40, 40, 40).Bg(),
		scrollerBackground:   sb,
		scrollerHandle:       sh,
	}
}

func (c *Container) Reset() {
}

func (c *Container) Container() *Container {
	return c
}

func (c *Container) GetDimensions() *Dimensions {
	return &c.dimensions
}

func (c *Container) GetContentDimensions() *Dimensions {
	return &c.contentDimensions
}

func (c *Container) SetBackgroundStyle(style *style.Style) {
	c.backgroundStyle = style
}

func (c *Container) SetFocusedBackgroundStyle(style *style.Style) {
	c.focusBackgroundStyle = style
}

func (c *Container) SetFocusable(canHaveFocus bool) *Container {
	c.canHaveFocus = canHaveFocus
	return c
}

func (c *Container) SetDimensions(left int, top int, width int, height int) {
	c.dimensions.Left = left
	c.dimensions.Top = top
	c.dimensions.Width = width
	c.dimensions.Height = height

	c.contentDimensions.Left = left
	c.contentDimensions.Width = width

	if len(c.title) > 0 {
		c.contentDimensions.Top = top + 1
		c.contentDimensions.Height = height - 1
	} else {
		c.contentDimensions.Top = top
		c.contentDimensions.Height = height
	}
}

func (c *Container) ScrollUp() {
	c.scrollPosition--
}

func (c *Container) ScrollDown() {
	c.scrollPosition++
}

func (c *Container) GetContainer() *Container {
	return c
}

func (c *Container) GetChildren() []Widget {
	return nil
}

func (c *Container) CaptureInput(r string) string {
	return r
}

func (c *Container) SetTitle(title string) *Container {
	c.title = []rune(title)
	c.titleString = title
	return c
}

func (c *Container) GetFocalWidgets(me Widget, focalWidgets *FocalWidgets) {
	if me.CanHaveFocus() {
		focalWidgets.Widgets = append(focalWidgets.Widgets, me)
	}

	for _, child := range me.GetChildren() {
		child.GetFocalWidgets(child, focalWidgets)
	}
}

func (c *Container) NextInFocus(inFocus Widget) Widget {
	return nil
}

func (c *Container) CanHaveFocus() bool {
	return c.canHaveFocus
}

func (c *Container) AbsorbsInput(input string) bool {
	return false
}

func (c *Container) Collect(me Widget) []Widget {
	widgets := []Widget{me}
	for _, child := range me.GetChildren() {
		widgets = append(widgets, child.Collect(child)...)
	}

	return widgets
}

func (c *Container) ResetScrollPosition() {
	c.scrollPosition = 0
}

func (c *Container) Render(contentWindow *ContentWindow, focusItem Widget) {
	if contentWindow == nil {
		return
	}
	c.scrollPosition = contentWindow.ScrollPosition

	inFocus := focusItem != nil && c == focusItem.GetContainer()

	dimensions := c.GetDimensions()
	if len(c.title) > 0 {
		terminal.SetCursorPos(dimensions.Left, dimensions.Top)
		var titleStyles style.Styles

		if inFocus {
			titleStyles = c.focusTitleStyle
		} else {
			titleStyles = c.titleStyle
		}
		s := style.S(Pad(c.title, dimensions.Width), titleStyles...)
		titleText := style.T(s)
		rendered, _ := titleText.Render()
		fmt.Print(rendered)
	}

	var backgroundStyle *style.Style
	if inFocus {
		backgroundStyle = c.focusBackgroundStyle
	} else {
		backgroundStyle = c.backgroundStyle
	}

	contentDimensions := c.GetContentDimensions()
	if backgroundStyle == nil {
		contentWindow.Render()
	} else {
		contentWindow.Render(backgroundStyle)
	}

	var startRow, endRow int
	if contentWindow.HasScrollbar {
		startRow, endRow = contentWindow.GetScrollWindow()

		print(style.ResetStyleAnsi)
		for i := 0; i < contentDimensions.Height; i++ {
			terminal.SetCursorPos(contentDimensions.Left+contentDimensions.Width-1, contentDimensions.Top+i)
			if i >= startRow && i <= endRow {
				// draw the cursor
				print(c.scrollerHandle)
			} else {
				// draw the ruler
				print(c.scrollerBackground)
			}
		}
	}
	print(style.ResetStyleAnsi)
}
