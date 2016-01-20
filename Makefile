SHA := $(shell git rev-parse --short HEAD)
VERSION := $(shell cat VERSION)
ITTERATION := $(shell date +%s)
OGOPATH := $(shell echo $$GOPATH)
SRCPATH := "binaries"
export GO15VENDOREXPERIMENT=0


sync-all: oort-cli oort-value oort-group syndicate cfsdvp cfs formic

save:
	godep save ./...

build:
	godep go build -v ./...

install:
	godep go install -v ./...

test: build
	godep go test -v ./...

prerelease: install
	ghr -t $(GITHUB_TOKEN) -u $(GITHUB_USER) --replace --prerelease $(VERSION) $(BUILDPATH)

release: install
	ghr -t $(GITHUB_TOKEN) -u $(GITHUB_USER) --replace $(VERSION) $(BUILDPATH)

oort-cli:
	cp -av $(OGOPATH)/src/github.com/pandemicsyn/oort/oort-cli $(SRCPATH)
	cp -av $(OGOPATH)/src/github.com/pandemicsyn/oort/oort-bench $(SRCPATH)

oort-value:
	cp -av $(OGOPATH)/src/github.com/pandemicsyn/oort/oort-valued $(SRCPATH)

oort-group:
	cp -av $(OGOPATH)/src/github.com/pandemicsyn/oort/oort-groupd $(SRCPATH)

syndicate:
	cp -av $(OGOPATH)/src/github.com/pandemicsyn/syndicate/synd $(SRCPATH)
	cp -av $(OGOPATH)/src/github.com/pandemicsyn/syndicate/syndicate-client $(SRCPATH)

cfsdvp:
	cp -av $(OGOPATH)/src/github.com/creiht/formic/cfsdvp $(SRCPATH)

cfs:
	cp -av $(OGOPATH)/src/github.com/creiht/formic/cfs $(SRCPATH)

formic:
	cp -av $(OGOPATH)/src/github.com/creiht/formic/formicd $(SRCPATH)


