FROM centos:7

RUN yum update -y

RUN yum install -y epel-release 

RUN yum install -y \
	createrepo \
	git \
	golang \
	make \
	mercurial \
	yum-utils

RUN mkdir /root/gocode /usr/src/y10k

ENV GOPATH=/root/gocode

CMD cd /usr/src/y10k; /bin/bash
