package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type Yumfile struct {
	YumRepos        []YumRepoMirror `json:"repos"`
	LocalPathPrefix string          `json:"pathPrefix"`
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

func strToBool(s string) (bool, error) {
	lc := strings.ToLower(s)

	switch lc {
	case "1", "true", "enabled", "yes":
		return true, nil

	case "0", "false", "disabled", "no":
		return false, nil
	}

	return false, errors.New(fmt.Sprintf("Invalid boolean value: %s", s))
}

// LoadYumfile loads a Yumfile from a json formated file
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
	var mirror *YumRepoMirror = nil
	for scanner.Scan() {
		n++
		s := scanner.Text()

		if matches := sectionHeadPattern.FindAllStringSubmatch(s, -1); len(matches) > 0 {
			// line is a [section header]
			id := matches[0][1]

			// append previous section
			if mirror != nil {
				yumfile.YumRepos = append(yumfile.YumRepos, *mirror)
			}

			// create new mirror def
			mirror = &YumRepoMirror{
				YumRepo: YumRepo{
					ID: id,
				},
			}
		} else if matches := keyValPattern.FindAllStringSubmatch(s, -1); len(matches) > 0 {
			// line is a key=val pair
			key := matches[0][1]
			val := matches[0][2]

			if mirror == nil {
				// global key/val pair
				switch key {
				case "pathprefix":
					yumfile.LocalPathPrefix = val

				default:
					return nil, errors.New(fmt.Sprintf("Syntax error in Yumfile on line %d: Unknown key: %s", n, key))
				}
			} else {
				// add key/val to current mirror
				switch key {
				case "name":
					mirror.YumRepo.Name = val
				case "mirrorlist":
					mirror.YumRepo.MirrorListURL = val
				case "baseurl":
					mirror.YumRepo.BaseURL = val
				case "localpath":
					mirror.LocalPath = val
				case "newonly":
					if b, err := strToBool(val); err != nil {
						return nil, errors.New(fmt.Sprintf("Syntax error in Yumfile on line %d: %s", n, err.Error()))
					} else {
						mirror.NewOnly = b
					}
				default:
					return nil, errors.New(fmt.Sprintf("Syntax error in Yumfile on line %d: Unknown key: %s", n, key))
				}
			}
		} else if commentPattern.MatchString(s) {
			// ignore line
		} else {
			return nil, errors.New(fmt.Sprintf("Syntax error in Yumfile on line %d: %s", n, s))
		}
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
	for i, mirror := range c.YumRepos {
		if err := mirror.Validate(); err != nil {
			return err
		}

		// append path prefix
		if c.LocalPathPrefix != "" {
			c.YumRepos[i].LocalPath = fmt.Sprintf("%s/%s", c.LocalPathPrefix, mirror.LocalPath)
		}
	}

	return nil
}

func (c *Yumfile) Repo(id string) *YumRepoMirror {
	for _, mirror := range c.YumRepos {
		if mirror.YumRepo.ID == id {
			return &mirror
		}
	}

	return nil
}

// Sync processes all repository mirrors defined in a Yumfile
func (c *Yumfile) Sync() error {
	// sync each repo
	for _, mirror := range c.YumRepos {
		if err := mirror.Sync(); err != nil {
			return err
		}

		if err := mirror.Update(); err != nil {
			return err
		}
	}

	return nil
}
