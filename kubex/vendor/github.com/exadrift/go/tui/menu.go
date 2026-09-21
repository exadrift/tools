package tui

import (
	"fmt"
	"sync"

	"github.com/exadrift/go/ansi/style"
)

type Menu struct {
	*Container
	contents      [][]rune
	index         map[string]int
	selectedIndex int
	selectHandler func(int, string) any
	completer     func(any)
	busyLabel     string

	nonSelectedStyle style.Styles
	selectedStyle    style.Styles
}

func NewMenu(contents ...string) *Menu {
	menu := &Menu{
		Container: NewContainer(),
	}

	menu.selectedStyle = style.Styles{style.Blue.Bg(), style.White.Fg()}

	menu.SetContents(contents...)

	return menu
}

// SetSelectHandler allows you to define a callback which will be executed when selection changes
func (m *Menu) SetSelectHandler(h func(selectedIndex int, selectedItem string) any, options ...*Option) *Menu {
	m.selectHandler = h

	for _, option := range options {
		switch option.optionType {
		case BusyModal:
			busyModalData := option.data.(*BusyModalData)
			m.busyLabel = busyModalData.Label
			m.completer = busyModalData.Completer
		default:
			panic(fmt.Errorf("unknown menu option: %v", option.optionType))
		}
	}

	return m
}

// SetSelectedStyle sets styles for the selected row
func (m *Menu) SetSelectedStyle(styles ...*style.Style) *Menu {
	m.selectedStyle = styles
	return m
}

// SetNonSelectedStyle sets styles for the non-selected row
func (m *Menu) SetNonSelectedStyle(styles ...*style.Style) *Menu {
	m.nonSelectedStyle = styles
	return m
}

func (m *Menu) SetContents(contents ...string) *Menu {
	m.contents = make([][]rune, len(contents))
	m.index = make(map[string]int, len(contents))
	m.selectedIndex = 0

	m.ResetScrollPosition()

	for i, item := range contents {
		// sorry, menus shouldn't have any ANSI codes in them
		m.contents[i] = []rune(StripAnsiCodes(item))
	}
	for i, val := range contents {
		m.index[val] = i
	}

	return m
}

func (m *Menu) SetSelectedIndex(index int) *Menu {
	if index < 0 {
		index = len(m.contents) - 1
	} else if index > len(m.contents)-1 {
		index = 0
	}
	m.selectedIndex = index

	return m
}

func (m *Menu) SetSelectedItem(item string) *Menu {
	index := m.index[item]
	m.SetSelectedIndex(index)

	return m
}

func (m *Menu) Render(contentWindow *ContentWindow, focusItem Widget) {
	dimensions := m.GetContentDimensions()

	if m.selectedIndex-m.scrollPosition >= dimensions.Height {
		m.scrollPosition = m.selectedIndex - dimensions.Height + 1
	}

	if m.selectedIndex < m.scrollPosition {
		m.scrollPosition = m.selectedIndex
	}

	contentDimensions := m.GetContentDimensions()
	cw := NewContentWindow(m.scrollPosition, contentDimensions, WithContentWindowRowCallback(len(m.contents), func(rowIndex int, yIndex int, width int) *style.Text {
		if rowIndex >= len(m.contents) {
			return nil
		}
		bareRow := Pad(m.contents[rowIndex], width)
		if m.selectedIndex == rowIndex {
			return style.T(style.S(bareRow, m.selectedStyle...))
		}
		return style.T(style.S(bareRow, m.nonSelectedStyle...))
	}))
	m.Container.Render(cw, focusItem)
}

func (m *Menu) CaptureInput(r string) string {
	switch r {
	case appSingleton.keyBindings.SelectionPrev:
		m.SetSelectedIndex(m.selectedIndex - 1)
	case appSingleton.keyBindings.SelectionNext:
		m.SetSelectedIndex(m.selectedIndex + 1)
	case appSingleton.keyBindings.Trigger:
		if m.selectHandler != nil {
			if m.completer == nil {
				m.selectHandler(m.selectedIndex, string(m.contents[m.selectedIndex]))
				return RenderFullCode
			}

			appSingleton.ShowLoader(m.busyLabel)
			go func() {
				// this is an async operation
				wg := sync.WaitGroup{}
				wg.Add(1)
				var resp any
				go func() {
					defer wg.Done()
					resp = m.selectHandler(m.selectedIndex, string(m.contents[m.selectedIndex]))
				}()
				wg.Wait()
				appSingleton.Async(func() {
					m.completer(resp)
					appSingleton.renderAll()
					appSingleton.HideLoader()
				})
			}()
		}
	default:
		return r
	}

	return ""
}
