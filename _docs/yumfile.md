---
layout: page
title: Yumfile format
menu: Yumfile
permalink: /yumfile/
---

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

[centos-7-x86_64-updates]
name=CentOS 7 x86_64 Updates
mirrorlist=http://mirrorlist.centos.org/?release=7&arch=x86_64&repo=updates
localpath=centos/7/updates/x86_64
arch=x86_64

{% endhighlight %}