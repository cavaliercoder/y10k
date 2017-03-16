APP = y10k
APPVER = 0.3.0
ARCH = $(shell uname -m)
PACKAGE = $(APP)-$(APPVER).$(ARCH)
TARBALL = $(PACKAGE).tar.gz

GO = go
GOGET = $(GO) get -v -u
RM = rm -f
TAR = tar

all: $(APP)

$(APP): *.go yum/*.go yum/compress/*.go yum/crypto/*.go
	$(GO) build -x -o $(APP)

get-deps:
	$(GOGET) github.com/cavaliercoder/go-rpm
	$(GOGET) github.com/cavaliercoder/grab
	$(GOGET) github.com/codegangsta/cli
	$(GOGET) github.com/dsnet/compress
	$(GOGET) github.com/mattn/go-sqlite3
	$(GOGET) code.cloudfoundry.org/bytefmt
	$(GOGET) github.com/pkg/errors
	$(GOGET) xi2.org/x/xz

tar: $(APP) README.md
	mkdir $(PACKAGE)
	cp -r $(APP) README.md $(PACKAGE)/
	$(TAR) -czf $(TARBALL) $(PACKAGE)/
	$(RM) -r $(PACKAGE)

clean:
	$(GO) clean
	$(RM) -f $(APP) $(TARBALL) debug

docker-image:
	docker build -t cavaliercoder/y10k .

docker-run:
	docker run -it --rm -v $(PWD):/usr/src/y10k cavaliercoder/y10k

.PHONY: all get-deps tar clean docker-image docker-run
