VERSION=$(shell git describe)
BUILDDIR = build
TARGETS = linux-amd64 linux-arm darwin-amd64
APP = vv
BINARIES = $(patsubst %, $(BUILDDIR)/%/$(APP), $(TARGETS))
ARCHIVES = $(patsubst %, $(BUILDDIR)/$(APP)-%.tar.gz, $(TARGETS))
CHECKSUM = sha256

.PHONY: build $(BINARIES) $(ARCHIVES)

build:
	go build -ldflags "-X main.version=$(VERSION)"

all: $(BINARIES)
archives: $(ARCHIVES) $(BUILDDIR)/$(CHECKSUM)

$(BINARIES):
	mkdir -p build/$(word 2,$(subst /, ,$@))
	GOOS=$(subst -, GOARCH=,$(word 2,$(subst /, ,$@))) go build -ldflags "-X main.version=$(VERSION)" -o $@

$(ARCHIVES): $(BINARIES)
	tar -czf $@ -C $(subst $(APP)-,,$(word 1,$(subst ., ,$@))) $(APP)

$(BUILDDIR)/$(CHECKSUM): $(BINARIES)
	rm -f $(BUILDDIR)/$(CHECKSUM)
	@LIST="$(BINARIES)";\
		for x in $$LIST; do\
		openssl $(CHECKSUM) $$x >> $(BUILDDIR)/$(CHECKSUM);\
		done

clean:
	rm -rf $(BUILDDIR)
