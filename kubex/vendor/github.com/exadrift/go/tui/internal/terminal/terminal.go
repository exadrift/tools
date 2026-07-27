package terminal

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

// Start starts a raw terminal session and returns a restore function to be invoked when the
// raw session is to end, prior to exiting the process
func Start() (int, func(), error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return 0, nil, fmt.Errorf("stdin is not a terminal")
	}

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return 0, nil, fmt.Errorf("unable to enable terminal raw mode: %w", err)
	}

	return 0, func() {
		_ = term.Restore(fd, oldState)
	}, nil
}

// SetCursorPos sets the cursor position using a zero based offset
func SetCursorPos(left int, top int) {
	fmt.Printf("\033[%d;%dH", top+1, left+1)
}

// HideCursor hides the cursor
func HideCursor() {
	fmt.Print("\x1b[?25l")
}

// ShowCursor shows the cursor
func ShowCursor() {
	fmt.Print("\x1b[?25h")
}

func Clear() {
	fmt.Print("\033[H\033[2J")
}
