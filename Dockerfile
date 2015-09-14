FROM centos:7

# install OS packages
RUN yum install -y epel-release && \
	yum clean all && yum makecache && \
	yum install -y \
	createrepo \
	git \
	golang \
	make \
	mercurial \
	yum-utils

# setup GOPATH and source directory
RUN mkdir -p /go/{bin,pkg,src} /usr/src/y10k
ENV GOPATH=/go PATH=$PATH:/go/bin

# install package deps
ADD Makefile /tmp/Makefile
RUN cd /tmp && make get-deps

# open shell in source dir
CMD cd /usr/src/y10k; /bin/bash
