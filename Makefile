NAME = systemd-lock-handler
VERSION = 1
LICENSE = ISC
MAINTAINER = Infiniti151
URL = https://github.com/Infiniti151/systemd-lock-handler
DESCRIPTION = "A systemd user service to handle lock/unlock events and trigger custom actions."
GPG_IDENTITY = 43163551+Infiniti151@users.noreply.github.com

# Update this to your desired location for binary installation (e.g., /usr/local/bin)
BIN_PATH ?= /usr/bin

DESTDIR      ?=
UNIT_PATH    ?= /usr/lib/systemd/user
DIST_DIR	 := dist
OUT_DIR 	 := out

REL_BIN_DIR  := $(patsubst /%,%,$(BIN_PATH))
REL_UNIT_DIR := $(patsubst /%,%,$(UNIT_PATH))

INSTALL_BIN_DIR  = $(DESTDIR)$(BIN_PATH)
INSTALL_UNIT_DIR = $(DESTDIR)$(UNIT_PATH)

ACTUAL_USER := $(shell logname 2>/dev/null || echo $$USER)
USER_ID     := $(shell id -u $(ACTUAL_USER))
USER_BUS    := DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/$(USER_ID)/bus

CYAN   = \033[1;36m
GREEN  = \033[1;32m
YELLOW = \033[1;33m
NC     = \033[0m # No Color

export ACTUAL_USER
export USER_ID
export USER_BUS

all: build

${NAME}:
	go build -ldflags '-s -w' -o ${OUT_DIR}/${NAME}

test:
	@echo -e "$(CYAN)--- Running tests... ---$(NC)"
	go test -v -race ./...
	@echo -e "$(GREEN)--- All tests passed. ---$(NC)"

build: test clean
	@echo -e "$(CYAN)---Building ${NAME}... ---$(NC)"
	@mkdir -p $(OUT_DIR)
	$(MAKE) ${NAME}
	@echo -e "$(GREEN)--- Build successful. ---$(NC)"

stage:
	@echo -e "$(CYAN)--- Processing systemd service template... ---$(NC)"
	@# Generate a temporary file so we don't overwrite our template source
	sed "s|{{BIN_PATH}}|$(BIN_PATH)/${NAME}|g" $(DIST_DIR)/${NAME}.service > $(DIST_DIR)/${NAME}.service.ready
	@echo -e "$(GREEN)--- Template generated successfully. ---$(NC)"

install: build stage
	@echo -e "$(CYAN)--- Installing ${NAME} to $(INSTALL_BIN_DIR) and systemd service files... ---$(NC)"
	sudo install -Dm755 $(OUT_DIR)/${NAME} $(INSTALL_BIN_DIR)/${NAME}
	sudo install -Dm644 $(DIST_DIR)/${NAME}.service.ready $(INSTALL_UNIT_DIR)/${NAME}.service
	sudo install -Dm644 $(DIST_DIR)/lock.target $(INSTALL_UNIT_DIR)/lock.target
	sudo install -Dm644 $(DIST_DIR)/unlock.target $(INSTALL_UNIT_DIR)/unlock.target
	sudo install -Dm644 $(DIST_DIR)/sleep.target $(INSTALL_UNIT_DIR)/sleep.target
	
	@echo -e "$(YELLOW)--- Reloading systemd user daemon and enabling service for $(ACTUAL_USER)... ---$(NC)"
	sudo -u $(ACTUAL_USER) $(USER_BUS) systemctl --user daemon-reload
	sudo -u $(ACTUAL_USER) $(USER_BUS) systemctl --user enable --now ${NAME}.service
	@echo -e "$(GREEN)--- Installation complete. ${NAME} is now running. ---$(NC)"

uninstall:
	@echo -e "$(YELLOW)--- Stopping and disabling ${NAME} services for $(ACTUAL_USER)... ---$(NC)"
	-sudo -u $(ACTUAL_USER) $(USER_BUS) systemctl --user stop ${NAME}.service lock.target unlock.target sleep.target
	-sudo -u $(ACTUAL_USER) $(USER_BUS) systemctl --user disable ${NAME}.service
	
	@echo -e "$(CYAN)--- Removing system files... ---$(NC)"
	sudo rm -f $(INSTALL_UNIT_DIR)/${NAME}.service
	sudo rm -f $(INSTALL_UNIT_DIR)/lock.target
	sudo rm -f $(INSTALL_UNIT_DIR)/unlock.target
	sudo rm -f $(INSTALL_UNIT_DIR)/sleep.target
	sudo rm -f $(INSTALL_BIN_DIR)/${NAME}

	@echo -e "$(CYAN)--- Cleaning up systemd state... ---$(NC)"
	sudo -u $(ACTUAL_USER) $(USER_BUS) systemctl --user daemon-reload || true
	sudo -u $(ACTUAL_USER) $(USER_BUS) systemctl --user reset-failed || true
	@echo -e "$(GREEN)--- Uninstallation complete. ---$(NC)"

update:
	@echo -e "$(CYAN)--- Updating ${NAME}... ---$(NC)"
	$(MAKE) uninstall
	$(MAKE) install
	@echo -e "$(GREEN)--- Update complete. ---$(NC)"

update-binary: build
	@echo -e "$(CYAN)--- Updating ${NAME} binary only... ---$(NC)"
	sudo install -Dm755 $(OUT_DIR)/${NAME} $(INSTALL_BIN_DIR)/${NAME}
	
	@echo -e "$(YELLOW)--- Reloading systemd user daemon and restarting service for $(ACTUAL_USER)... ---$(NC)"
	sudo -u $(ACTUAL_USER) $(USER_BUS) systemctl --user daemon-reload
	sudo -u $(ACTUAL_USER) $(USER_BUS) systemctl --user restart ${NAME}.service
	sudo -u $(ACTUAL_USER) $(USER_BUS) systemctl --user reset-failed ${NAME}.service || true
	@echo -e "$(GREEN)--- Binary update complete. ---$(NC)"

# CI-optimized packaging
fpm-packages: build stage
	@echo -e "$(CYAN)--- Packaging version ${VERSION}... ---$(NC)"
	for type in rpm deb; do \
		fpm -s dir -t $$type \
			-n ${NAME} \
			-v ${VERSION} \
			--prefix / \
			--license ${LICENSE} \
			--maintainer ${MAINTAINER} \
			--url ${URL} \
			--description ${DESCRIPTION} \
			--package ${OUT_DIR}/${NAME}-v${VERSION}.$$type \
			${OUT_DIR}/${NAME}=$(REL_BIN_DIR)/${NAME} \
			${DIST_DIR}/${NAME}.service.ready=$(REL_UNIT_DIR)/${NAME}.service \
			${DIST_DIR}/lock.target=$(REL_UNIT_DIR)/lock.target \
			${DIST_DIR}/unlock.target=$(REL_UNIT_DIR)/unlock.target \
			${DIST_DIR}/sleep.target=$(REL_UNIT_DIR)/sleep.target; \
	done
	@echo -e "$(GREEN)--- Packaging complete. ---$(NC)"

sign:
	@echo -e "$(CYAN)--- Signing RPM package... ---$(NC)"
	rpmsign --addsign ${OUT_DIR}/${NAME}-v${VERSION}.rpm
	
	@echo -e "$(CYAN)--- Signing Deb package... ---$(NC)"
	#debsigs --sign=origin --default-key="$(GPG_IDENTITY)" ${OUT_DIR}/${NAME}-v${VERSION}.deb

checksums:
	@echo -e "$(CYAN)--- Generating SHA256 checksums... ---$(NC)"
	sha256sum $(OUT_DIR)/$(NAME)-v$(VERSION).* | \
	gpg --batch --yes --clearsign --digest-algo SHA256 --output $(OUT_DIR)/hashes.sha256.asc

publickey:
	@echo -e "$(CYAN)--- Exporting GPG public key... ---$(NC)"
	gpg --batch --armor --export ${GPG_IDENTITY} > out/public.key
	@echo -e "$(GREEN)--- Public key exported to ${OUT_DIR}/public.key ---$(NC)"

release: fpm-packages sign checksums publickey
	@echo -e "$(GREEN)--- Build, Packaging, Signing, and Checksums complete. ---$(NC)"

clean:
	@echo -e "$(CYAN)--- Cleaning up build artifacts... ---$(NC)"
	rm -rf $(OUT_DIR) $(DIST_DIR)/*.ready
	@echo -e "$(GREEN)--- Clean complete. ---$(NC)"

.PHONY: all test build stage install uninstall update update-binary fpm-packages sign checksums publickey release clean