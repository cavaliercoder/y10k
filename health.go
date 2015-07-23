package main

import (
	"fmt"
	"os/exec"
	"regexp"
)

var createrepoVersionPattern = regexp.MustCompile("^createrepo\\s+(.*)")
var repoqueryVersionPattern = regexp.MustCompile("^Repoquery version (.*)")
var rpmVersionPattern = regexp.MustCompile("^RPM version (.*)")
var yumVersionPattern = regexp.MustCompile("^(.*)")

func HealthCheck() error {
	var colWidth int = 12
	var out []byte = []byte{}
	var msg string = ""
	var err error = nil
	var cmd *exec.Cmd = nil

	Dprintf("Checking dependencies:\n")

	// check for yum
	cmd = exec.Command("yum", "--version")
	out, err = cmd.CombinedOutput()
	if err != nil {
		return err
	} else {
		// extract version string
		matches := yumVersionPattern.FindAllStringSubmatch(string(out), -1)
		if len(matches) > 0 {
			msg = fmt.Sprintf("%s (v%s)", cmd.Path, matches[0][1])
		} else {
			msg = string(out)
		}
	}

	Dprintf("  %-*s%s\n", colWidth, "yum:", msg)

	// check for rpm
	cmd = exec.Command("rpm", "--version")
	out, err = cmd.CombinedOutput()
	if err != nil {
		return err
	} else {
		// extract version string
		matches := rpmVersionPattern.FindAllStringSubmatch(string(out), -1)
		if len(matches) > 0 {
			msg = fmt.Sprintf("%s (v%s)", cmd.Path, matches[0][1])
		} else {
			msg = string(out)
		}
	}

	Dprintf("  %-*s%s\n", colWidth, "rpm:", msg)

	// check for reposync
	cmd = exec.Command("reposync", "--help")
	_, err = cmd.CombinedOutput()
	if err != nil {
		return err
	} else {
		msg = cmd.Path
	}
	Dprintf("  %-*s%s\n", colWidth, "reposync:", msg)

	// check for createrepo
	cmd = exec.Command("createrepo", "--version")
	out, err = cmd.CombinedOutput()
	if err != nil {
		return err
	} else {
		// extract version string
		matches := createrepoVersionPattern.FindAllStringSubmatch(string(out), -1)
		if len(matches) > 0 {
			msg = fmt.Sprintf("%s (v%s)", cmd.Path, matches[0][1])
		} else {
			msg = string(out)
		}
	}

	Dprintf("  %-*s%s\n", colWidth, "createrepo:", msg)

	// check for repoquery
	cmd = exec.Command("repoquery", "--version")
	out, err = cmd.CombinedOutput()
	if err != nil {
		return err
	} else {
		// extract version string
		matches := repoqueryVersionPattern.FindAllStringSubmatch(string(out), -1)
		if len(matches) > 0 {
			msg = fmt.Sprintf("%s (v%s)", cmd.Path, matches[0][1])
		} else {
			msg = string(out)
		}
	}

	Dprintf("  %-*s%s\n", colWidth, "repoquery:", msg)

	return nil
}
