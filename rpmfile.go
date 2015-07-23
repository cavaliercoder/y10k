package main

import (
	"time"
)

type RpmFile struct {
	Name      string
	Version   string
	Release   string
	Path      string
	BuildTime time.Time
	FileTime  time.Time
}
