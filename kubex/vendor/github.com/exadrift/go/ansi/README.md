# ansi
provides a mechanism to define keyboard key combinations and map to and from ANSI values, in order to provide configurable input processing for terminal based applications

## getting started
```
import (
    "github.com/exadrift/go/ansi
)

# get a KeyCombo object from a human readable key combination string
altEnter := ansi.MustParseHumanName("alt+enter")

# now print the ANSI code
fmt.Print(altEnter.Ansi)

# lookup a KeyCombo object by an ANSI code
keyCombo, err := ansi.ParseAnsiCode("\x1b")
if err != nil {
    // the ANSI sequence isn't mapped in this library
}
```