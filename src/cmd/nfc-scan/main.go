package main

import (
	"fmt"
	"os"
	"signal-atlas/src/internal/nfc"
	"signal-atlas/src/internal/sensor"
)

func main() {
	if e := sensor.RunCLI("nfc", func(p string) sensor.Provider { return nfc.New(p) }, os.Args[1:]); e != nil {
		fmt.Fprintln(os.Stderr, "Error:", e)
		os.Exit(1)
	}
}
