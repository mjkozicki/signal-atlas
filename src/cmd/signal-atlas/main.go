// signal-atlas starts the three independent processes and stops its children on exit.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
)

func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
func run() error {
	demo := flag.Bool("demo", false, "use isolated demo databases and seed synthetic observations")
	container := flag.Bool("container", false, "bind the dashboard to 0.0.0.0 for Docker; publish port 8787 on host loopback only")
	python := flag.String("python", "", "native-provider Python interpreter")
	flag.Parse()
	if flag.NArg() != 0 {
		return fmt.Errorf("usage: signal-atlas [--demo] [--container] [--python PATH]")
	}
	exe, e := os.Executable()
	if e != nil {
		return e
	}
	bin := filepath.Dir(exe)
	names := []string{"bluetooth", "nfc", "wifi"}
	commands := []*exec.Cmd{}
	for _, name := range names {
		db := ".data/" + name + ".db"
		if name == "wifi" {
			db = ".data/wifi-scanner.db"
		}
		if *demo {
			db = ".data/" + name + "-demo.db"
			if name == "wifi" {
				db = ".data/demo.db"
			}
		}
		path := filepath.Join(bin, name+"-scan")
		if *demo {
			if _, e = os.Stat(db); os.IsNotExist(e) {
				seed := exec.Command(path, "demo", "--db", db)
				seed.Stderr = os.Stderr
				if e = seed.Run(); e != nil {
					return fmt.Errorf("seed %s: %w", name, e)
				}
			}
		}
		args := []string{"serve", "--db", db}
		if name == "wifi" && *container {
			args = append(args, "--container", "--addr", "0.0.0.0:8787")
		}
		if name != "wifi" && *python != "" {
			args = append(args, "--python", *python)
		}
		commands = append(commands, exec.Command(path, args...))
	}
	started := []*exec.Cmd{}
	done := make(chan error, len(commands))
	stop := func() {
		for _, c := range started {
			c.Process.Kill()
		}
	}
	defer stop()
	for _, c := range commands {
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if e = c.Start(); e != nil {
			return e
		}
		started = append(started, c)
		go func(cmd *exec.Cmd) { done <- cmd.Wait() }(c)
	}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	select {
	case <-signals:
		for _, c := range started {
			c.Process.Signal(os.Interrupt)
		}
		for range started {
			<-done
		}
		return nil
	case e = <-done:
		if e == nil {
			return fmt.Errorf("a scanner service exited")
		}
		return fmt.Errorf("scanner service exited: %w", e)
	}
}
