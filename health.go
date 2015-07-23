package main

import (
	"fmt"
	"os/exec"
	"regexp"
)

var yumVersionPattern = regexp.MustCompile("^(.*)")
var rpmVersionPattern = regexp.MustCompile("^RPM version (.*)")
var createrepoVersionPattern = regexp.MustCompile("^createrepo\\s+(.*)")

func HealthCheck() error {
	var colWidth int = 16
	var out []byte = []byte{}
	var msg string = ""
	var err error = nil

	Dprintf("Checking dependencies:\n")

	// check for yum
	out, err = exec.Command("yum", "--version").CombinedOutput()
	if err != nil {
		return err
	} else {
		// extract version string
		matches := yumVersionPattern.FindAllStringSubmatch(string(out), -1)
		if len(matches) > 0 {
			msg = fmt.Sprintf("installed (v%s)", matches[0][1])
		} else {
			msg = string(out)
		}
	}

	Dprintf("  %-*s%s\n", colWidth, "yum:", msg)

	// check for rpm
	out, err = exec.Command("rpm", "--version").CombinedOutput()
	if err != nil {
		return err
	} else {
		// extract version string
		matches := rpmVersionPattern.FindAllStringSubmatch(string(out), -1)
		if len(matches) > 0 {
			msg = fmt.Sprintf("installed (v%s)", matches[0][1])
		} else {
			msg = string(out)
		}
	}

	Dprintf("  %-*s%s\n", colWidth, "rpm:", msg)

	// check for reposync
	_, err = exec.Command("reposync", "--help").CombinedOutput()
	if err != nil {
		return err
	} else {
		msg = "installed"
	}

	Dprintf("  %-*s%s\n", colWidth, "reposync:", msg)

	// check for createrepo
	out, err = exec.Command("createrepo", "--version").CombinedOutput()
	if err != nil {
		return err
	} else {
		// extract version string
		matches := createrepoVersionPattern.FindAllStringSubmatch(string(out), -1)
		if len(matches) > 0 {
			msg = fmt.Sprintf("installed (v%s)", matches[0][1])
		} else {
			msg = string(out)
		}
	}

	Dprintf("  %-*s%s\n", colWidth, "createrepo:", msg)

	return nil
}
