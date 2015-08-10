APP = y10k
APPVER = 0.1.0
ARCH = $(shell uname -m)
PACKAGE = $(APP)-$(APPVER).$(ARCH)
TARBALL = $(PACKAGE).tar.gz

GO = go
GFLAGS = -x
RM = rm -f
TAR = tar


all: $(APP)

$(APP): main.go health.go yumfile.go yumrepo.go yumrepo_mirror.go io.go
	$(GO) build -x -o $(APP) $(GFLAGS)

get-deps:
	$(GO) get -u github.com/codegangsta/cli

tar: $(APP) README.md
	mkdir $(PACKAGE)
	cp -r $(APP) README.md $(PACKAGE)/
	$(TAR) -czf $(TARBALL) $(PACKAGE)/
	$(RM) -r $(PACKAGE)

clean:
	$(GO) clean
	$(RM) -f $(APP) $(TARBALL)

.PHONY: all get-deps tar clean