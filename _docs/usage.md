---
layout: page
title: Usage
menu: Usage
permalink: /usage/
---

      NAME:
         y10k - simplied yum mirror management

      USAGE:
         y10k [global options] command [command options] [arguments...]

      VERSION:
         0.1.0

      AUTHOR:
        Ryan Armstrong - <ryan@cavaliercoder.com>

      COMMANDS:
         yumfile  work with a Yumfile
         version  print the version of y10k
         help, h  Shows a list of commands or help for one command
         
      GLOBAL OPTIONS:
         --logfile, -l  redirect output to a log file [$Y10K_LOGFILE]
         --debug, -d    print debug output [$Y10K_DEBUG]
         --help, -h     show help
         --version, -v  print the version
   

## Examples

By default, all `yumfile` subcommands will assume there is file named `Yumfile`
in the current working directory. The Yumfile format is detailed [here]({{ site.baseurl }}/yumfile/).

To specify a different file path for your Yumfile, use `-f`:

    $ y10k yumfile -f /path/to/file [subcommand] [options]

To list the available repositories in a Yumfile:

    $ y10k yumfile list

To syncronize one or all of the repositories in a Yumfile, use the `sync`
command. 

    $ y10k yumfile sync

By default, all repositories are syncronized into a subdirectory of the current
working directory. Each directory is named after the repository `[id]` unless
the the `localpath` directive is set.

If the global `pathprefix` directive is set in the Yumfile, all repositories
are syncronized relative to that path, instead of the current working
directory.

You may choose to syncronize one repository at a time:

    $ y10k yumfile sync [repo-id]

