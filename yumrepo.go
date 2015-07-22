// See: http://linux.die.net/man/5/yum.conf
package main

import (
	"bufio"
	"os/exec"
	"regexp"
	"time"
)

type YumRepo struct {
	ID         string
	Name       string
	Enabled    bool
	Revision   string
	UpdateDate time.Time
	BaseURL    string
	ExpireDate time.Time
	Filename   string
}

var yumTimeLayout = "Mon Jan 2 15:04:05 2006"
var fieldPattern = regexp.MustCompile("^Repo-([^\\s]*)\\s*:\\s*(.*)$")

func GetInstalledRepos() ([]YumRepo, error) {
	repos := make([]YumRepo, 0)

	// get repo list from yum
	cmd := exec.Command("yum", "repolist", "all", "-v")
	reader, err := cmd.StdoutPipe()
	if err != nil {
		return repos, err
	}

	scanner := bufio.NewScanner(reader)
	go func() {
		var repo *YumRepo = nil
		for scanner.Scan() {
			line := scanner.Text()
			res := fieldPattern.FindAllStringSubmatch(line, -1)

			if len(res) > 0 {
				key := res[0][1]
				val := res[0][2]

				// is this a new repo?
				if key == "id" {
					// append previous repo
					if repo != nil {
						repos = append(repos, *repo)
					}

					// create new repo
					repo = &YumRepo{
						ID: val,
					}
				} else {
					// update current repo
					switch key {
					case "name":
						repo.Name = val

					case "status":
						repo.Enabled = val == "enabled"

					case "revision":
						repo.Revision = val

					case "baseurl":
						repo.BaseURL = val

					case "filename":
						repo.Filename = val

					case "updated":
						t, _ := time.Parse(yumTimeLayout, val)
						repo.UpdateDate = t
					}
				}
			}
		}

		// append last repo
		if repo != nil {
			repos = append(repos, *repo)
		}
	}()

	err = cmd.Start()
	if err != nil {
		return repos, err
	}

	err = cmd.Wait()
	if err != nil {
		return repos, err
	}

	return repos, nil
}
