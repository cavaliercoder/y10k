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
configuration section. The supported parameters and syntax are identical to
`yum.conf` repositories, except for some additional parameters which are unique
to y10k. These are detailed below.

Each section is delimited with an INI style `[header]` which is used as the
identifier for the repository in Yum configuration. The ID is also used as the
local path for the mirror unless the `localpath` directive is set.

All repositories must declare at least one of the following:

 * `mirrorlist` - URL of the upstream repository mirror list

 * `baseurl` - URL of the upstream repository

The following directives are unique to a Yumfile repository and are used to
configure `reposync` and `createrepo`:

 * `localpath` - local path where the upstream respository will be syncronized
   to. Defaults to the ID specified in the repository section header
 
 * `arch` - syncronize only the specified machine architecture (as per 
   `uname -m`)

 * `newonly` (0 or 1) - syncronize only the most recent version of upstream
   packages

 * `sources` (0 or 1) - download source RPMs in addition to compiled packages

 * `deleteremoved` (0 or 1) - delete local packages that have been removed from
   the upstream repository

 * `gpgcheck` (0 or 1) - delete local packages that fail GPG signature check.

 * `checksum` (sha256 or sha) - checksum type to use when creating a repository
   database (default: sha256)
