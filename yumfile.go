package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type Yumfile struct {
	YumRepos        []YumRepoMirror
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

// LoadYumfile loads a Yumfile from a json formated file
func LoadYumfile(path string) (*Yumfile, error) {
	Dprintf("Loading Yumfile: %s\n", path)

	yumfile := Yumfile{}

	// open file
	// TODO: Add support for 'includes' statements in Yumfiles
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
			mirror = NewYumRepoMirror()

			mirror.YumfilePath = path
			mirror.YumfileLineNo = n
			mirror.YumRepo.ID = id
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
					return nil, NewErrorf("Syntax error in Yumfile on line %d: Unknown key: %s", n, key)
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
				case "arch":
					mirror.Architecture = val
				case "newonly":
					if b, err := strToBool(val); err != nil {
						return nil, NewErrorf("Syntax error in Yumfile on line %d: %s", n, err.Error())
					} else {
						mirror.NewOnly = b
					}
				default:
					return nil, NewErrorf("Syntax error in Yumfile on line %d: Unknown key: %s", n, key)
				}
			}
		} else if commentPattern.MatchString(s) {
			// ignore line
		} else {
			return nil, NewErrorf("Syntax error in Yumfile on line %d: %s", n, s)
		}
	}

	// add last scanned mirror
	if mirror != nil {
		yumfile.YumRepos = append(yumfile.YumRepos, *mirror)
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

		// append path prefix to each mirror
		if c.LocalPathPrefix != "" {
			c.YumRepos[i].LocalPath = fmt.Sprintf("%s/%s", c.LocalPathPrefix, mirror.LocalPath)
		}

		// TODO: Prevent duplicate local paths and repo IDs
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
func (c *Yumfile) Sync(breakOnError bool) error {
	// sync each repo
	for _, mirror := range c.YumRepos {
		// sync packages
		if err := mirror.Sync(); err != nil {
			if breakOnError {
				return err
			} else {
				Errorf(err, "Error syncronizing repo '%s", mirror.YumRepo.ID)
			}
		} else {
			// update database
			if err := mirror.Update(); err != nil {
				if breakOnError {
					return err
				} else {
					Errorf(err, "Error updating database for repo '%s'", mirror.YumRepo.ID)
				}
			}
		}
	}

	return nil
}
