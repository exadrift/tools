package vt10x

import "fmt"

type RgbColor struct {
	Red    uint8
	Green  uint8
	Blue   uint8
	Value  uint32
	AnsiFg string
	AnsiBg string
}

func NewRgbColor(red uint8, green uint8, blue uint8) *RgbColor {
	color := &RgbColor{red, green, blue, uint32(red)<<16 + uint32(green)<<8 + uint32(blue), "", ""}

	color.AnsiFg = fmt.Sprintf("\x1b[38;2;%d;%d;%dm", color.Red, color.Green, color.Blue)
	color.AnsiBg = fmt.Sprintf("\x1b[48;2;%d;%d;%dm", color.Red, color.Green, color.Blue)

	return color
}

// ANSI color values
const (
	Black Color = iota
	Red
	Green
	Yellow
	Blue
	Magenta
	Cyan
	LightGrey
	DarkGrey
	LightRed
	LightGreen
	LightYellow
	LightBlue
	LightMagenta
	LightCyan
	White
)

const (
	AnsiReset string = "\x1b[0m"
)

// Default colors are potentially distinct to allow for special behavior.
// For example, a transparent background. Otherwise, the simple case is to
// map default colors to another color.
const (
	DefaultFG Color = 1<<24 + iota
	DefaultBG
	DefaultCursor
)

// Color maps to the ANSI colors [0, 16) and the xterm colors [16, 256).
type Color uint32

// ANSI returns true if Color is within [0, 16).
func (c Color) ANSI() bool {
	return (c < 16)
}

func build256ColorPalette() []*RgbColor {
	palette := make([]*RgbColor, 256)

	palette[0] = NewRgbColor(0, 0, 0)
	palette[1] = NewRgbColor(128, 0, 0)
	palette[2] = NewRgbColor(0, 128, 0)
	palette[3] = NewRgbColor(128, 128, 0)
	palette[4] = NewRgbColor(0, 0, 128)
	palette[5] = NewRgbColor(128, 0, 128)
	palette[6] = NewRgbColor(0, 128, 128)
	palette[7] = NewRgbColor(192, 192, 192)
	palette[8] = NewRgbColor(128, 128, 128)
	palette[9] = NewRgbColor(255, 0, 0)
	palette[10] = NewRgbColor(0, 255, 0)
	palette[11] = NewRgbColor(255, 255, 0)
	palette[12] = NewRgbColor(0, 0, 255)
	palette[13] = NewRgbColor(255, 0, 255)
	palette[14] = NewRgbColor(0, 255, 255)
	palette[15] = NewRgbColor(255, 255, 255)

	// color section
	intensity := [6]uint8{0, 95, 135, 175, 215, 255}
	for r := range 6 {
		for g := range 6 {
			for b := range 6 {
				index := 16 + (36*r + 6*g + b)
				palette[index] = NewRgbColor(intensity[r], intensity[g], intensity[b])
			}
		}
	}

	// grayscale section
	for i := range 24 {
		color := uint8(10*i + 8)
		palette[232+i] = NewRgbColor(color, color, color)
	}

	return palette
}

func ansiRgbFg(red uint32, green uint32, blue uint32) string {
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", red, green, blue)
}

func ansiRgbBg(red uint32, green uint32, blue uint32) string {
	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", red, green, blue)
}

var palette256Color = build256ColorPalette()
