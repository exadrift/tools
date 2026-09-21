package style

import (
	"fmt"
)

type Color uint32

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
func (c Color) Fg() *Style {
	// palette color
	if c < 256 {
		return &Style{fmt.Sprintf("\x1b[%dm", fgColor(c)), StyleTypeFgColor}
	}

	red, green, blue := toRgb(c)
	return &Style{fmt.Sprintf("\x1b[38;2;%d;%d;%dm", red, green, blue), StyleTypeFgColor}
}

// Bg return an ANSI background style representation of the color
func (c Color) Bg() *Style {
	// palette color
	if c < 256 {
		return &Style{fmt.Sprintf("\x1b[%dm", bgColor(c)), StyleTypeBgColor}
	}

	red, green, blue := toRgb(c)
	return &Style{fmt.Sprintf("\x1b[48;2;%d;%d;%dm", red, green, blue), StyleTypeBgColor}
}

// Construct a color from an RGB tri
func FromRgb(red uint32, green uint32, blue uint32) Color {
	return Color(red<<24 + green<<16 + blue<<8)
}

const (
	Black Color = iota
	Red
	Green
	Yellow
	Blue
	Magenta
	Cyan
	White
)

var color256Palette = build256ColorPalette()

func build256ColorPalette() []Color {
	palette := make([]Color, 256)

	palette[0] = FromRgb(0, 0, 0)
	palette[1] = FromRgb(128, 0, 0)
	palette[2] = FromRgb(0, 128, 0)
	palette[3] = FromRgb(128, 128, 0)
	palette[4] = FromRgb(0, 0, 128)
	palette[5] = FromRgb(128, 0, 128)
	palette[6] = FromRgb(0, 128, 128)
	palette[7] = FromRgb(192, 192, 192)
	palette[8] = FromRgb(128, 128, 128)
	palette[9] = FromRgb(255, 0, 0)
	palette[10] = FromRgb(0, 255, 0)
	palette[11] = FromRgb(255, 255, 0)
	palette[12] = FromRgb(0, 0, 255)
	palette[13] = FromRgb(255, 0, 255)
	palette[14] = FromRgb(0, 255, 255)
	palette[15] = FromRgb(255, 255, 255)

	// color section
	intensity := [6]uint32{0, 95, 135, 175, 215, 255}
	for r := range 6 {
		for g := range 6 {
			for b := range 6 {
				index := 16 + (36*r + 6*g + b)
				palette[index] = FromRgb(intensity[r], intensity[g], intensity[b])
			}
		}
	}

	// grayscale section
	for i := range 24 {
		color := uint32(10*i + 8)
		palette[232+i] = FromRgb(color, color, color)
	}

	return palette
}

// Returns a color from the standard palette
func GetPaletteColor(color uint8) Color {
	return color256Palette[color]
}
