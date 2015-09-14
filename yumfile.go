package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strings"
)

type Yumfile struct {
	Repos           []Repo
	LocalPathPrefix string
}

var boolMap = map[bool]int{
	true:  1,
	false: 0,
}

var (
	sectionHeadPattern = regexp.MustCompile("^\\[(.*)\\]")
	keyValPattern      = regexp.MustCompile("^(\\w+)\\s*=\\s*(.*)")
	commentPattern     = regexp.MustCompile("(^$)|(^\\s+$)|(^#)|(^;)")
)

// LoadYumfile loads a Yumfile from disk
func LoadYumfile(path string) (*Yumfile, error) {
	Dprintf("Loading Yumfile: %s\n", path)

	yumfile := Yumfile{}

	// open file
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// read each line
	n := 0
	scanner := bufio.NewScanner(f)
	var repo *Repo = nil
	for scanner.Scan() {
		n++
		s := scanner.Text()

		if matches := sectionHeadPattern.FindAllStringSubmatch(s, -1); len(matches) > 0 {
			// line is a [section header]
			id := matches[0][1]

			// append previous section
			if repo != nil {
				yumfile.Repos = append(yumfile.Repos, *repo)
			}

			// create new repo def
			repo = NewRepo()

			repo.YumfilePath = path
			repo.YumfileLineNo = n
			repo.ID = id
		} else if matches := keyValPattern.FindAllStringSubmatch(s, -1); len(matches) > 0 {
			// line is a key=val pair
			key := matches[0][1]
			val := matches[0][2]

			if repo == nil {
				// global key/val pair
				switch key {
				case "pathprefix":
					yumfile.LocalPathPrefix = val

				default:
					return nil, NewErrorf("Syntax error in Yumfile on line %d: Unknown key: %s", n, key)
				}
			} else {
				// add key/val to current repo
				switch key {
				case "localpath":
					repo.LocalPath = val

				case "arch":
					repo.Architecture = val

				case "newonly":
					if b, err := strToBool(val); err != nil {
						return nil, NewErrorf("Syntax error in Yumfile on line %d: %s", n, err.Error())
					} else {
						repo.NewOnly = b
					}

				case "sources":
					if b, err := strToBool(val); err != nil {
						return nil, NewErrorf("Syntax error in Yumfile on line %d: %s", n, err.Error())
					} else {
						repo.IncludeSources = b
					}

				case "deleteremoved":
					if b, err := strToBool(val); err != nil {
						return nil, NewErrorf("Syntax error in Yumfile on line %d: %s", n, err.Error())
					} else {
						repo.DeleteRemoved = b
					}

				case "gpgcheck":
					if b, err := strToBool(val); err != nil {
						return nil, NewErrorf("Syntax error in Yumfile on line %d: %s", n, err.Error())
					} else {
						repo.GPGCheck = b

						// pass through to yum
						repo.Parameters[key] = val
					}

				case "checksum":
					repo.Checksum = val

				default:
					repo.Parameters[key] = val
				}
			}
		} else if commentPattern.MatchString(s) {
			// ignore line
		} else {
			return nil, NewErrorf("Syntax error in Yumfile on line %d: %s", n, s)
		}
	}

	// add last scanned repo
	if repo != nil {
		yumfile.Repos = append(yumfile.Repos, *repo)
	}

	// check for scan errors
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// validate
	if err = yumfile.Validate(); err != nil {
		return nil, err
	}

	return &yumfile, nil
}

// Validate ensures all Yumfile fields contain valid values
func (c *Yumfile) Validate() error {
	for i, repo := range c.Repos {
		if err := repo.Validate(); err != nil {
			return err
		}

		// append path prefix to each repo
		if c.LocalPathPrefix != "" {
			c.Repos[i].LocalPath = fmt.Sprintf("%s/%s", c.LocalPathPrefix, repo.LocalPath)
		}

		// TODO: Prevent duplicate local paths and repo IDs
	}

	return nil
}

func (c *Yumfile) GetRepoByID(id string) *Repo {
	for _, repo := range c.Repos {
		if repo.ID == id {
			return &repo
		}
	}

	return nil
}

func (c *Yumfile) SyncAll() error {
	return c.Sync(c.Repos)
}

// Sync processes all repository mirrors defined in a Yumfile
func (c *Yumfile) Sync(repos []Repo) error {
	//if err := c.installYumConf(repos); err != nil {
	//	return err
	//}

	for _, repo := range repos {
		if err := c.installYumConf(&repo); err != nil {
			Errorf(err, "Failed to create yum.conf for %s", repo.ID)
		} else {
			if err := c.reposync(&repo); err != nil {
				Errorf(err, "Failed to download updates for %s", repo.ID)
			} else {
				if err := c.createrepo(&repo); err != nil {
					Errorf(err, "Failed to update repo database for %s", repo.ID)
				}
			}
		}
	}

	return nil
}

func (c *Yumfile) installYumConf(repo *Repo) error {
	Dprintf("Installing yum.conf file: %s\n", TmpYumConfPath)

	// create temp path
	if err := os.MkdirAll(TmpBasePath, 0750); err != nil {
		return err
	}

	// create config file
	f, err := os.Create(TmpYumConfPath)
	if err != nil {
		return err
	}
	defer f.Close()

	// global yum conf
	fmt.Fprintf(f, "[main]\n")
	fmt.Fprintf(f, "cachedir=%s\n", TmpYumCachePath)
	fmt.Fprintf(f, "debuglevel=10\n")
	fmt.Fprintf(f, "exactarch=0\n")
	fmt.Fprintf(f, "gpgcheck=0\n")
	fmt.Fprintf(f, "keepcache=0\n")
	fmt.Fprintf(f, "logfile=%s\n", TmpYumLogFile)
	fmt.Fprintf(f, "plugins=0\n")
	fmt.Fprintf(f, "reposdir=\n")
	fmt.Fprintf(f, "rpmverbosity=debug\n")
	fmt.Fprintf(f, "timeout=5\n")
	fmt.Fprintf(f, "\n")

	// append repo config
	fmt.Fprintf(f, "[%s]\n", repo.ID)
	for key, val := range repo.Parameters {
		fmt.Fprintf(f, "%s=%s\n", key, val)
	}
	fmt.Fprintf(f, "\n")

	return nil
}

func (c *Yumfile) reposync(repo *Repo) error {
	Printf("Syncronizing repo: %s\n", repo.ID)

	// compute args for reposync command
	args := []string{
		fmt.Sprintf("--config=%s", TmpYumConfPath),
		fmt.Sprintf("--repoid=%s", repo.ID),
		"--norepopath",
		"--downloadcomps",
		"--download-metadata",
	}

	if QuietMode {
		args = append(args, "--quiet")
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

	if repo.Architecture != "" {
		args = append(args, fmt.Sprintf("--arch=%s", repo.Architecture))
	}

	if repo.LocalPath != "" {
		args = append(args, fmt.Sprintf("--download_path=%s", repo.LocalPath))
	} else {
		args = append(args, fmt.Sprintf("--download_path=./%s", repo.ID))
	}

	// execute and capture output
	if err := Exec("reposync", args...); err != nil {
		return err
	}

	return nil
}

func (c *Yumfile) createrepo(repo *Repo) error {
	Printf("Updating repo database: %s\n", repo.ID)

	// compute args for createrepo command
	args := []string{
		"--update",
		"--database",
		"--checkts",
		fmt.Sprintf("--workers=%d", runtime.NumCPU()*2),
	}

	if QuietMode {
		args = append(args, "--quiet")
	} else {
		args = append(args, "--profile")
	}

	// debug switches
	if DebugMode {
		args = append(args, "--verbose")
	}

	// non-default checksum type
	if repo.Checksum != "" {
		args = append(args, fmt.Sprintf("--checksum=%s", repo.Checksum))
	}

	// path to create repo for
	if repo.LocalPath != "" {
		args = append(args, repo.LocalPath)
	} else {
		args = append(args, fmt.Sprintf("./%s", repo.ID))
	}

	// execute and capture output
	if err := Exec("createrepo", args...); err != nil {
		return err
	}

	return nil
}

func strToBool(s string) (bool, error) {
	lc := strings.ToLower(s)

	switch lc {
	case "1", "true", "enabled", "yes":
		return true, nil

	case "0", "false", "disabled", "no":
		return false, nil
	}

	return false, NewErrorf("Invalid boolean value: %s", s)
}
