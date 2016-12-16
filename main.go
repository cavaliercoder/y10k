package main

import (
	"errors"
	"fmt"
	"github.com/codegangsta/cli"
	"os"
	"os/signal"
	"path/filepath"
)

var (
	QuietMode       bool
	DebugMode       bool
	YumfilePath     string
	LogFilePath     string
	TmpBasePath     string
	TmpYumConfPath  string
	TmpYumLogFile   string
	TmpYumCachePath string
	DownloadThreads int
)

func main() {
	// ensure logfile handle gets cleaned up
	defer CloseLogFile()

	// route request
	app := cli.NewApp()
	app.Name = "y10k"
	app.Version = "0.3.0"
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
			Name:  "quiet, q",
			Usage: "less verbose",
		},
		cli.BoolFlag{
			Name:   "debug, d",
			Usage:  "print debug output",
			EnvVar: "Y10K_DEBUG",
		},
		cli.StringFlag{
			Name:   "cachedir, c",
			Usage:  "path to y10k cache",
			Value:  "/var/cache/y10k",
			EnvVar: "Y10K_CACHEDIR",
		},
		cli.IntFlag{
			Name:  "threads, t",
			Usage: "maximum simultanious downloads",
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
			Name:  "serve",
			Usage: "naive directory index server",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "listen, l",
					Usage: "listen address",
					Value: ":8080",
				},
			},
			Action: ActionServe,
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
		QuietMode = context.GlobalBool("quiet")
		DebugMode = context.GlobalBool("debug")
		LogFilePath = context.GlobalString("logfile")

		TmpBasePath = context.GlobalString("cachedir")

		TmpYumConfPath = filepath.Join(TmpBasePath, "yum.conf")
		TmpYumLogFile = filepath.Join(TmpBasePath, "yum.log")
		TmpYumCachePath = TmpBasePath

		DownloadThreads = context.GlobalInt("threads")
		if DownloadThreads == 0 {
			DownloadThreads = 3
		}

		// configure logging
		InitLogFile()

		return nil
	}

	// sig handler
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for _ = range c {
			Printf("Caught SIGINT/Ctrl-C. Cleaning up...\n")
			os.Exit(2)
		}
	}()

	app.Run(os.Args)
}

// ActionYumfileValidate processes the 'yumfile validate' command
func ActionYumfileValidate(context *cli.Context) {
	yumfile, err := LoadYumfile(YumfilePath)
	PanicOn(err)
	Printf("Yumfile appears valid (%d repos)\n", len(yumfile.Repos))
}

// ActionYumfileList processes the 'yumfile list' command
func ActionYumfileList(context *cli.Context) {
	yumfile, err := LoadYumfile(YumfilePath)
	PanicOn(err)

	repoCount := len(yumfile.Repos)
	padding := (len(fmt.Sprintf("%d", repoCount)) * 2) + 1
	for i, repo := range yumfile.Repos {
		Printf("%*s %s -> %s\n", padding, fmt.Sprintf("%d/%d", i+1, repoCount), repo.ID, repo.LocalPath)
	}
}

// ActionYumfileSync processes the 'yumfile sync' command
func ActionYumfileSync(context *cli.Context) {
	yumfile, err := LoadYumfile(YumfilePath)
	PanicOn(err)

	repo := context.Args().First()
	if repo == "" {
		// sync/update all repos in Yumfile
		if err := yumfile.SyncAll(); err != nil {
			Fatalf(err, "Error running Yumfile")
		}
	} else {
		// sync/update one repo in the Yumfile
		mirror := yumfile.GetRepoByID(repo)
		if mirror == nil {
			Fatalf(nil, "No such repo found in Yumfile: %s", repo)
		}

		if err := yumfile.SyncRepos([]Repo{*mirror}); err != nil {
			Fatalf(err, "Error syncronizing repo '%s'", mirror.ID)
		}
	}
}

func ActionServe(context *cli.Context) {
	addr := context.String("listen")
	path := context.Args().First()
	if path == "" {
		path = "."
	}

	serve(path, addr)
}

func PanicOn(err error) {
	if err != nil {
		Fatalf(err, "Fatal error")
	}
}

func NewErrorf(format string, a ...interface{}) error {
	return errors.New(fmt.Sprintf(format, a...))
}
