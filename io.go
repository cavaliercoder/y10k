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
	cmd           *exec.Cmd   = nil
	logfileHandle *os.File    = nil
	logger        *log.Logger = nil
)

func InitLogFile() {
	if LogFilePath == "" {
		return
	}

	f, err := os.OpenFile(LogFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	PanicOn(err)

	logger = log.New(f, "", log.LstdFlags)
}

// CloseLogFile cleans up any file handles associates with the log file.
func CloseLogFile() {
	if logfileHandle != nil {
		PanicOn(logfileHandle.Close())
	}
}

// Logf prints output to a logfile with a category and timestamp
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

// Printf prints output to STDOUT or the logfile
func Printf(format string, a ...interface{}) {
	if logger == nil {
		fmt.Printf(format, a...)
	} else {
		Logf(LOG_CAT_INFO, format, a...)
	}
}

// Errorf prints an error message to log or STDOUT
func Errorf(err error, format string, a ...interface{}) {
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
}

// Fatalf prints an error message to log or STDOUT and exits the program with
// a non-zero exit code
func Fatalf(err error, format string, a ...interface{}) {
	Errorf(err, format, a...)
	os.Exit(1)
}

// Dprintf prints verbose output only if debug mode is enabled
func Dprintf(format string, a ...interface{}) {
	if DebugMode {
		if logger == nil {
			fmt.Fprintf(os.Stderr, fmt.Sprintf("DEBUG: %s", format), a...)
		} else {
			Logf(LOG_CAT_DEBUG, format, a...)
		}
	}
}

// Exec executes a system command and redirects the commands output to debug
func Exec(path string, args ...string) error {
	if cmd != nil {
		return NewErrorf("Child process is aleady running (%s:%d)", cmd.Path, cmd.Process.Pid)
	}

	cmd = exec.Command(path, args...)
	defer func() {
		cmd = nil
	}()

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
	Dprintf("exec: started with PID: %d\n", cmd.Process.Pid)

	// wait for process to finish
	err = cmd.Wait()
	if err != nil {
		return err
	}
	Dprintf("exec: finished\n")
	cmd = nil

	return nil
}
