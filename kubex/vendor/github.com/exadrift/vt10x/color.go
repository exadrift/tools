package vt10x

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

var ansiColorMap = map[Color]int{
	Black:        0,
	Red:          1,
	Green:        2,
	Yellow:       3,
	Blue:         4,
	Magenta:      5,
	Cyan:         6,
	LightGrey:    7,
	DarkGrey:     60,
	LightRed:     61,
	LightGreen:   62,
	LightYellow:  63,
	LightBlue:    64,
	LightMagenta: 65,
	LightCyan:    66,
	White:        67,
}

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
