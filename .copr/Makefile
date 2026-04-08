# .copr/Makefile

TOP_DIR := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))/..

ARCH := $(shell uname -m)
ifeq ($(ARCH),x86_64)
	GO_ARCH := amd64
else ifeq ($(ARCH),aarch64)
	GO_ARCH := arm64
else
	GO_ARCH := amd64
endif

srpm:
	if [ -n "$(outdir)" ]; then \
		dnf install -y git go-rpm-macros rpmautospec python3-rpmautospec+{click,pygit2}; \
	fi

	cd $(TOP_DIR) && git fetch --tags --unshallow 2>/dev/null || cd $(TOP_DIR) && git fetch --tags

	$(eval VERSION := $(shell cd $(TOP_DIR) && git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//'))	
	$(eval VERSION := $(if $(VERSION),$(VERSION),6))
	
	@echo "Project Root: $(TOP_DIR)"
	@echo "Building Version: $(VERSION)"

	$(eval LATEST_GO := $(shell curl -s https://go.dev/VERSION?m=text | head -n1))

	@echo "Installing $(LATEST_GO)..."
	curl -sL https://go.dev/dl/$(LATEST_GO).linux-${GO_ARCH}.tar.gz -o /tmp/go.tar.gz
	mkdir -p /tmp/go-home
	tar -C /tmp/go-home -xzf /tmp/go.tar.gz

	@echo "Syncing Spec to Version: $(VERSION)"
	sed "s/^\(Version:\)\s*.*/\1 $(VERSION)/" $(TOP_DIR)/systemd-lock-handler.spec > $(TOP_DIR)/systemd-lock-handler.temp.spec

	rpmautospec process-distgit \
		$(TOP_DIR)/systemd-lock-handler.temp.spec \
		$(TOP_DIR)/systemd-lock-handler.spec.processed
	
	rm $(TOP_DIR)/systemd-lock-handler.temp.spec

	PATH=/tmp/go-home/go/bin:$(PATH) $(MAKE) -C $(TOP_DIR) stage VERSION=$(VERSION)
	
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
		$(TOP_DIR)/systemd-lock-handler.spec.processed