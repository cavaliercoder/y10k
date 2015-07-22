# -*- mode: ruby -*-
# vi: set ft=ruby :

$script = <<end
yum install -y epel-release yum-utils createrepo
yum install -y golang
end

Vagrant.configure(2) do |config|
  config.vm.box = "chef/centos-7.0"
end
