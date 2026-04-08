# .copr/Makefile

TOP_DIR := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))/..

ARCH := $(shell uname -m)
GO_ARCH := $(if $(filter x86_64,$(ARCH)),amd64,$(if $(filter aarch64,$(ARCH)),arm64,amd64))

srpm:
	if [ -n "$(outdir)" ]; then \
        dnf install -y git go-rpm-macros rpmautospec; \
    fi; \
	\
	cd $(TOP_DIR) && git config --global --add safe.directory "$$(pwd)"; \
    (git fetch --tags --unshallow 2>/dev/null || true); \
	\
    V_TAG=$$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//'); \
    VERSION=$${V_TAG:-6}; \
	\
    LATEST_GO=$$(curl -s https://go.dev/VERSION?m=text | head -n1); \
    GO_URL="https://go.dev/dl/$${LATEST_GO}.linux-$(GO_ARCH).tar.gz"; \
	\
    echo "Project Root: $(TOP_DIR)"; \
    echo "Building Version: $$VERSION"; \
    echo "Detected Arch: $(ARCH) -> Go Arch: $(GO_ARCH)"; \
    echo "Target Go: $$LATEST_GO"; \
	\
	curl -sL "$$GO_URL" -o /tmp/go.tar.gz || { echo "Go download failed!"; exit 1; }; \
    mkdir -p /tmp/go-home; \
    tar -C /tmp/go-home -xzf /tmp/go.tar.gz; \
	\
    cd $(TOP_DIR) && PATH=/tmp/go-home/go/bin:$$PATH GOTOOLCHAIN=local \
		$(MAKE) stage VERSION=$$VERSION; \
	\
	mkdir -p $(TOP_DIR)/out; \
    tar -czf $(TOP_DIR)/out/systemd-lock-handler.tar.gz \
        --exclude=out --exclude=.git --exclude=.github --exclude=*.png \
        --exclude=README.md --exclude=.gitignore -C $(TOP_DIR) .; \
	\
	sed -r "s/^(Version:[[:space:]]*).*/\1$$VERSION/" $(TOP_DIR)/systemd-lock-handler.spec.template > $(TOP_DIR)/systemd-lock-handler.spec; \
	\
	echo "=== Running rpmbuild on .spec.processed with version $$VERSION ==="; \
	cd $(TOP_DIR) && rpmbuild -bs \
        --define "_sourcedir $$(pwd)/out" \
        --define "_srcrpmdir $(if $(outdir),$(outdir),$$(pwd)/out)" \
		--define "custom_go_path /tmp/go-home/go/bin" \
        $(TOP_DIR)/systemd-lock-handler.spec && \
	echo "=== SUCCESS: SRPM built with version $$VERSION ===" || \
	{ echo "ERROR: rpmbuild failed"; exit 1; }; \
	\
	if [ -n "$(outdir)" ]; then \
        ls -l "$(outdir)"/systemd-lock-handler-*.src.rpm 2>/dev/null || echo "No SRPM found"; \
    fi; \