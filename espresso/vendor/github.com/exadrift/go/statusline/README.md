# statusline
a status bar system for command line applications, which allows persistent and emitted status messages, along with an animated spinner

## usage

```
package main

import (
	"time"

	"github.com/exadrift/go/statusline"
)

func main() {
	s := statusline.NewStatusLine()
	defer s.Stop()

	s.Status("calculating...")
	time.Sleep(5 * time.Second)
	s.Emit("1 + 1 is 2")
	s.Status("verifying...")
	time.Sleep(5 * time.Second)
	s.Emit("it's definitely the case")
}
```