package utils

import "github.com/exadrift/go/statusline"

func NewStatusLine() *statusline.StatusLine {
	return statusline.NewStatusLine(statusline.WithSpinnerStyling("\x1b[0;38;2;255;31;87;49m"))
}
