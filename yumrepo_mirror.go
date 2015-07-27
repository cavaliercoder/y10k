package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	repoPrefix      = "tmp-y10k-"
	repoFilePrefix  = "tmp-y10k-"
	repoFileSuffix  = ".repo"
	repoFileDir     = "/etc/yum.repos.d"
	repoqueryFormat = "%{name} %{version} %{release} %{buildtime} %{filetime}"
)

type YumRepoMirror struct {
	YumRepo        YumRepo `json:"upstream,omitempty"`
	CachePath      string  `json:"cachePath,omitempty"`
	EnablePlugins  bool    `json:"enablePlugins,omitempty"`
	IncludeSources bool    `json:"includeSources,omitempty"`
	LocalPath      string  `json:"localPath,omitempty"`
	NewOnly        bool    `json:"newOnly,omitempty"`
	DeleteRemoved  bool    `json:"deleteRemoved,omitempty"`
	GPGCheck       bool    `json:"gpgCheck,omitempty"`
	Architecture   string  `json:"arch,omitempty"`
	YumfilePath    string  `json:"-"`
	YumfileLineNo  int     `json:"_"`
}

func (c *YumRepoMirror) Validate() error {
	if c.YumRepo.ID == "" {
		return Errorf("Upstream repository has no ID specified (in %s:%d)", c.YumfilePath, c.YumfileLineNo)
	}

	if c.YumRepo.MirrorListURL == "" && c.YumRepo.BaseURL == "" {
		return Errorf("Upstream repository for '%s' has no mirror list or base URL (in %s:%d)", c.YumRepo.ID, c.YumfilePath, c.YumfileLineNo)
	}

	return nil
}

func (c *YumRepoMirror) FullLocalPath() string {
	path, _ := filepath.Abs(c.LocalPath)
	return path
}

func (c *YumRepoMirror) repoFilePath() string {
	return fmt.Sprintf("%s/%s%s%s", repoFileDir, repoFilePrefix, c.YumRepo.ID, repoFileSuffix)
}

func (c *YumRepoMirror) repoName() string {
	return fmt.Sprintf("%s%s", repoPrefix, c.YumRepo.ID)
}

func (c *YumRepoMirror) installRepoFile() error {
	repoName := c.repoName()
	repoFilePath := c.repoFilePath()

	Dprintf("Installing repo file: %s\n", repoFilePath)

	// create repo file
	f, err := os.Create(repoFilePath)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintf(f, "[%s]\n", repoName)

	if c.YumRepo.Name != "" {
		fmt.Fprintf(f, "name=%s\n", c.YumRepo.Name)
	}

	if c.YumRepo.MirrorListURL != "" {
		fmt.Fprintf(f, "mirrorlist=%s\n", c.YumRepo.MirrorListURL)
	} else if c.YumRepo.BaseURL != "" {
		fmt.Fprintf(f, "baseurl=%s\n", c.YumRepo.BaseURL)
	}

	fmt.Fprintf(f, "enabled=%d\n", boolMap[c.YumRepo.Enabled])
	fmt.Fprintf(f, "gpgcheck=%d\n", boolMap[c.YumRepo.GPGCheck])

	if c.YumRepo.GPGKeyPath != "" {
		fmt.Fprintf(f, "gpgkey=%s\n", c.YumRepo.GPGKeyPath)
	}

	fmt.Fprintf(f, "\n")

	return nil
}

func (c *YumRepoMirror) deleteRepoFile() error {
	repoFilePath := c.repoFilePath()
	Dprintf("Deleting repo file: %s\n", repoFilePath)
	return os.Remove(repoFilePath)
}

func (c *YumRepoMirror) Sync() error {
	// create repo file
	err := c.installRepoFile()
	if err != nil {
		return err
	}
	defer c.deleteRepoFile()

	Printf("Syncronizing repo: %s\n", c.YumRepo.ID)

	// compute args for reposync command
	args := []string{
		fmt.Sprintf("--repoid=%s%s", repoPrefix, c.YumRepo.ID),
		"--norepopath",
		"--quiet", // unfortunately reposync uses lots of control chars...
	}

	if c.NewOnly {
		args = append(args, "--newest-only")
	}

	if c.IncludeSources {
		args = append(args, "--source")
	}

	if c.DeleteRemoved {
		args = append(args, "--delete")
	}

	if c.GPGCheck {
		args = append(args, "--gpgcheck")
	}

	if c.Architecture != "" {
		args = append(args, fmt.Sprint("--arch=%s", c.Architecture))
	}

	if c.LocalPath != "" {
		args = append(args, fmt.Sprintf("--download_path=%s", c.LocalPath))
	} else {
		args = append(args, fmt.Sprintf("--download_path=./%s", c.YumRepo.ID))
	}

	// execute and capture output
	Exec("reposync", args...)

	return nil
}

func (c *YumRepoMirror) Update() error {
	Printf("Updating repo database: %s\n", c.YumRepo.ID)

	// compute args for createrepo command
	args := []string{
		"--update",
		"--database",
		"--checkts",
		"--workers=1",
	}

	// debug switches
	if DebugMode {
		args = append(args, "--verbose", "--profile")
	}

	// path to create repo for
	if c.LocalPath != "" {
		args = append(args, c.LocalPath)
	} else {
		args = append(args, fmt.Sprintf("./%s", c.YumRepo.ID))
	}

	// execute and capture output
	Exec("createrepo", args...)

	return nil
}

func (c *YumRepoMirror) QueryAll() ([]RpmFile, error) {
	results := []RpmFile{}

	// build command and args
	args := []string{
		"--all",
		"--show-duplicates",
		"--disablerepo=*",
		fmt.Sprintf("--queryformat=%s", repoqueryFormat),
		fmt.Sprintf("--enablerepo=%s", c.YumRepo.ID),
		fmt.Sprintf("--repofrompath=%s,file://%s", c.YumRepo.ID, c.FullLocalPath()),
	}
	cmd := exec.Command("repoquery", args...)
	Dprintf("exec: %s %s\n", cmd.Path, strings.Join(args, " "))

	// attach to stdout
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return results, err
	}

	scanner := bufio.NewScanner(stdout)
	go func() {
		for scanner.Scan() {
			fields := strings.Split(scanner.Text(), " ")
			buildTime, err := strconv.Atoi(fields[3])
			if err != nil {
				Dprintf("Cannot convert string '%s' to an integer\n", fields[3])
			}

			fileTime, err := strconv.Atoi(fields[4])
			if err != nil {
				Dprintf("Cannot convert string '%s' to an integer\n", fields[4])
			}

			rpm := RpmFile{
				Name:      fields[0],
				Version:   fields[1],
				Release:   fields[2],
				BuildTime: time.Unix(int64(buildTime), 0),
				FileTime:  time.Unix(int64(fileTime), 0),
			}

			results = append(results, rpm)
		}
	}()

	// execute and wait
	if err := cmd.Start(); err != nil {
		return results, err
	}

	if err := cmd.Wait(); err != nil {
		return results, err
	}

	return results, nil
}
