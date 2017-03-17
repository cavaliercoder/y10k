package main

import (
	"./yum"
	"fmt"
	"github.com/cavaliercoder/go-rpm"
)

// FilterPackages returns a list of packages filtered according the repo's
// settings.
func FilterPackages(repo *Repo, packages yum.PackageEntries) yum.PackageEntries {
	newest := make(map[string]*yum.PackageEntry, 0)

	// calculate which packages are the latest
	if repo.NewOnly {
		for i, p := range packages {
			// index on name and architecture
			id := fmt.Sprint("%s.%s", p.Name(), p.Architecture())

			// lookup previous index
			if n, ok := newest[id]; ok {
				// compare version with previous index
				if 1 == rpm.VersionCompare(rpm.PackageVersion(&p), rpm.PackageVersion(n)) {
					newest[id] = &packages[i]
				}
			} else {
				// add new index for this package
				newest[id] = &packages[i]
			}
		}

		// replace packages with only the latest packages
		i := 0
		packages = make(yum.PackageEntries, len(newest))
		for _, p := range newest {
			packages[i] = *p
			i++
		}
	}

	// filter the package list
	filtered := make(yum.PackageEntries, 0)
	for _, p := range packages {
		include := true

		// filter by architecture
		if repo.Architecture != "" {
			if p.Architecture() != repo.Architecture {
				include = false
			}
		}

		// filter by minimum build date
		if !repo.MinDate.IsZero() {
			if p.BuildTime().Before(repo.MinDate) {
				include = false
			}
		}

		// filter by maximum build date
		if !repo.MaxDate.IsZero() {
			if p.BuildTime().After(repo.MaxDate) {
				include = false
			}
		}

		// append to output
		if include {
			filtered = append(filtered, p)
		}
	}

	return filtered
}
