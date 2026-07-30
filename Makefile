NAME = systemd-lock-handler
VERSION ?= 1
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
export CGO_ENABLED=0

all: build

${NAME}:
	go build -ldflags '-s -w' -o ${OUT_DIR}/${NAME}

test:
	@printf "$(CYAN)=== Running tests... ===$(NC)\n"
	go test -v ./...
	@printf "$(GREEN)=== All tests passed. ===$(NC)\n"

build: test clean
	@printf "$(CYAN)===Building ${NAME}... ===$(NC)\n"
	@mkdir -p $(OUT_DIR)
	$(MAKE) ${NAME}
	@printf "$(GREEN)=== Build successful. ===$(NC)\n"

stage:
	@printf "$(CYAN)=== Processing systemd service template... ===$(NC)\n"
	@# Generate a temporary file so we don't overwrite our template source
	sed "s|{{BIN_PATH}}|$(BIN_PATH)/${NAME}|g" $(DIST_DIR)/${NAME}.service > $(DIST_DIR)/${NAME}.service.ready
	@printf "$(GREEN)=== Template generated successfully. ===$(NC)\n"

install: build stage
	@printf "$(CYAN)=== Installing ${NAME} to $(INSTALL_BIN_DIR) and systemd service files... ===$(NC)\n"
	sudo install -Dm755 $(OUT_DIR)/${NAME} $(INSTALL_BIN_DIR)/${NAME}
	sudo install -Dm644 $(DIST_DIR)/${NAME}.service.ready $(INSTALL_UNIT_DIR)/${NAME}.service
	sudo install -Dm644 $(DIST_DIR)/lock.target $(INSTALL_UNIT_DIR)/lock.target
	sudo install -Dm644 $(DIST_DIR)/unlock.target $(INSTALL_UNIT_DIR)/unlock.target
	sudo install -Dm644 $(DIST_DIR)/sleep.target $(INSTALL_UNIT_DIR)/sleep.target
	sudo install -Dm644 $(DIST_DIR)/wake.target $(INSTALL_UNIT_DIR)/wake.target

	@printf "$(YELLOW)=== Reloading systemd user daemon and enabling service for $(ACTUAL_USER)... ===$(NC)\n"
	sudo -u $(ACTUAL_USER) $(USER_BUS) systemctl --user daemon-reload
	sudo -u $(ACTUAL_USER) $(USER_BUS) systemctl --user enable --now ${NAME}.service
	@printf "$(GREEN)=== Installation complete. ${NAME} is now running. ===$(NC)\n"

uninstall:
	@printf "$(YELLOW)=== Stopping and disabling ${NAME} services for $(ACTUAL_USER)... ===$(NC)\n"
	-sudo -u $(ACTUAL_USER) $(USER_BUS) systemctl --user stop ${NAME}.service lock.target unlock.target sleep.target
	-sudo -u $(ACTUAL_USER) $(USER_BUS) systemctl --user disable ${NAME}.service

	@printf "$(CYAN)=== Removing system files... ===$(NC)\n"
	sudo rm -f $(INSTALL_UNIT_DIR)/${NAME}.service
	sudo rm -f $(INSTALL_UNIT_DIR)/lock.target
	sudo rm -f $(INSTALL_UNIT_DIR)/unlock.target
	sudo rm -f $(INSTALL_UNIT_DIR)/sleep.target
	sudo rm -f $(INSTALL_UNIT_DIR)/wake.target
	sudo rm -f $(INSTALL_BIN_DIR)/${NAME}

	@printf "$(CYAN)=== Cleaning up systemd state... ===$(NC)\n"
	sudo -u $(ACTUAL_USER) $(USER_BUS) systemctl --user daemon-reload || true
	sudo -u $(ACTUAL_USER) $(USER_BUS) systemctl --user reset-failed || true
	@printf "$(GREEN)=== Uninstallation complete. ===$(NC)\n"

update:
	@printf "$(CYAN)=== Updating ${NAME}... ===$(NC)\n"
	$(MAKE) uninstall
	$(MAKE) install
	@printf "$(GREEN)=== Update complete. ===$(NC)\n"

update-binary: build
	@printf "$(CYAN)=== Updating ${NAME} binary only... ===$(NC)\n"
	sudo install -Dm755 $(OUT_DIR)/${NAME} $(INSTALL_BIN_DIR)/${NAME}

	@printf "$(YELLOW)=== Reloading systemd user daemon and restarting service for $(ACTUAL_USER)... ===$(NC)\n"
	sudo -u $(ACTUAL_USER) $(USER_BUS) systemctl --user daemon-reload
	sudo -u $(ACTUAL_USER) $(USER_BUS) systemctl --user restart ${NAME}.service
	sudo -u $(ACTUAL_USER) $(USER_BUS) systemctl --user reset-failed ${NAME}.service || true
	@printf "$(GREEN)=== Binary update complete. ===$(NC)\n"

# CI-optimized packaging
fpm-packages: build stage
	@printf "$(CYAN)=== Packaging version ${VERSION}... ===$(NC)\n"
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
			${DIST_DIR}/sleep.target=$(REL_UNIT_DIR)/sleep.target \
			${DIST_DIR}/wake.target=$(REL_UNIT_DIR)/wake.target; \
	done
	@printf "$(GREEN)=== Packaging complete. ===$(NC)\n"

sign:
	@printf "$(CYAN)=== Signing RPM package... ===$(NC)\n"
	rpmsign --addsign ${OUT_DIR}/${NAME}-v${VERSION}.rpm

	@printf "$(CYAN)=== Signing Deb package... ===$(NC)\n"
	debsigs --sign=origin --default-key="$(GPG_IDENTITY)" ${OUT_DIR}/${NAME}-v${VERSION}.deb

checksums:
	@printf "$(CYAN)=== Generating SHA256 checksums... ===$(NC)\n"
	sha256sum $(OUT_DIR)/$(NAME)-v$(VERSION).* | \
	gpg --batch --yes --clearsign --digest-algo SHA256 --output $(OUT_DIR)/hashes.sha256.asc

publickey:
	@printf "$(CYAN)=== Exporting GPG public key... ===$(NC)\n"
	gpg --batch --armor --export ${GPG_IDENTITY} > out/public.key
	@printf "$(GREEN)=== Public key exported to ${OUT_DIR}/public.key ===$(NC)\n"

release: fpm-packages sign checksums publickey
	@printf "$(GREEN)=== Build, Packaging, Signing, and Checksums complete. ===$(NC)\n"

clean:
	@printf "$(CYAN)=== Cleaning up build artifacts... ===$(NC)\n"
	rm -rf $(OUT_DIR) $(DIST_DIR)/*.ready
	@printf "$(GREEN)=== Clean complete. ===$(NC)\n"

.PHONY: all test build stage install uninstall update update-binary fpm-packages sign checksums publickey release clean