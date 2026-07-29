package tui

import "fmt"

type Menu struct {
	*Box
	contents         []string
	index            map[string]int
	selectedIndex    int
	selectHandler    func(int, string)
	nonSelectedStyle string
	selectedStyle    string
}

func NewMenu(contents ...string) *Menu {
	menu := &Menu{
		Box: NewBox(),
	}

	menu.selectedStyle = StyleFgBg(White, Blue)

	menu.SetContents(contents...)

	return menu
}

func (m *Menu) SetSelectHandler(h func(selectedIndex int, selectedItem string)) *Menu {
	m.selectHandler = h
	return m
}

// SetStyle sets independent styles for the selected and non-selected states.  Each style is expected to be
// represented as an ANSI escape sequence.  An empty string indicates no applied style.
func (m *Menu) SetStyle(selected string, nonSelected string) *Menu {
	m.selectedStyle = selected
	m.nonSelectedStyle = nonSelected
	return m
}

func (m *Menu) SetContents(contents ...string) *Menu {
	m.contents = make([]string, len(contents))
	m.index = make(map[string]int, len(contents))
	m.selectedIndex = 0

	m.ResetScrollPosition()

	for i, item := range contents {
		// sorry, menus shouldn't have any ANSI codes in them
		m.contents[i] = StripAnsiCodes(item)
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

func (m *Menu) Render(mode RenderMode, focusItem Widget) {
	m.RenderWithScroll(mode, focusItem, len(m.contents), m.selectedIndex, func(index int) string {
		menuLabel := Pad(Constrain(m.contents[index], m.contentDimensions.Width), m.contentDimensions.Width)
		switch index {
		case m.selectedIndex:
			return fmt.Sprintf("%s%s%s", m.selectedStyle, menuLabel, StyleReset)
		default:
			return fmt.Sprintf("%s%s%s", m.nonSelectedStyle, menuLabel, StyleReset)
		}
	})
}

func (m *Menu) CaptureInput(r string) string {
	switch r {
	case UpArrow:
		m.SetSelectedIndex(m.selectedIndex - 1)
	case DownArrow:
		m.SetSelectedIndex(m.selectedIndex + 1)
	case Enter:
		if m.selectHandler != nil {
			m.selectHandler(m.selectedIndex, m.contents[m.selectedIndex])
			return RenderFullCode
		}
	default:
		return r
	}

	return ""
}
