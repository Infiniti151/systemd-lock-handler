# .copr/Makefile

VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//')
VERSION := $(if $(VERSION),$(VERSION),6)

TOP_DIR := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))/..

srpm:
	if [ -n "$(outdir)" ]; then
		dnf install -y git curl go-rpm-macros;
	fi

	$(eval LATEST_GO := $(shell curl -s https://go.dev/VERSION?m=text | head -n1))
	
	@echo "Installing $(LATEST_GO)..."
	curl -sL https://go.dev/dl/$(LATEST_GO).linux-amd64.tar.gz -o /tmp/go.tar.gz
	mkdir -p /tmp/go-home
	tar -C /tmp/go-home -xzf /tmp/go.tar.gz
	
	$(eval export PATH := /tmp/go-home/go/bin:$(PATH))
	
	go version
	# -----------------------------

	$(MAKE) -C $(TOP_DIR) stage VERSION=$(VERSION)
	
	mkdir -p $(TOP_DIR)/out
	tar -czf $(TOP_DIR)/out/systemd-lock-handler.tar.gz \
		--exclude=out \
		--exclude=.git \
		--exclude=.github \
		--exclude=*.png \
		--exclude=README.md \
		--exclude=.gitignore \
		-C $(TOP_DIR) .
	
	rpmbuild -bs \
		--define "_sourcedir $(TOP_DIR)/out" \
		--define "_srcrpmdir $(if $(outdir),$(outdir),$(TOP_DIR)/out)" \
		--define "_version $(VERSION)" \
		$(TOP_DIR)/systemd-lock-handler.spec