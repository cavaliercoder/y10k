# y10k [![Build Status](https://travis-ci.org/cavaliercoder/y10k.svg?branch=master)](https://travis-ci.org/cavaliercoder/y10k)

*Simplified Yum repository management from the year 10,000 AD*

y10k is a tool to deploy Yum/RPM repositories and mirrors in your local
environment using settings described in a INI formatted `Yumfile`.

It is a wrapper for `reposync` and `createrepo` but takes the hard work out of
writing shell scripts for each of your mirrors. It also provides an abstraction
to ease management with configuration management tools like Puppet and Chef.

What about Pulp/Satellite/Other? I wanted a cron job that syncronizes my repos
with the upstreams into a folder shared in Apache/nginx. I don't want to deploy
a database, server, agents, configure channel registrations, etc. etc.

y10k is inspired by tools such as Puppet's [R10K](https://github.com/puppetlabs/r10k)
and Ruby's [Bundler](http://bundler.io/gemfile.html).

Hey cool there's [documentation](http://cavaliercoder.github.io/y10k).

Oh, and you can [download y10k](https://sourceforge.net/projects/y10k/files/latest/download)
precompiled binaries.

## Usage

```
NAME:
   y10k - simplified yum mirror management

USAGE:
   y10k [global options] command [command options] [arguments...]

VERSION:
   0.3.0

AUTHOR(S):
   Ryan Armstrong <ryan@cavaliercoder.com>

COMMANDS:
   yumfile	work with a Yumfile
   version	print the version of y10k
   help, h	Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --logfile, -l 		redirect output to a log file [$Y10K_LOGFILE]
   --quiet, -q			less verbose
   --debug, -d			print debug output [$Y10K_DEBUG]
   --tmppath, -t "/tmp/y10k"	path to y10k temporary objects [$Y10K_TMPPATH]
   --help, -h			show help
   --version, -v		print the version

```

## Yumfile format

```ini
#
# Global settings
#
pathprefix=/var/www/html/pub

#
# CentOS 7 x86_64 mirror
#
[centos-7-x86_64-base]
name=CentOS 7 x86_64 Base
mirrorlist=http://mirrorlist.centos.orgbroken/?release=7&arch=x86_64&repo=os
localpath=centos/7/os/x86_64
arch=x86_64

[centos-7-x86_64-updates]
name=CentOS 7 x86_64 Updates
mirrorlist=http://mirrorlist.centos.org/?release=7&arch=x86_64&repo=updates
localpath=centos/7/updates/x86_64
arch=x86_64

```  

## License

Y10K Copyright (C) 2014 Ryan Armstrong (ryan@cavaliercoder.com)

This program is free software: you can redistribute it and/or modify it under
the terms of the GNU General Public License as published by the Free Software
Foundation, either version 3 of the License, or (at your option) any later
version.

This program is distributed in the hope that it will be useful, but WITHOUT ANY
WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A
PARTICULAR PURPOSE. See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License along with
this program. If not, see http://www.gnu.org/licenses/.
