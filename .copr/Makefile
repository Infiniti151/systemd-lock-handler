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
	# 1. Install dependencies
	if [ -n "$(outdir)" ]; then \
		dnf install -y git go-rpm-macros rpmautospec python3-rpmautospec+{click,pygit2}; \
	fi; \
	\
	# 2. Environment Setup
	cd $(TOP_DIR) && (git fetch --tags --unshallow 2>/dev/null || git fetch --tags); \
	V_TAG=$$(cd $(TOP_DIR) && git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//'); \
	VERSION=$${V_TAG:-6}; \
	LATEST_GO=$$(curl -s https://go.dev/VERSION?m=text | head -n1); \
	\
	echo "Project Root: $(TOP_DIR)"; \
	echo "Building Version: $$VERSION"; \
	echo "Detected Arch: $(ARCH) -> Using Go Arch: $(GO_ARCH)"; \
	\
	# 3. Spec Manipulation (Local Paths to fix rpmautospec parsing)
	cd $(TOP_DIR) && sed -r "s/^(Version:[[:space:]]*).*/\1$$VERSION/" systemd-lock-handler.spec > systemd-lock-handler.temp.spec; \
	\
	# 4. Go Installation
	echo "Installing $$LATEST_GO..."; \
	curl -sL https://go.dev/dl/$$LATEST_GO.linux-$(GO_ARCH).tar.gz -o /tmp/go.tar.gz; \
	mkdir -p /tmp/go-home; \
	tar -C /tmp/go-home -xzf /tmp/go.tar.gz; \
	\
	# 5. Process Spec with rpmautospec
	cd $(TOP_DIR) && rpmautospec process-distgit systemd-lock-handler.temp.spec systemd-lock-handler.spec.processed; \
	\
	# 6. Build stage
	cd $(TOP_DIR) && PATH=/tmp/go-home/go/bin:$$PATH $(MAKE) stage VERSION=$$VERSION; \
	\
	# 7. Package and Build SRPM
	mkdir -p $(TOP_DIR)/out; \
	tar -czf $(TOP_DIR)/out/systemd-lock-handler.tar.gz \
		--exclude=out --exclude=.git --exclude=.github --exclude=*.png \
		--exclude=README.md --exclude=.gitignore -C $(TOP_DIR) .; \
	\
	cd $(TOP_DIR) && rpmbuild -bs \
		--define "_sourcedir $$(pwd)/out" \
		--define "_srcrpmdir $(if $(outdir),$(outdir),$$(pwd)/out)" \
		--define "_version $$VERSION" \
		systemd-lock-handler.spec.processed; \
	\
	# Clean up
	rm $(TOP_DIR)/systemd-lock-handler.temp.spec