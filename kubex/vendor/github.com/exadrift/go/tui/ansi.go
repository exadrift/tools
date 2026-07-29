package tui

import (
	"fmt"
	"regexp"
	"strings"
)

var ansiEscStripper = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
var textWrapTokenizer = regexp.MustCompile(`(\x1b\[[0-9;]*[a-zA-Z])|(\n)`)

type TokenType int

const (
	TokenTypeText TokenType = iota
	TokenTypeAnsiCode
	TokenTypeNewline
)

type StringToken struct {
	Text      string
	TokenType TokenType
}

const (
	Black         int = 30
	Red           int = 31
	Green         int = 32
	Yellow        int = 33
	Blue          int = 34
	Magenta       int = 35
	Cyan          int = 36
	White         int = 37
	BlackBright   int = 90
	RedBright     int = 91
	GreenBright   int = 92
	YellowBright  int = 93
	BlueBright    int = 94
	MagentaBright int = 95
	CyanBright    int = 96
	WhiteBright   int = 97
)

const (
	UpArrow   = "\x1b[A"
	DownArrow = "\x1b[B"
	Enter     = "\r"
	CtrlC     = string(rune(3))
	Tab       = "\x09"
	ShiftTab  = "\x1b[Z"

	RenderFullCode = "\x1b[RENDER"
	StyleReset     = "\x1b[0m"
)

func StyleFg(color int) string {
	return fmt.Sprintf("\x1b[%dm", color)
}

func StyleFgBg(fgColor int, bgColor int) string {
	return fmt.Sprintf("\x1b[%d;%dm", fgColor, bgColor+10)
}

func StripAnsiCodes(text string) string {
	return ansiEscStripper.ReplaceAllString(text, "")
}

// SplitAtAnsiTokens splits text into a slice of StringTokens which represent either
// regular text, or an ANSI escape sequence
func SplitAtAnsiTokens(text string) []*StringToken {
	matches := textWrapTokenizer.FindAllStringIndex(text, -1)
	if len(matches) == 0 {
		return []*StringToken{{
			Text:      text,
			TokenType: TokenTypeText,
		}}
	}

	var tokens []*StringToken
	prevEnd := 0
	for _, match := range matches {
		start, end := match[0], match[1]
		var ty TokenType
		switch text[start] {
		case '\n':
			ty = TokenTypeNewline
		default:
			ty = TokenTypeAnsiCode
		}
		if start-prevEnd > 0 {
			tokens = append(tokens, &StringToken{
				Text:      text[prevEnd:start],
				TokenType: TokenTypeText,
			}, &StringToken{
				Text:      text[start:end],
				TokenType: ty,
			})
		} else {
			tokens = append(tokens, &StringToken{
				Text:      text[start:end],
				TokenType: ty,
			})
		}
		prevEnd = end
	}

	if len(text) > prevEnd {
		tokens = append(tokens, &StringToken{
			Text:      text[prevEnd:],
			TokenType: TokenTypeText,
		})
	}

	return tokens
}

// Returns an array of text which separates each line of text at the wrapping point (width)
func WrapTextBasic(text string, width int) []string {
	var wrapped []string
	var sb strings.Builder
	tokens := SplitAtAnsiTokens(text)
	l := 0
	for _, token := range tokens {
		switch token.TokenType {
		case TokenTypeAnsiCode:
			sb.WriteString(token.Text)
			l += len(token.Text)
		case TokenTypeNewline:
			if l > 0 {
				wrapped = append(wrapped, sb.String())
				sb.Reset()
				l = 0
			} else {
				wrapped = append(wrapped, "")
			}
		case TokenTypeText:
			for {
				tokLen := len(token.Text)
				if tokLen+l < width {
					sb.WriteString(token.Text)
					l += tokLen
					break
				} else {
					sb.WriteString(token.Text[:width-l])
					token.Text = token.Text[width-l:]
					wrapped = append(wrapped, sb.String())
					sb.Reset()
					l = 0
				}
			}
		}
	}

	if l > 0 {
		wrapped = append(wrapped, sb.String())
	}

	return wrapped
}
