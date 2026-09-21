package style

import (
	"fmt"
	"strings"
)

type StyleType int

const (
	StyleTypeNone StyleType = iota
	StyleTypeFgColor
	StyleTypeBgColor
	StyleTypeStrikeout
	StyleTypeUnderline
	StyleTypeBold
)

const ResetStyleAnsi = "\x1b[0m"

type Style struct {
	ansi      string
	styleType StyleType
}

func (s *Style) Ansi() string {
	return s.ansi
}

type Styles []*Style

func (s Styles) Overrides(styleMap StyleMap) string {
	var b strings.Builder
	for _, st := range s {
		if !styleMap.HasStyleType(st.styleType) {
			b.WriteString(st.ansi)
		}
	}

	return b.String()
}

func (st Styles) Remove(styleTypes ...StyleType) Styles {
	var newStyles Styles
	for _, s := range st {
		filtered := false
		for _, ty := range styleTypes {
			if s.styleType == ty {
				filtered = true
				break
			}
		}

		if !filtered {
			newStyles = append(newStyles, s)
		}
	}

	return newStyles
}

func (st Styles) Add(styles ...*Style) Styles {
	var newStyles Styles
	for _, s := range st {
		filtered := false
		for _, sty := range styles {
			if s.styleType == sty.styleType {
				filtered = true
				break
			}
		}

		if !filtered {
			newStyles = append(newStyles, s)
		}
	}

	newStyles = append(newStyles, styles...)

	return newStyles
}

func (st Styles) Ansi() string {
	var b strings.Builder
	for _, s := range st {
		b.WriteString(s.Ansi())
	}

	return b.String()
}

type StyleMap map[StyleType]*Style

func (sm StyleMap) Ansi() string {
	var b strings.Builder
	for _, s := range sm {
		b.WriteString(s.ansi)
	}

	return b.String()
}

func (sm StyleMap) HasStyleType(styleType StyleType) bool {
	_, ok := sm[styleType]
	return ok
}

func MakeStyleMap(styles ...*Style) StyleMap {
	styleMap := make(map[StyleType]*Style, len(styles))
	for _, s := range styles {
		styleMap[s.styleType] = s
	}

	return styleMap
}

type StyledText struct {
	styles StyleMap
	text   []rune
}

func (st *StyledText) Len() int {
	return len(st.text)
}

func S(text any, styles ...*Style) *StyledText {
	r := &StyledText{
		styles: MakeStyleMap(styles...),
	}

	switch t := text.(type) {
	case string:
		r.text = []rune(t)
	case []rune:
		r.text = t
	default:
		panic(fmt.Errorf("unknown text type %T", t))
	}

	return r
}

type Text struct {
	parts  []*StyledText
	length int
}

type TextBlock struct {
	text                  []*Text
	lastCheckedWidthFits  int
	lastCheckedHeightFits int
	lastCheckedFitsOnPage bool
	lastCheckedRowsWidth  int
	lastCheckedRowsNum    int
}

// T returns a Text object which represents a single line of styled text
func T(items ...any) *Text {
	var parts []*StyledText
	var length int
	for _, item := range items {
		switch ty := item.(type) {
		case *StyledText:
			parts = append(parts, ty)
			length += ty.Len()
		case string:
			st := &StyledText{text: []rune(ty)}
			parts = append(parts, st)
			length += st.Len()
		case []rune:
			st := &StyledText{text: ty}
			parts = append(parts, st)
			length += st.Len()
		case *Text:
			parts = append(parts, ty.parts...)
			length += ty.length
		default:
			panic(fmt.Errorf("unsupported type when initializing textline: %T", ty))
		}
	}

	return &Text{
		parts:  parts,
		length: length,
	}
}

type RenderOptions struct {
	FixedWidth    int
	StyleDefaults Styles
}

func WithStyles(styles ...*Style) func(opt *RenderOptions) {
	return func(opt *RenderOptions) {
		opt.StyleDefaults = styles
	}
}

func WithFixedWidth(width int) func(opt *RenderOptions) {
	return func(opt *RenderOptions) {
		opt.FixedWidth = width
	}
}

func (t *Text) Render(options ...func(*RenderOptions)) (string, *Text) {
	var leftOver *Text
	var leftOverParts []*StyledText

	var builder strings.Builder
	consumedLength := 0
	opts := &RenderOptions{}
	for _, opt := range options {
		opt(opts)
	}

	var remaining int
	if opts.FixedWidth > 0 {
		remaining = opts.FixedWidth
	} else {
		remaining = 1000000
	}

	for i, part := range t.parts {
		styleStr := part.styles.Ansi()
		if styleStr != "" {
			builder.WriteString(styleStr)
		}
		ovrStyleStr := opts.StyleDefaults.Overrides(part.styles)
		if ovrStyleStr != "" {
			builder.WriteString(ovrStyleStr)
		}

		l := len(part.text)

		if l > remaining {
			builder.WriteString(string(part.text[:remaining]))
			builder.WriteString(ResetStyleAnsi)

			leftOverParts = append(leftOverParts, &StyledText{
				styles: part.styles,
				text:   part.text[remaining:],
			})
			if i < len(t.parts)-1 {
				leftOverParts = append(leftOverParts, t.parts[i+1:]...)
			}
			consumedLength += remaining
			leftOver = &Text{
				parts:  leftOverParts,
				length: t.length - consumedLength,
			}

			remaining = 0
			break
		}

		builder.WriteString(string(part.text))
		builder.WriteString(ResetStyleAnsi)
		consumedLength += l
		remaining -= l
	}

	if opts.FixedWidth > 0 && remaining > 0 {
		ovrStyleStr := opts.StyleDefaults.Ansi()
		if ovrStyleStr != "" {
			builder.WriteString(ovrStyleStr)
		}
		builder.WriteString(strings.Repeat(" ", remaining))
		builder.WriteString(ResetStyleAnsi)
	}

	return builder.String(), leftOver
}

func (t *Text) Len() int {
	return t.length
}

// B returns a TextBlock from a series of inputs.  Each input represents a single line of text
func B(items ...any) *TextBlock {
	textItems := make([]*Text, len(items))
	for i, item := range items {
		switch ty := item.(type) {
		case string:
			textItems[i] = T(ty)
		case []rune:
			textItems[i] = T(ty)
		case *Text:
			textItems[i] = ty
		case *StyledText:
			textItems[i] = T(ty)
		}
	}

	return &TextBlock{
		text: textItems,
	}
}

// FitsOnPage returns true if the TextBlock can be rendered to a space with the provided dimensions
func (tb *TextBlock) FitsOnPage(width int, height int) bool {
	if tb.lastCheckedWidthFits == width && tb.lastCheckedHeightFits == height {
		return tb.lastCheckedFitsOnPage
	}

	lines := 0
	for _, row := range tb.text {
		lines += row.length / width
		if row.length%width > 0 || row.length == 0 {
			lines++
		}
		if lines > height {
			tb.lastCheckedWidthFits = width
			tb.lastCheckedHeightFits = height
			tb.lastCheckedFitsOnPage = false
			return false
		}
	}

	tb.lastCheckedWidthFits = width
	tb.lastCheckedHeightFits = height
	tb.lastCheckedFitsOnPage = true
	return true
}

// NumLines returns the number of lines needed to represent itself, given the supplied width
func (tb *TextBlock) NumLines(width int) int {
	if tb.lastCheckedRowsWidth == width {
		return tb.lastCheckedRowsNum
	}

	lines := 0
	for _, row := range tb.text {
		lines += row.length / width
		if row.length%width > 0 || row.length == 0 {
			lines++
		}
	}

	tb.lastCheckedRowsWidth = width
	tb.lastCheckedRowsNum = lines

	return lines
}

func (tb *TextBlock) Render(width int, height int, yOffset int, defaultStyles ...*Style) []string {
	lines := make([]string, height)
	curLine := 0
	yIndex := 0
Outer:
	for _, text := range tb.text {
		var remaining = text
		var rendered string
		for remaining != nil {
			rendered, remaining = remaining.Render(WithFixedWidth(width), WithStyles(defaultStyles...))
			if curLine >= yOffset {
				lines[yIndex] = rendered
				yIndex++
			}
			curLine++
			if yIndex >= height {
				break Outer
			}
		}
	}

	// see if more lines are required
	if yIndex < height {
		whitespace := strings.Repeat(" ", width)
		t := T(whitespace)
		for yIndex < height {
			line, _ := t.Render(WithStyles(defaultStyles...))
			lines[yIndex] = line
			yIndex++
		}
	}

	return lines
}
