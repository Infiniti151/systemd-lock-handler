DESTDIR?=/
PREFIX=/usr
NAME=systemd-lock-handler
VERSION=1
LICENSE=ISC
MAINTAINER=Infiniti151
URL=https://github.com/Infiniti151/systemd-lock-handler
DESCRIPTION="A systemd user service to handle lock/unlock events and trigger custom actions."
GPG_IDENTITY=Infiniti151 <43163551+Infiniti151@users.noreply.github.com>

all: build

${NAME}:
	go build -ldflags '-s -w' -o ${NAME}

build: ${NAME}

test: build
	go test -v -race ./...

install: test
	@install -Dm755 ${NAME} ${DESTDIR}${PREFIX}/lib/${NAME}
	@install -Dm644 ${NAME}.service ${DESTDIR}${PREFIX}/lib/systemd/user/${NAME}.service
	@install -Dm644 lock.target ${DESTDIR}${PREFIX}/lib/systemd/user/lock.target
	@install -Dm644 unlock.target ${DESTDIR}${PREFIX}/lib/systemd/user/unlock.target
	# @install -Dm644 sleep.target ${DESTDIR}${PREFIX}/lib/systemd/user/sleep.target

# CI-optimized packaging
fpm-packages: build
	@echo "Packaging version ${VERSION}..."
	for type in rpm deb; do \
		fpm -s dir -t $$type \
		    -n ${NAME} \
			-v ${VERSION} \
			--prefix ${PREFIX} \
			--license ${LICENSE} \
			--maintainer ${MAINTAINER} \
			--url ${URL} \
			--description ${DESCRIPTION} \
			--package ${NAME}-v${VERSION}.$$type \
			${NAME}=lib/${NAME} \
			${NAME}.service=lib/systemd/user/${NAME}.service \
			lock.target=lib/systemd/user/lock.target \
			unlock.target=lib/systemd/user/unlock.target; \
			# sleep.target=lib/systemd/user/sleep.target; \
	done

sign:
	@echo "Signing RPM package..."
	rpmsign --addsign ${NAME}-v${VERSION}.rpm
	
	@echo "Signing Debian package..."
	debsigs --sign=origin --default-key="$(GPG_IDENTITY)" ${NAME}-v${VERSION}.deb

checksums:
	sha256sum ${NAME}-v${VERSION}.* > hashes.sha256
	gpg --batch --yes --clearsign --digest-algo SHA256 hashes.sha256
	rm hashes.sha256

release: fpm-packages sign checksums
	@echo "Build, Packaging, Signing, and Checksums complete."

clean:
	rm -f ${NAME} *.rpm *.deb *.asc

.PHONY: all build test install fpm-packages sign checksums release clean