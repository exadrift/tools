package keys

import (
	"fmt"
	"log"
	"strings"
)

type KeyStroke int

const (
	None  int = 0
	Shift int = 1
	Alt   int = 2
	Ctrl  int = 4
)

type KeyCombo struct {
	Ascii     byte
	Modifier  int
	Name      string
	Ansi      string
	HumanName string
}

func EscapedAnsiString(value string) string {
	b := strings.Builder{}
	for _, c := range value {
		if c <= 31 {
			_, err := fmt.Fprintf(&b, "\\x%x", c)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			_, err := fmt.Fprintf(&b, "%c", c)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	return b.String()
}

func (k *KeyCombo) EscapedAnsiString() string {
	return EscapedAnsiString(k.Ansi)
}

var modCombos = []int{
	Shift,
	Alt,
	Ctrl,
	Shift | Alt,
	Shift | Ctrl,
	Alt | Ctrl,
	Shift | Alt | Ctrl,
}

// makeHumanName produces a human readable key combination name from a base key and a modifier bitmap
func makeHumanName(base string, modifier int) string {
	var keys []string
	if modifier&Ctrl > 0 {
		keys = append(keys, "ctrl")
	}
	if modifier&Alt > 0 {
		keys = append(keys, "alt")
	}

	if modifier&Shift > 0 {
		keys = append(keys, "shift")
	}

	keys = append(keys, base)

	return strings.Join(keys, "+")
}

// NewKeyCombo returns a new KeyCombo instance constructed from
// ascii - an ascii character representing the key where available
// modifier - a bitmap of ctrl + alt + shift
// name - the human name that proxies the ascii character if an ascii character is available, or is unavailable
// ansi - the ansi sequence representing the combination
func NewKeyCombo(ascii byte, modifier int, name string, ansi string) *KeyCombo {
	var base string
	switch name {
	case "":
		base = fmt.Sprintf("%c", ascii)
	default:
		base = name
	}

	return &KeyCombo{ascii, modifier, name, ansi, makeHumanName(base, modifier)}
}

// ParseHumanName parses a human name, regardless of order, and returns an existing KeyCombo representation, or an error
// if one does not exist or the combination is otherwise invalid
func ParseHumanName(value string) (*KeyCombo, error) {
	var tokens []string
	tokenStart := 0
	for i, char := range value {
		if i != tokenStart && char == '+' {
			tokens = append(tokens, strings.Trim(value[tokenStart:i], " "))
			tokenStart = i + 1
		}
	}

	if tokenStart <= len(value)-1 {
		tokens = append(tokens, strings.Trim(value[tokenStart:], " "))
	}

	var modifier int
	var base string
	for _, token := range tokens {
		if len(token) > 1 {
			// always deal in lowercase for named keys
			token = strings.ToLower(token)
		}
		switch token {
		case "ctrl":
			modifier += Ctrl
		case "alt":
			modifier += Alt
		case "shift":
			modifier += Shift
		default:
			if base != "" {
				return nil, fmt.Errorf("human readable name cannot have multiple base keys: %s", value)
			}
			base = token
		}
	}

	humanName := makeHumanName(base, modifier)

	keyCombo, ok := humanNameToKey[humanName]
	if !ok {
		return nil, fmt.Errorf("unsupported key combination: %s", humanName)
	}

	return keyCombo, nil
}

// MustParseHumanName parses a human name and returns the associated KeyCombo object or panics
func MustParseHumanName(value string) *KeyCombo {
	keyCombo, err := ParseHumanName(value)
	if err != nil {
		panic(err)
	}

	return keyCombo
}

// ParseAnsiCode returns a KeyCombo corresponding to a documented ANSI code within this system
// If the ANSI code cannot be identified, and UnknownAnsiCode will be returned
func ParseAnsiCode(code string) (*KeyCombo, error) {
	key := ansiToKey[code]
	if key == nil {
		return nil, fmt.Errorf("unmapped ANSI sequence: %s", EscapedAnsiString(code))
	}

	return key, nil
}

// initAllKeys returns an array of well defined KeyCombo objects
func initAllKeys() []*KeyCombo {
	var allKeys []*KeyCombo
	var ascii byte

	// ! - )
	for ascii = 33; ascii <= 41; ascii++ {
		allKeys = append(
			allKeys,
			NewKeyCombo(ascii, None, "", fmt.Sprintf("%c", ascii)),
			NewKeyCombo(ascii, Alt, "", fmt.Sprintf("\x1b%c", ascii)),
		)
	}

	// 0 - 9
	for ascii = 48; ascii <= 57; ascii++ {
		allKeys = append(
			allKeys,
			NewKeyCombo(ascii, None, "", fmt.Sprintf("%c", ascii)),
			NewKeyCombo(ascii, Alt, "", fmt.Sprintf("\x1b%c", ascii)),
		)
	}

	// : - @
	for ascii = 58; ascii <= 64; ascii++ {
		allKeys = append(
			allKeys,
			NewKeyCombo(ascii, None, "", fmt.Sprintf("%c", ascii)),
			NewKeyCombo(ascii, Alt, "", fmt.Sprintf("\x1b%c", ascii)),
		)
	}

	// A - Z
	for ascii = 65; ascii <= 90; ascii++ {
		allKeys = append(
			allKeys,
			NewKeyCombo(ascii, None, "", fmt.Sprintf("%c", ascii)),
			NewKeyCombo(ascii, Ctrl, "", fmt.Sprintf("%c", ascii-64)), // caps and lowercase both produce this same keystroke ANSI
			NewKeyCombo(ascii, Alt, "", fmt.Sprintf("\x1b%c", ascii)),
			NewKeyCombo(ascii, Ctrl|Alt, "", fmt.Sprintf("\x1b%c", ascii-64)), // caps and lowercase both produce this same keystroke ANSI
		)
	}

	// a - z
	for ascii = 97; ascii <= 122; ascii++ {
		allKeys = append(
			allKeys,
			NewKeyCombo(ascii, None, "", fmt.Sprintf("%c", ascii)),
			NewKeyCombo(ascii, Ctrl, "", fmt.Sprintf("%c", ascii-96)), // caps and lowercase both produce this same keystroke ANSI
			NewKeyCombo(ascii, Alt, "", fmt.Sprintf("\x1b%c", ascii)),
			NewKeyCombo(ascii, Ctrl|Alt, "", fmt.Sprintf("\x1b%c", ascii-96)), // caps and lowercase both produce this same keystroke ANSI
		)
	}

	allKeys = append(
		allKeys,
		NewKeyCombo('[', None, "", "["),
		NewKeyCombo('[', Ctrl, "", "\x1b"),
		NewKeyCombo('[', Ctrl|Alt, "", "\x1b\x1b"),
		NewKeyCombo(']', None, "", "]"),
		NewKeyCombo(']', Ctrl, "", "\x1d"),
		NewKeyCombo(']', Ctrl|Alt, "", "\x1b\x1d"),
		NewKeyCombo('{', None, "", "{"),
		NewKeyCombo('}', None, "", "}"),
		NewKeyCombo(0, None, "enter", "\x0d"),
		NewKeyCombo(0, Alt, "enter", "\x1b\x0d"),
		NewKeyCombo(0, None, "space", " "),
		NewKeyCombo(0, Ctrl|Alt, "space", "\x1b\x00"),
		NewKeyCombo(0, None, "esc", "\x1b"),
	)

	allKeys = append(
		allKeys,
		NewKeyCombo('\x09', None, "tab", "\x09"),
		NewKeyCombo('\x09', Shift, "tab", "\x1b[Z"),
	)

	allKeys = append(allKeys, makeNavModifierKeyCombos("home", "1", "H", modCombos...)...)
	allKeys = append(allKeys, makeNavModifierKeyCombos("end", "1", "F", modCombos...)...)
	allKeys = append(allKeys, makeNavModifierKeyCombos("ins", "2", "~", modCombos...)...)
	allKeys = append(allKeys, makeNavModifierKeyCombos("del", "3", "~", modCombos...)...)
	allKeys = append(allKeys, makeNavModifierKeyCombos("pgup", "5", "~", modCombos...)...)
	allKeys = append(allKeys, makeNavModifierKeyCombos("pgdn", "6", "~", modCombos...)...)

	allKeys = append(allKeys, NewKeyCombo(0, None, "ins", "\x1b[2~"))
	allKeys = append(allKeys, NewKeyCombo(0, None, "del", "\x1b[3~"))
	allKeys = append(allKeys, NewKeyCombo(0, None, "home", "\x1b[H"))
	allKeys = append(allKeys, NewKeyCombo(0, None, "end", "\x1b[F"))
	allKeys = append(allKeys, NewKeyCombo(0, None, "pgup", "\x1b[5~"))
	allKeys = append(allKeys, NewKeyCombo(0, None, "pgdn", "\x1b[6~"))

	allKeys = append(allKeys, makeNavModifierKeyCombos("up", "1", "A", modCombos...)...)
	allKeys = append(allKeys, NewKeyCombo(0, None, "up", "\x1b[A"))
	allKeys = append(allKeys, makeNavModifierKeyCombos("down", "1", "B", modCombos...)...)
	allKeys = append(allKeys, NewKeyCombo(0, None, "down", "\x1b[B"))
	allKeys = append(allKeys, makeNavModifierKeyCombos("left", "1", "D", modCombos...)...)
	allKeys = append(allKeys, NewKeyCombo(0, None, "left", "\x1b[D"))
	allKeys = append(allKeys, makeNavModifierKeyCombos("right", "1", "C", modCombos...)...)
	allKeys = append(allKeys, NewKeyCombo(0, None, "right", "\x1b[C"))

	return allKeys
}

func makeNavModifierKeyCombos(name string, keycode string, ansiSuffix string, modCombinations ...int) []*KeyCombo {
	var keyCombos []*KeyCombo
	for _, modCbo := range modCombinations {
		ansi := fmt.Sprintf("\x1b[%s;%d%s", keycode, modCbo+1, ansiSuffix)
		keyCombo := NewKeyCombo(0, modCbo, name, ansi)
		keyCombos = append(keyCombos, keyCombo)
	}

	return keyCombos
}

func initHumanNamedKeyMap(source []*KeyCombo) map[string]*KeyCombo {
	keyCombos := map[string]*KeyCombo{}
	for _, k := range source {
		keyCombos[k.HumanName] = k
	}

	return keyCombos
}

func initAnsiKeyMap(source []*KeyCombo) map[string]*KeyCombo {
	keyCombos := map[string]*KeyCombo{}
	for _, k := range source {
		keyCombos[k.Ansi] = k
	}

	return keyCombos
}

var AllKeys = initAllKeys()
var humanNameToKey = initHumanNamedKeyMap(AllKeys)
var ansiToKey = initAnsiKeyMap(AllKeys)
