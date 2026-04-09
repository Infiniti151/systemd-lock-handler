# .copr/Makefile

TOP_DIR := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))/..

srpm:
	if [ -n "$(outdir)" ]; then \
        dnf in -y git go-rpm-macros systemd-rpm-macros mock-rpmautospec \
		rpmautospec python3-rpmautospec+all python3-rpmautospec-core; \
    fi; \
	\
	git config --global --add safe.directory "*"; \
    echo "=== Fetching full history and tags ==="; \
    git fetch --tags --prune --unshallow 2>/dev/null || true; \
    git fetch --tags --prune --depth=100 origin "+refs/heads/*:refs/remotes/origin/*" 2>/dev/null || true; \
    git fetch --tags --prune origin tag 'v*' 2>/dev/null || true; \
	\
    echo "=== Current git describe: $$(git describe --tags --abbrev=0 2>/dev/null || echo 'no tag found') ==="; \
    echo "=== Number of commits available: $$(git rev-list --count HEAD 2>/dev/null || echo 'unknown') ==="; \
	\
    V_TAG=$$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//'); \
    VERSION=$${V_TAG:-6}; \
	\
    LATEST_GO=$$(curl -s https://go.dev/VERSION?m=text | head -n1); \
	\
    echo "=== Project Root: $(TOP_DIR) ==="; \
    echo "=== Building Version: $$VERSION ==="; \
    echo "=== Target Go: $$LATEST_GO ==="; \
	\
	mkdir -p $(TOP_DIR)/out; \
	curl -sL "https://go.dev/dl/$$LATEST_GO.linux-amd64.tar.gz" -o $(TOP_DIR)/out/go-amd64.tar.gz || { echo "Go amd64 download failed!"; exit 1; }; \
	curl -sL "https://go.dev/dl/$$LATEST_GO.linux-arm64.tar.gz" -o $(TOP_DIR)/out/go-arm64.tar.gz || { echo "Go arm64 download failed!"; exit 1; }; \
	\
	echo "=== Downloaded Go binaries ==="; \
    cd $(TOP_DIR) && $(MAKE) stage VERSION=$$VERSION; \
	\
    tar -czf $(TOP_DIR)/out/systemd-lock-handler.tar.gz \
        --exclude=out --exclude=.git --exclude=.github --exclude=.copr --exclude=*.png \
        --exclude=README.md --exclude=Makefile --exclude=systemd-lock-handler.spec.template \
		--exclude=dist/systemd-lock-handler.service --exclude=.gitignore -C $(TOP_DIR) .; \
	\
	sed -r "s/^(Version:[[:space:]]*).*/\1$$VERSION/" $(TOP_DIR)/systemd-lock-handler.spec.template > $(TOP_DIR)/systemd-lock-handler-copr.spec; \
	echo "=== Version $$VERSION written to spec file ==="; \
	echo "=== Content of copr.spec (first 30 lines) ==="; \
    head -n 30 $(TOP_DIR)/systemd-lock-handler-copr.spec; \
	echo "=== Evaluating macros ==="; \
	cd $(TOP_DIR) && echo "autorelease:" $$(rpm --eval '%autorelease'); \
    cd $(TOP_DIR) && echo "autochangelog:" $$(rpm --eval '%autochangelog'); \
    echo "=== Running rpmautospec process-distgit ==="; \
	cd $(TOP_DIR) && rpmautospec process-distgit systemd-lock-handler-copr.spec systemd-lock-handler.spec; \
	echo "=== Generated .spec file ==="; \
	\
	echo "=== Running rpmbuild with version $$VERSION ==="; \
	cd $(TOP_DIR) && rpmbuild -bs \
        --define "_sourcedir $$(pwd)/out" \
        --define "_srcrpmdir $(if $(outdir),$(outdir),$$(pwd)/out)" \
        $(TOP_DIR)/systemd-lock-handler.spec && \
	echo "=== SUCCESS: SRPM built with version $$VERSION ===" || \
	{ echo "ERROR: rpmbuild failed"; exit 1; }; \
	\
	if [ -n "$(outdir)" ]; then \
        ls -l "$(outdir)"/systemd-lock-handler-*.src.rpm 2>/dev/null || echo "No SRPM found"; \
    fi; \
	rm -f ./systemd-lock-handler-copr.spec