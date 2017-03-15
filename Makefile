APP = y10k
APPVER = 0.3.0
ARCH = $(shell uname -m)
PACKAGE = $(APP)-$(APPVER).$(ARCH)
TARBALL = $(PACKAGE).tar.gz

GO = go
RM = rm -f
TAR = tar

all: $(APP)

$(APP): *.go yum/*.go
	$(GO) build -x -o $(APP)

get-deps:
	$(GO) get github.com/cavaliercoder/go-rpm
	$(GO) get github.com/cavaliercoder/grab
	$(GO) get github.com/codegangsta/cli
	$(GO) get github.com/dsnet/compress
	$(GO) get github.com/mattn/go-sqlite3
	$(GO) get code.cloudfoundry.org/bytefmt
	$(GO) get github.com/pkg/errors
	$(GO) get xi2.org/x/xz

tar: $(APP) README.md
	mkdir $(PACKAGE)
	cp -r $(APP) README.md $(PACKAGE)/
	$(TAR) -czf $(TARBALL) $(PACKAGE)/
	$(RM) -r $(PACKAGE)

clean:
	$(GO) clean
	$(RM) -f $(APP) $(TARBALL)

docker-image:
	docker build -t cavaliercoder/y10k .

docker-run:
	docker run -it --rm -v $(PWD):/usr/src/y10k cavaliercoder/y10k

.PHONY: all get-deps tar clean docker-image docker-run
