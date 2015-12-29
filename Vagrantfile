# -*- mode: ruby -*-
# vi: set ft=ruby :

$script = <<end
# install build tools and runtime prerequisities
yum install -y epel-release
yum install -y \
	createrepo \
	git \
	golang \
	make \
	mercurial \
	yum-utils

# configure GOPATH for user vagrant
mkdir /home/vagrant/go
cat >> /home/vagrant/.bashrc <<EOF
export GOPATH=\\$HOME/go
export PATH=\\$PATH:/vagrant:\\$HOME/go/bin

EOF

# symlink go-rpm if checked out locally
GORPM_PATH=/home/vagrant/go/src/github.com/cavaliercoder/go-rpm
mkdir -p $GORPM_PATH
if [[ -d /vagrant/go-rpm ]]; then
	# delete existing clone
	[[ ! -h $GORPM_PATH ]] && rm -rf $GORPM_PATH

	ln -s /vagrant/go-rpm $GORPM_PATH
fi

# fix perms
chown -R vagrant.vagrant /home/vagrant

end

Vagrant.configure(2) do |config|
  config.vm.box = "chef/centos-7.0"
  config.vm.provision "shell", inline: $script
end
