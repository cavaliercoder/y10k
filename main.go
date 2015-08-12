package main

import (
	"errors"
	"fmt"
	"github.com/codegangsta/cli"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
)

var (
	DebugMode   = false
	YumfilePath = ""
	LogFilePath = ""
)

func main() {
	// ensure logfile handle gets cleaned up
	defer CloseLogFile()

	// route request
	app := cli.NewApp()
	app.Name = "y10k"
	app.Version = "0.1.0"
	app.Author = "Ryan Armstrong"
	app.Email = "ryan@cavaliercoder.com"
	app.Usage = "simplified yum mirror management"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "logfile, l",
			Usage:  "redirect output to a log file",
			EnvVar: "Y10K_LOGFILE",
		},
		cli.BoolFlag{
			Name:   "debug, d",
			Usage:  "print debug output",
			EnvVar: "Y10K_DEBUG",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:  "yumfile",
			Usage: "work with a Yumfile",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "file, f",
					Usage: "path to Yumfile",
					Value: "./Yumfile",
				},
			},
			Before: func(context *cli.Context) error {
				YumfilePath = context.String("file")
				return nil
			},
			Subcommands: []cli.Command{
				{
					Name:   "validate",
					Usage:  "validate a Yumfile's syntax",
					Action: ActionYumfileValidate,
				},
				{
					Name:   "list",
					Usage:  "list repositories in a Yumfile",
					Action: ActionYumfileList,
				},
				{
					Name:   "sync",
					Usage:  "syncronize repos described in a Yumfile",
					Action: ActionYumfileSync,
				},
			},
		},
		{
			Name:  "version",
			Usage: "print the version of y10k",
			Action: func(context *cli.Context) {
				fmt.Printf("%s version %s\n", app.Name, app.Version)
			},
		},
	}

	app.Before = func(context *cli.Context) error {
		// set globals from command line context
		DebugMode = context.GlobalBool("debug")
		LogFilePath = context.GlobalString("logfile")

		// configure logging
		InitLogFile()

		// check system health
		// if err := HealthCheck(); err != nil {
		//	   Fatalf(err, "Health check failed")
		// }

		return nil
	}

	// sig handler
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for _ = range c {
			Printf("Caught SIGINT/Ctrl-C. Cleaning up...\n")

			if cmd != nil {
				Printf("Attempting to terminate %s (PID: %d)...\n", cmd.Path, cmd.Process.Pid)
				cmd.Process.Kill()
			}

			Printf("Exiting\n")
			os.Exit(2)
		}
	}()

	app.Run(os.Args)
}

func cleanUpTempFiles() {
	filepath.Walk(repoFileDir, cleanUpTempFile)
}

func cleanUpTempFile(path string, f os.FileInfo, err error) error {
	if strings.HasPrefix(f.Name(), repoFilePrefix) && strings.HasSuffix(f.Name(), repoFileSuffix) {
		Dprintf("Deleting repo file: %s\n", path)
		os.Remove(path)
	}

	return nil
}

// ActionYumfileValidate processes the 'yumfile validate' command
func ActionYumfileValidate(context *cli.Context) {
	yumfile, err := LoadYumfile(YumfilePath)
	PanicOn(err)
	Printf("Yumfile appears valid (%d repos)\n", len(yumfile.YumRepos))
}

// ActionYumfileList processes the 'yumfile list' command
func ActionYumfileList(context *cli.Context) {
	yumfile, err := LoadYumfile(YumfilePath)
	PanicOn(err)

	repoCount := len(yumfile.YumRepos)
	padding := (len(fmt.Sprintf("%d", repoCount)) * 2) + 1
	for i, repo := range yumfile.YumRepos {
		Printf("%*s %s\n", padding, fmt.Sprintf("%d/%d", i+1, repoCount), repo.YumRepo.ID)
	}
}

// ActionYumfileSync processes the 'yumfile sync' command
func ActionYumfileSync(context *cli.Context) {
	yumfile, err := LoadYumfile(YumfilePath)
	PanicOn(err)

	repo := context.Args().First()
	if repo == "" {
		// sync/update all repos in Yumfile
		if err := yumfile.Sync(false); err != nil {
			Fatalf(err, "Error running Yumfile")
		}
	} else {
		// sync/update one repo in the Yumfile
		mirror := yumfile.Repo(repo)
		if mirror == nil {
			Fatalf(nil, "No such repo found in Yumfile: %s", repo)
		}

		if err := mirror.Sync(); err != nil {
			Fatalf(err, "Error syncronizing repo '%s'", mirror.YumRepo.ID)
		} else {
			if err := mirror.Update(); err != nil {
				Fatalf(err, "Error updating database for repo '%s'", mirror.YumRepo.ID)
			}
		}
	}
}

func PanicOn(err error) {
	if err != nil {
		Fatalf(err, "Fatal error")
	}
}

func NewErrorf(format string, a ...interface{}) error {
	return errors.New(fmt.Sprintf(format, a...))
}
