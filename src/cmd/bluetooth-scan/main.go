package main

import (
	"fmt"
	"os"
	"signal-atlas/src/internal/bluetooth"
	"signal-atlas/src/internal/sensor"
)

func main() {
	if e := sensor.RunCLI("bluetooth", func(p string) sensor.Provider { return bluetooth.New(p) }, os.Args[1:]); e != nil {
		fmt.Fprintln(os.Stderr, "Error:", e)
		os.Exit(1)
	}
}
