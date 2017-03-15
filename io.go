package main

import (
	"code.cloudfoundry.org/bytefmt"
	"fmt"
	"github.com/cavaliercoder/grab"
	"log"
	"os"
	"strings"
	"time"
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
			fmt.Fprintf(os.Stderr, "ERROR: %s: %+v\n", fmt.Sprintf(format, a...), err)
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

// URLJoin naively joins paths of a URL to enforce a single '/' separator
// between each segment.
func urljoin(v ...string) string {
	url := ""

	for _, s := range v {
		if url == "" {
			url = s
		} else if s != "" {
			url = fmt.Sprintf("%s/%s", strings.TrimRight(url, "/"), strings.TrimLeft(s, "/"))
		}
	}

	return url
}

// download transfers multiple file requests simultaneously and sends the
// responses through the returned channel once each transfer is complete.
func download(reqs []*grab.Request, workers int) <-chan *grab.Response {
	ret := make(chan *grab.Response, workers)

	go func() {
		// timer to udpate display and progress
		ticker := time.NewTicker(time.Millisecond * 200)
		defer ticker.Stop()

		// client to download files
		respch := grab.DefaultClient.DoBatch(workers, reqs...)

		// progress indicators
		completed := 0
		inProgress := 0
		responses := make([]*grab.Response, 0)

		// loop until done
		for completed < len(reqs) {
			select {
			case resp := <-respch:
				// response received. add to watch list.
				if resp != nil {
					responses = append(responses, resp)
				}

			case <-ticker.C:
				// clear lines
				if inProgress > 0 {
					fmt.Printf("\033[%dA\033[K", inProgress)
				}

				// update completed downloads
				for i, resp := range responses {
					if resp != nil && resp.IsComplete() {
						// print final result
						if resp.Error != nil {
							fmt.Fprintf(os.Stderr, "Error downloading %s: %v\033[K\n", resp.Request.Label, resp.Error)
						} else {
							fmt.Printf("Finished %s (%s in %v)\033[K\n", resp.Request.Label, bytefmt.ByteSize(resp.BytesTransferred()), resp.Duration())
						}

						// mark completed
						responses[i] = nil
						completed++

						// ship to caller
						ret <- resp
					}
				}

				// update downloads in progress
				inProgress = 0
				for _, resp := range responses {
					if resp != nil {
						inProgress++
						fmt.Printf("Downloading %s (%d%% of %s)...\033[K\n", resp.Request.Label, int(100*resp.Progress()), bytefmt.ByteSize(resp.Size))
					}
				}
			}
		}

		// close receiver channel
		close(ret)
	}()

	return ret
}
