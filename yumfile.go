package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
				case "baseurl":
					repo.BaseURL = val

				case "mirrorlist":
					repo.MirrorURL = val

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

				case "groupfile":
					repo.Groupfile = val

				default:
					repo.Parameters[key] = val
				}
			}
		} else if !commentPattern.MatchString(s) {
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

// Sync processes all repository mirrors defined in a Yumfile
func (c *Yumfile) SyncRepos(repos []Repo) error {
	// TODO: make sure cache path is always unique for all repos with same ID
	for _, repo := range repos {
		cachedir := filepath.Join(TmpYumCachePath, repo.ID)
		packagedir := filepath.Join(c.LocalPathPrefix, repo.LocalPath)
		if err := repo.Sync(cachedir, packagedir); err != nil {
			Errorf(err, "Error synchronizing repo %v", repo)
		}
	}

	return nil
}

func (c *Yumfile) SyncAll() error {
	return c.SyncRepos(c.Repos)
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
