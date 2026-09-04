package style

import (
	"fmt"
	"regexp"
	"strings"
)

var finder = regexp.MustCompile("\n")

var ansiCodeRemover = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-A]`)

type Color uint32

const (
	Black Color = iota
	Red
	Green
	Yellow
	Blue
	Magenta
	Cyan
	White
	BlackHi
	RedHi
	GreenHi
	YellowHi
	BlueHi
	MagentaHi
	CyanHi
	WhiteHi
)

type StyleType int

const (
	StyleTypeFgColor StyleType = iota
	StyleTypeBgColor
	StyleTypeLineBreak
	StyleTypeReset
)

type Style struct {
	Ansi      string
	StyleType StyleType
}
type Text []any

var (
	Break = Style{"", StyleTypeLineBreak}
)

var (
	StyleReset = Style{"\x1b[0m", StyleTypeReset}
)

func toRgb(color Color) (uint32, uint32, uint32) {
	c := uint32(color)
	return c & 0xFF000000 >> 24, c & 0x00FF0000 >> 16, c & 0x0000FF00 >> 8
}

func fgColor(color Color) uint32 {
	var intensity Color
	if color < 8 {
		intensity = 30
	} else {
		intensity = 90
	}

	return uint32(intensity + color)
}

func bgColor(color Color) uint32 {
	var intensity Color
	if color < 8 {
		intensity = 40
	} else {
		intensity = 100
	}

	return uint32(intensity + color)
}

// Fg return an ANSI foreground style representation of the color
func (c Color) Fg() Style {
	// palette color
	if c < 256 {
		return Style{fmt.Sprintf("\x1b[%dm", fgColor(c)), StyleTypeFgColor}
	}

	red, green, blue := toRgb(c)
	return Style{fmt.Sprintf("\x1b[38;2;%d;%d;%dm", red, green, blue), StyleTypeBgColor}
}

// Bg return an ANSI background style representation of the color
func (c Color) Bg() Style {
	// palette color
	if c < 256 {
		return Style{fmt.Sprintf("\x1b[%dm", bgColor(c)), StyleTypeBgColor}
	}

	red, green, blue := toRgb(c)
	return Style{fmt.Sprintf("\x1b[48;2;%d;%d;%dm", red, green, blue), StyleTypeBgColor}
}

// Construct a color from an RGB tri
func FromRgb(red uint32, green uint32, blue uint32) Color {
	return Color(red<<24 + green<<16 + blue<<8)
}

func (t Text) Len() int {
	l := 0
	for _, t := range t {
		switch ty := t.(type) {
		case []rune:
			l += len(ty)
		case Style:
			continue
		default:
			panic("unknown type in text object")
		}
	}

	return l
}

// Generates a styled Text object from a collection of strings and/or style cues
func T(text ...any) Text {
	var cat Text
	for _, t := range text {
		switch ty := t.(type) {
		case string:
			prevStart := 0
			indexList := finder.FindAllStringIndex(ty, -1)
			for _, indexes := range indexList {
				start := indexes[0]
				end := indexes[1]

				if start > prevStart {
					cat = append(cat, []rune(ty[prevStart:start]))
				}
				cat = append(cat, Break)
				prevStart = end
			}
			if prevStart < len(ty) {
				cat = append(cat, []rune(ty[prevStart:]))
			}
		case Style:
			cat = append(cat, ty)
		default:
			panic("unknown type in text")
		}
	}

	return cat
}

type RenderOption struct {
	Width   int
	MinRows int
}

type RenderOptionFunc func(*RenderOption)

func WithWidthConstraint(width int) RenderOptionFunc {
	return func(opt *RenderOption) {
		opt.Width = width
	}
}

func WithMinRows(minRows int) RenderOptionFunc {
	return func(opt *RenderOption) {
		opt.MinRows = minRows
	}
}

func (t Text) Extend(add ...any) Text {
	newText := t
	for _, item := range add {
		switch ty := item.(type) {
		case string:
			newText = append(newText, []rune(ty))
		case []rune:
			newText = append(newText, ty)
		case Style:
			newText = append(newText, ty)
		case Text:
			newText = append(newText, ty...)
		default:
			panic("unknown item being added")
		}
	}

	return newText
}

func (t Text) Render(options ...RenderOptionFunc) []string {
	opt := &RenderOption{}
	for _, ofunc := range options {
		ofunc(opt)
	}

	var rows []string
	var curRow strings.Builder
	curRowLen := 0

	for _, token := range t {
		switch tType := token.(type) {
		case Style:
			switch tType.StyleType {
			case StyleTypeLineBreak:
				// this if block isn't executed when width is zero, which means no padding happens, this is correct
				if curRowLen < opt.Width {
					curRow.WriteString(strings.Repeat(" ", opt.Width-curRowLen))
				}
				rows = append(rows, curRow.String())
				curRow.Reset()
				curRowLen = 0
			default:
				curRow.WriteString(tType.Ansi)
			}
		case []rune:
			for len(tType) > 0 {
				if curRowLen+len(tType) <= opt.Width || opt.Width == 0 {
					curRow.WriteString(string(tType))
					curRowLen += len(tType)
					tType = nil
				} else {
					curRow.WriteString(string(tType[:opt.Width-curRowLen]))
					tType = tType[opt.Width-curRowLen:]
					rows = append(rows, curRow.String())
					curRow.Reset()
					curRowLen = 0
				}
			}
		}
	}

	if curRowLen > 0 && curRowLen < opt.Width {
		curRow.WriteString(strings.Repeat(" ", opt.Width-curRowLen))
	}
	if curRow.Len() > 0 {
		rows = append(rows, curRow.String())
		curRow.Reset()
	}

	// if there are empty rows as far as the minimum height is concerned, this will fill them
	for i := len(rows); i < opt.MinRows; i++ {
		if opt.Width > 0 {
			rows = append(rows, strings.Repeat(" ", opt.Width))
		} else {
			rows = append(rows, "")
		}
	}

	return rows
}

// WithDefaultStyles returns a new Text object with default styles imposed against the text.  This
// ensures that whenever a reset is applied, the default styles are re-imposed.  Default styles will
// be overridden only when an explicit style is applied within the text.
func (t Text) WithDefaultStyles(styles ...Style) Text {
	var newText Text
	// first apply underride styles
	for _, style := range styles {
		newText = append(newText, style)
	}
	for _, part := range t {
		newText = append(newText, part)
		if style, ok := part.(Style); ok {
			if style.StyleType == StyleTypeReset {
				// re-apply the style underrides
				for _, style := range styles {
					newText = append(newText, style)
				}
			}
		}
	}

	return newText
}

// StripAnsi will remove any ANSI sequences from the provided text
func StripAnsi(text string) string {
	return ansiCodeRemover.ReplaceAllString(text, "")
}
