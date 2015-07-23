APP = y10k
APPVER = 0.1.0
ARCH = $(shell uname -i)
PACKAGE = $(APP)-$(APPVER).$(ARCH)
TARBALL = $(PACKAGE).tar.gz

GO = go
GFLAGS = -x
RM = rm -f
TAR = tar

all: $(APP)

$(APP): main.go health.go yumfile.go yumrepo.go yumrepo_mirror.go
	$(GO) build -o $(APP) $(GFLAGS)

tar: $(APP) README.md
	mkdir $(PACKAGE)
	cp -r $(APP) README.md $(PACKAGE)/
	$(TAR) -czf $(TARBALL) $(PACKAGE)/
	$(RM) -r $(PACKAGE)

clean:
	$(GO) clean
	$(RM) -f $(APP) $(TARBALL)
