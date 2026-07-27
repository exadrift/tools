package tui

import (
	"fmt"

	"github.com/exadrift/go/tui/internal/terminal"
)

type Alignment int

const (
	AlignmentLeft Alignment = iota
	AlignmentRight
	AlignmentCenter
)

const (
	BorderSingleTopLeftCorner     = "┌"
	BorderSingleHoriz             = "─"
	BorderSingleTopRightCorner    = "┐"
	BorderSingleVert              = "│"
	BorderSingleBottomLeftCorner  = "└"
	BorderSingleBottomRightCorner = "┘"

	BorderDoubleTopLeftCorner     = "╔"
	BorderDoubleHoriz             = "═"
	BorderDoubleTopRightCorner    = "╗"
	BorderDoubleVert              = "║"
	BorderDoubleBottomLeftCorner  = "╚"
	BorderDoubleBottomRightCorner = "╝"
)

type RenderMode int

const (
	RenderModeAll RenderMode = iota
	RenderModeBorder
	RenderModeContent
)

type Box struct {
	hasBorder bool
	title     string
	// alignment         Alignment
	dimensions        Dimensions
	contentDimensions Dimensions
}

func NewBox() *Box {
	return &Box{}
}

func (b *Box) Box() *Box {
	return b
}

func (b *Box) GetDimensions() *Dimensions {
	return &b.dimensions
}

func (b *Box) GetContentDimensions() *Dimensions {
	return &b.contentDimensions
}

func (b *Box) EnableBorder(enable bool) *Box {
	b.hasBorder = true
	return b
}

func (b *Box) SetDimensions(left int, top int, width int, height int) {
	b.dimensions.Left = left
	b.dimensions.Top = top
	b.dimensions.Width = width
	b.dimensions.Height = height

	if b.hasBorder {
		b.contentDimensions.Left = left + 1
		b.contentDimensions.Top = top + 1
		b.contentDimensions.Width = width - 2
		b.contentDimensions.Height = height - 2
	} else {
		b.contentDimensions.Left = left
		b.contentDimensions.Top = top
		b.contentDimensions.Width = width
		b.contentDimensions.Height = height
	}
}

func (b *Box) GetBox() *Box {
	return b
}

func (b *Box) Render(mode RenderMode, focusItem Widget) {
	if mode == RenderModeAll || mode == RenderModeBorder {
		if b.hasBorder && b.dimensions.Width >= 3 && b.dimensions.Height >= 3 {
			var (
				topLeft     string
				horiz       string
				topRight    string
				vert        string
				bottomLeft  string
				bottomRight string
			)

			if focusItem != nil && b == focusItem.GetBox() {
				topLeft = BorderDoubleTopLeftCorner
				horiz = BorderDoubleHoriz
				topRight = BorderDoubleTopRightCorner
				vert = BorderDoubleVert
				bottomLeft = BorderDoubleBottomLeftCorner
				bottomRight = BorderDoubleBottomRightCorner
			} else {
				topLeft = BorderSingleTopLeftCorner
				horiz = BorderSingleHoriz
				topRight = BorderSingleTopRightCorner
				vert = BorderSingleVert
				bottomLeft = BorderSingleBottomLeftCorner
				bottomRight = BorderSingleBottomRightCorner
			}

			terminal.SetCursorPos(b.dimensions.Left, b.dimensions.Top)
			fmt.Print(topLeft)
			for i := 1; i < b.dimensions.Width-1; i++ {
				fmt.Print(horiz)
			}
			fmt.Print(topRight)
			for i := 1; i < b.dimensions.Height-1; i++ {
				terminal.SetCursorPos(b.dimensions.Left, b.dimensions.Top+i)
				fmt.Print(vert)

				if mode == RenderModeAll {
					for i := 1; i < b.dimensions.Width-1; i++ {
						fmt.Print(" ")
					}
				} else {
					terminal.SetCursorPos(b.dimensions.Left+b.dimensions.Width-1, b.dimensions.Top+i)
				}

				fmt.Print(vert)
			}
			terminal.SetCursorPos(b.dimensions.Left, b.dimensions.Top+b.dimensions.Height-1)
			fmt.Print(bottomLeft)
			for i := 1; i < b.dimensions.Width-1; i++ {
				fmt.Print(horiz)
			}
			fmt.Print(bottomRight)

			if b.title != "" {
				terminal.SetCursorPos(b.dimensions.Left+2, b.dimensions.Top)
				avail := b.dimensions.Width - 4
				if avail > 2 {
					fmt.Print(Constrain(b.title, avail))
				}
			}
		}
	}
}

func (b *Box) GetChildren() []Widget {
	return nil
}

func (b *Box) CaptureInput(r string) string {
	return r
}

func (b *Box) SetTitle(title string) *Box {
	b.title = title
	return b
}

func (b *Box) GetFocalWidgets(me Widget, focalWidgets *FocalWidgets) {
	if me.CanHaveFocus() {
		focalWidgets.Widgets = append(focalWidgets.Widgets, me)
	}

	for _, child := range me.GetChildren() {
		child.GetFocalWidgets(child, focalWidgets)
	}
}

func (b *Box) NextInFocus(inFocus Widget) Widget {
	return nil
}

func (b *Box) CanHaveFocus() bool {
	return true
}

func (b *Box) AbsorbsInput(input string) bool {
	return false
}

func (b *Box) Collect(me Widget) []Widget {
	widgets := []Widget{me}
	for _, child := range me.GetChildren() {
		widgets = append(widgets, child.Collect(child)...)
	}

	return widgets
}
