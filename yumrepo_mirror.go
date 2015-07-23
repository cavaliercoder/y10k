package main

import (
	"fmt"
	"os"
)

const (
	repoPrefix     = "tmp-y10k-"
	repoFilePrefix = "tmp-y10k-"
	repoFileSuffix = ".repo"
	repoFileDir    = "/etc/yum.repos.d"
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
}

func (c *YumRepoMirror) Validate() error {
	// TODO validate mirror fields before processing
	return nil
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
