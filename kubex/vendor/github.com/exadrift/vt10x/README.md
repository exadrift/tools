# vt10x

[![GoDoc](https://godoc.org/github.com/exadrift/vt10x?status.svg)](https://godoc.org/github.com/exadrift/vt10x)

Package vt10x is a vt10x terminal emulation backend, influenced
largely by st, rxvt, xterm, and iTerm as reference. Use it for terminal
muxing, a terminal emulation frontend, or wherever else you need
terminal emulation.

Original vt10x interface has been extended to support the following:
- ANSI line renderer (can return full terminal buffer as ANSI escaped strings for direct rendering to external terminal emulator)
- Maintain history buffer for supporting a scroll-back window
- Standard 256 color palette
- Support for RGB ANSI color rendering (rgb fg / gb colors on the cell will always be larger than 255)

## usage

the vt10x terminal emulator needs to be paired with a pseudo TTY in order to work.  the following demonstrates basic operation:

```
package main

import (
    "os/exec"
    "github.com/creack/pty"
)


func main() {
    // create the terminal with initial size
    term := vt10x.New(vt10x.WithSize(80, 25))

    cmd := exec.Command("/bin/bash")

    // start the command with the pseudo TTY
    ptyFile, err := pty.Start(cmd)

    go func() {
        // create a temporary output buffer
        buf := make([]byte, 4096)

        for {
            // read output from the pseudo TTY
            n, err := ptyFile.Read(buf)
            if err != nil {
                if errors.Is(err, syscall.EIO) {
                    // take appropriate action here, this means the command process exited
					os.Exit(0)
                    return
				}

                log.Fatal(err)
            }

            // write the stdout data from the ptty to the terminal
            _, _ = term.Write(buf[:n])

            // here, your main application should receive a request to redraw the terminal.
            // this will be illustrated below in another block
            ...
        }
    }()

    go func() {
        // this function should receive input and send it to the ptty
        for {
            nBytes, err := os.Stdin.Read(byteBuf)
			if err != nil {
				log.Fatal(err)
			}

            _, _ = ptyFile.Write(byteBuf[:nBytes])
        }
    }()


}
```

there are two primary interfaces for drawing the terminal, depending on what type of output interface is available.  vt10x exposes both a cell interface, allowing an integrator to query data on a per character (cell) basis, which will obtain the character, foreground, and background color

```
for y := 0; y < term.rows; y++ {
    for x := 0; x < term.cols; x++ {
        glyph := term.Cell(x, y)

        // TODO: draw the character
    }
}
```

the second interface allows for rendering of ANSI strings to an external terminal emulator.  if the application is being executed within a terminal emulator, such as gnome terminal, it often makes sense to merely output ANSI escaped text line by line in raw terminal mode and for that, this interface is available:

```
rows := term.AnsiRows()
for y, row := range rows {
    // set cursor position at the top left of the current row
    fmt.Print(row)
}
```

lastly, a similar interface is also supported to render data from the history buffer.

```
// the 0 offset means no offset from the present.  it will be similar to simply calling term.AnsiRows().  A negative offset indicates a position back in the history buffer.  For example -10 would start from 10 rows above the top line
// the history buffer size (including what's not yet part of the history) can be obtained by calling

maxRows := term.HistoryBufferLength()
rows := term.History(0)
for y, row := range rows {
    // set cursor position at the top left of the current row
    fmt.Print(row)
}
```

history can also be accessed using the cell interface, where the row index < 0
