---
layout: page
title: y10k
menu: About
permalink: /
---

*Simplified Yum repository management from the year 10,000 AD*

Y10K is a tool to deploy Yum/RPM repositories and mirrors in your local
environment using settings described in a INI formatted `Yumfile`.

It is a wrapper for `reposync` and `createrepo` but takes the hard work out of
writing shell scripts for each of your mirrors. It also provides an abstraction
to ease management with configuration management tools like Puppet and Chef.

What about Pulp/Satellite/Other? I wanted a cron job that syncronizes my repos
with the upstreams into a folder shared in Apache/nginx. I don't want to deploy
a database, server, agents, configure channel registrations, etc. etc.

Y10K is inspired by tools such as Puppet's [R10K](https://github.com/puppetlabs/r10k)
and Ruby's [Bundler](http://bundler.io/gemfile.html).


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