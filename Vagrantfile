# -*- mode: ruby -*-
# vi: set ft=ruby :

$script = <<end
yum install -y epel-release
yum install -y \
	createrepo \
	git \
	golang \
	make \
	mercurial \
	yum-utils

end

Vagrant.configure(2) do |config|
  config.vm.box = "chef/centos-7.0"
end
