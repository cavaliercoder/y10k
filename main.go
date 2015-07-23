package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var DebugMode = false

func main() {
	// parse flags
	flag.BoolVar(&DebugMode, "d", false, "print debug output")
	flag.Parse()

	// load default Yumfile
	yumfile, err := LoadYumfile("Yumfile")
	PanicOn(err)

	// check system health
	if err := HealthCheck(); err != nil {
		Fatalf(err, "Health check failed")
	}

	PanicOn(yumfile.Sync())
}

func PanicOn(err error) {
	if err != nil {
		Fatalf(err, "Fatal error")
	}
}

func Fatalf(err error, format string, a ...interface{}) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s: %s\n", fmt.Sprintf(format, a...), err.Error())
	} else {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", fmt.Sprintf(format, a...))
	}

	os.Exit(1)
}

func Printf(format string, a ...interface{}) {
	fmt.Printf(format, a...)
}

func Dprintf(format string, a ...interface{}) {
	if DebugMode {
		fmt.Fprintf(os.Stderr, fmt.Sprintf("DEBUG: %s", format), a...)
	}
}

func Exec(path string, args ...string) error {
	cmd := exec.Command(path, args...)

	// parse stdout async
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			Dprintf("%s: %s\n", cmd.Path, scanner.Text())
		}
	}()

	// attach to stderr
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			Dprintf("%s: %s\n", cmd.Path, scanner.Text())
		}
	}()

	// execute
	Dprintf("exec: %s %s\n", path, strings.Join(args, " "))
	err = cmd.Start()
	if err != nil {
		return err
	}

	// wait for process to finish
	err = cmd.Wait()
	if err != nil {
		return err
	}

	return nil
}
