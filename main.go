package main

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/codegangsta/cli"
	"os"
	"os/exec"
	"strings"
)

var (
	DebugMode   = false
	YumfilePath = ""
)

func main() {
	// route request
	app := cli.NewApp()
	app.Name = "y10k"
	app.Version = "0.1.0"
	app.Usage = "simplied yum mirror management"
	app.Flags = []cli.Flag{
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
		DebugMode = context.GlobalBool("debug")

		// check system health
		if err := HealthCheck(); err != nil {
			Fatalf(err, "Health check failed")
		}

		return nil
	}

	app.Run(os.Args)
}

func ActionYumfileValidate(context *cli.Context) {
	yumfile, err := LoadYumfile(YumfilePath)
	PanicOn(err)
	Printf("Yumfile appears valid (%d repos)\n", len(yumfile.YumRepos))
}

func ActionYumfileList(context *cli.Context) {
	yumfile, err := LoadYumfile(YumfilePath)
	PanicOn(err)

	repoCount := len(yumfile.YumRepos)
	padding := (len(fmt.Sprintf("%d", repoCount)) * 2) + 1
	for i, repo := range yumfile.YumRepos {
		fmt.Printf("%*s %s\n", padding, fmt.Sprintf("%d/%d", i+1, repoCount), repo.YumRepo.ID)
	}
}

func ActionYumfileSync(context *cli.Context) {
	yumfile, err := LoadYumfile(YumfilePath)
	PanicOn(err)

	repo := context.Args().First()
	if repo == "" {
		PanicOn(yumfile.Sync())
	} else {
		mirror := yumfile.Repo(repo)
		if mirror == nil {
			Fatalf(nil, "No such repo found in Yumfile: %s", repo)
		}

		PanicOn(mirror.Sync())
		PanicOn(mirror.Update())
	}
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

func Errorf(format string, a ...interface{}) error {
	return errors.New(fmt.Sprintf(format, a...))
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
