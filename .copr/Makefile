# .copr/Makefile

VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//')
VERSION := $(if $(VERSION),$(VERSION),6)

srpm:
	if [ -n "$(outdir)" ]; then dnf install -y rpm-build golang git; fi
	
	make -C .. stage VERSION=$(VERSION)
	mkdir -p ../out
	
	tar -czf ../out/systemd-lock-handler.tar.gz \
		--exclude=out \
		--exclude=.git \
		--exclude=.github \
		--exclude=*.png \
		--exclude=README.md \
		--exclude=.gitignore \
		-C .. .
	
	rpmbuild -bs \
		--define "_sourcedir $(shell pwd)/../out" \
		--define "_srcrpmdir $(if $(outdir),$(outdir),$(shell pwd)/../out)" \
		--define "_version $(VERSION)" \
		../systemd-lock-handler.spec