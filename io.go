package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

const (
	LOG_CAT_ERROR = iota
	LOG_CAT_WARN
	LOG_CAT_INFO
	LOG_CAT_DEBUG
)

var (
	logfileHandle *os.File    = nil
	logger        *log.Logger = nil
)

func InitLogfile() {
	if LogfilePath == "" {
		return
	}

	f, err := os.OpenFile(LogfilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	PanicOn(err)

	logger = log.New(f, "", log.LstdFlags)
}

func CloseLogFile() {
	if logfileHandle != nil {
		PanicOn(logfileHandle.Close())
	}
}

func Logf(category int, format string, a ...interface{}) {
	var cat string
	switch category {
	case LOG_CAT_ERROR:
		cat = "ERROR"
	case LOG_CAT_WARN:
		cat = "WARNING"
	case LOG_CAT_INFO:
		cat = "INFO"
	case LOG_CAT_DEBUG:
		cat = "DEBUG"
	default:
		panic(fmt.Sprintf("Unrecognized log category: %s", category))
	}

	logger.Printf("%s %s", cat, fmt.Sprintf(format, a...))
}

func Printf(format string, a ...interface{}) {
	if logger == nil {
		fmt.Printf(format, a...)
	} else {
		Logf(LOG_CAT_INFO, format, a...)
	}
}

func Fatalf(err error, format string, a ...interface{}) {
	if logger == nil {
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %s: %s\n", fmt.Sprintf(format, a...), err.Error())
		} else {
			fmt.Fprintf(os.Stderr, "ERROR: %s\n", fmt.Sprintf(format, a...))
		}
	} else {
		if err != nil {
			Logf(LOG_CAT_ERROR, "%s: %s\n", fmt.Sprintf(format, a...), err.Error())
		} else {
			Logf(LOG_CAT_ERROR, format, a...)
		}
	}

	os.Exit(1)
}

func Dprintf(format string, a ...interface{}) {
	if DebugMode {
		if logger == nil {
			fmt.Fprintf(os.Stderr, fmt.Sprintf("DEBUG: %s", format), a...)
		} else {
			Logf(LOG_CAT_DEBUG, format, a...)
		}
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
