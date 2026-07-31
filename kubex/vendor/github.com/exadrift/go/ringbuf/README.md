# ringbuf
implementation of a ring buffer with generics

## usage
```
package main

import (
    "github.com/exadrift/go/ringbuf"
)

func main() {
    rb := ringbuf.New[string](100)
    rb.Push("------ start -------")
    rb.Push("- this is the start -")
    rb.Push("- this is the next line -")
    rb.Push("- this is the third line -")
    rb.Push("------ end -------")

    // this will print the items out in the order that they were inserted
    fmt.Println(rb.Item(-4))
    fmt.Println(rb.Item(-3))
    fmt.Println(rb.Item(-2))
    fmt.Println(rb.Item(-1))
    fmt.Println(rb.Item(0))
}
```