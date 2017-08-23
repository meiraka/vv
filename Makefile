VERSION=$(shell git describe)
TARGETS = linux-amd64 linux-arm darwin-amd64
BINARIES = $(patsubst %, vv-%, $(TARGETS))

.PHONY: build all $(BINARIES)

build:
	go build -ldflags "-X main.version=$(VERSION)"

all: $(BINARIES)

$(BINARIES):
	GOOS=`echo "$@" | cut -d - -f 2` GOARCH=`echo "$@" | cut -d - -f 3` go build -ldflags "-X main.version=$(VERSION)" -o $@

clean:
	rm -f $(BINARIES)
