package main

import (
	"fmt"
	"os"
)

const (
	repoPrefix   = "y10k-temp-"
	tempRepoFile = "/etc/yum.repos.d/y10k-temp.repo"
)

type Yumfile struct {
	YumRepos []RepoMirror `json:"repoMirrors"`
}

type RepoMirror struct {
	YumRepo

	CachePath      string `json:"cachePath,omitempty"`
	EnablePlugins  bool   `json:"enablePlugins,omitempty"`
	IncludeSources bool   `json:"includeSources,omitempty"`
	LocalPath      string `json:"localPath,omitempty"`
	NewOnly        bool   `json:"newOnly,omitempty"`
	DeleteRemoved  bool   `json:"deleteRemoved,omitempty"`
	GPGCheck       bool   `json:"gpgCheck,omitempty"`
}

var boolMap = map[bool]int{
	true:  1,
	false: 0,
}

func (c *Yumfile) installRepoFile() error {
	// create repo file
	f, err := os.Create(tempRepoFile)
	if err != nil {
		return err
	}
	defer f.Close()

	// write each repo
	Dprintf("Creating temp repo config: %s\n", tempRepoFile)
	for _, repo := range c.YumRepos {
		fmt.Fprintf(f, "[%s%s]\n", repoPrefix, repo.ID)

		if repo.Name != "" {
			fmt.Fprintf(f, "name=%s\n", repo.Name)
		}

		if repo.MirrorListURL != "" {
			fmt.Fprintf(f, "mirrorlist=%s\n", repo.MirrorListURL)
		} else if repo.BaseURL != "" {
			fmt.Fprintf(f, "baseurl=%s\n", repo.BaseURL)
		}

		fmt.Fprintf(f, "enabled=%d\n", boolMap[repo.Enabled])
		fmt.Fprintf(f, "gpgcheck=%d\n", boolMap[repo.GPGCheck])

		if repo.GPGKeyPath != "" {
			fmt.Fprintf(f, "gpgkey=%s\n", repo.GPGKeyPath)
		}

		fmt.Fprintf(f, "\n")
	}

	return nil
}

func (c *Yumfile) deleteRepoFile() error {
	Dprintf("Deleting temp repo config: %s\n", tempRepoFile)
	return os.Remove(tempRepoFile)
}

func (c *Yumfile) Sync() error {
	// create repo file
	err := c.installRepoFile()
	if err != nil {
		return err
	}
	defer c.deleteRepoFile()

	// sync each repo
	for _, repo := range c.YumRepos {
		Printf("Syncronizing repo: %s\n", repo.ID)

		// compute args for reposync command
		args := []string{
			fmt.Sprintf("--repoid=%s%s", repoPrefix, repo.ID),
			"--norepopath",
			"--quiet", // unfortunately reposync uses lots of control chars...
		}

		if repo.NewOnly {
			args = append(args, "--newest-only")
		}

		if repo.IncludeSources {
			args = append(args, "--source")
		}

		if repo.DeleteRemoved {
			args = append(args, "--delete")
		}

		if repo.GPGCheck {
			args = append(args, "--gpgcheck")
		}

		if repo.LocalPath != "" {
			args = append(args, fmt.Sprintf("--download_path=%s", repo.LocalPath))
		} else {
			args = append(args, fmt.Sprintf("--download_path=./%s", repo.ID))
		}

		// execute and capture output
		Exec("reposync", args...)
	}

	return nil
}

func (c *Yumfile) Update() error {
	// update each repo database
	for _, repo := range c.YumRepos {
		Printf("Updating repo database: %s\n", repo.ID)

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
		if repo.LocalPath != "" {
			args = append(args, repo.LocalPath)
		} else {
			args = append(args, fmt.Sprintf("./%s", repo.ID))
		}

		// execute and capture output
		Exec("createrepo", args...)
	}

	return nil
}
