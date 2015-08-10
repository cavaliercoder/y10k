---
layout: page
title: Yumfile format
menu: Yumfile
permalink: /yumfile/
---

A Yumfile is an INI section/key/val document, similar to `yum.conf`. Global
directives are defined first. Repository mirrors are defined in discrete
sections, starting with an identifier `[header]`.

Blank lines and lines starting with a `#` or `;` are ignored.

The following Yumfile example will syncronize two upstream yum repos into local
path `/var/www/html/pub/centos/7/*`. Only 64bit packages will be downloaded for
both repos, and only the most recent version of each package will be downloaded
for `centos-7-x86_64-base`.

{% highlight ini %}
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
newonly=yes

[centos-7-x86_64-updates]
name=CentOS 7 x86_64 Updates
mirrorlist=http://mirrorlist.centos.org/?release=7&arch=x86_64&repo=updates
localpath=centos/7/updates/x86_64
arch=x86_64

{% endhighlight %}

	$ y10k yumfile -f example.conf sync

## Global options

 * `pathprefix` - all repositories will be synronized into a local directory
   relative to this path, instead of the current working directory

## Repository options

For every repository you wish to mirror locally, you must define a repository
configuration section. Each section is delimited with an INI style `[header]`
which is used as the identifier for the repository in Yum configuration. The ID
is also used as the local path for the mirror unless the `localpath` directive
is set.

 * `name` - a friendly name for the repository

 * `mirrorlist` - URL of the upstream repository mirror list. One of
   `mirrorlist` or `baseurl` must be specified

 * `baseurl` - URL of the upstream repository

 * `localpath` - local path where the upstream respository will be syncronized
   to. Defaults to the ID specified in the repository section header
 
 * `arch` - syncronize only the specified machine architecture. Passed to the
   `--arch` argument of `reposync`

 * `newonly` - syncronize only the most recent version of upstream packages.
   Passed to the `--newest-only` argument of `reposync`
