# Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
# All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

NAME = gopherpit
VERSION ?= $(shell cat version)
DOCKER_IMAGE ?= gopherpit/gopherpit
GO_PACKAGE_PATH = gopherpit.com/gopherpit
GOLANG_DOCKER_IMAGE ?= golang:1.8.3
GO ?= go
GOLINT ?= golint

ROOT_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

LDFLAGS = -w -X $(GO_PACKAGE_PATH)/server/config.Version="$(VERSION)"
ifdef CONFIG_DIR
LDFLAGS += -X $(GO_PACKAGE_PATH)/server/config.Dir="$(CONFIG_DIR)"
endif
LDFLAGS += -X $(GO_PACKAGE_PATH)/server/config.BuildInfo="$(shell git describe --long --dirty --always || true)"

BULMA_VERSION ?= 0.4.2
FONTAWESOME_VERSION ?= 4.7.0
VUE_VERSION ?= 2.2.4
AXIOS_VERSION ?= 0.15.3
LODASH_VERSION ?= 4.17.4
LODASH_INCLUDE ?= debounce,throttle,includes

NODEJS ?= docker run -it --rm -v $$(pwd):/usr/src/app -w /usr/src/app node

.PHONY: all
all: dist/$(NAME) dist/version dist/assets dist/static dist/templates dist/docker

.PHONY: binary
binary: dist/$(NAME)

dist:
	mkdir $@

dist/$(NAME): dist FORCE
	$(GO) version
ifndef CGO_ENABLED
	export CGO_ENABLED=0
endif
	$(GO) build -ldflags "$(LDFLAGS)" -o $@ ./cmd/$(NAME)

dist/assets: FORCE
	rm -rf dist/assets
	cp -a assets dist/.

dist/static: FORCE
	rm -rf dist/static
	cp -a static dist/.

dist/templates: FORCE
	rm -rf dist/templates
	cp -a templates dist/.

dist/docker: FORCE
	rm -rf dist/docker
	cp -a docker dist/.

dist/version: dist FORCE
	echo $(VERSION) > $@

.PHONY: assets
assets:
	mkdir -p dist/frontend
	echo "" > assets/vendor.css
	echo "" > assets/vendor.js

	# Bulma
	cd dist/frontend && \
		$(NODEJS) npm install bulma@$(BULMA_VERSION)
	echo "$$(cat frontend/bulma.sass)\n$$(cat dist/frontend/node_modules/bulma/bulma.sass)" > dist/frontend/node_modules/bulma/bulma.sass
	cd dist/frontend/node_modules/bulma && \
		$(NODEJS) npm install && \
		$(NODEJS) npm run build
	cd dist/frontend/node_modules/bulma && \
		$(NODEJS) npm install clean-css-cli && \
		$(NODEJS) ./node_modules/clean-css-cli/bin/cleancss -o css/bulma.min.css css/bulma.css
	cat dist/frontend/node_modules/bulma/css/bulma.min.css >> assets/vendor.css

	# FontAwesome
	mkdir -p assets/fonts
	cd assets/fonts && \
	curl -sSL \
		 -O https://github.com/FortAwesome/Font-Awesome/raw/v$(FONTAWESOME_VERSION)/fonts/FontAwesome.otf \
		 -O https://github.com/FortAwesome/Font-Awesome/raw/v$(FONTAWESOME_VERSION)/fonts/fontawesome-webfont.eot \
		 -O https://github.com/FortAwesome/Font-Awesome/raw/v$(FONTAWESOME_VERSION)/fonts/fontawesome-webfont.svg\
		 -O https://github.com/FortAwesome/Font-Awesome/raw/v$(FONTAWESOME_VERSION)/fonts/fontawesome-webfont.ttf\
		 -O https://github.com/FortAwesome/Font-Awesome/raw/v$(FONTAWESOME_VERSION)/fonts/fontawesome-webfont.woff \
		 -O https://github.com/FortAwesome/Font-Awesome/raw/v$(FONTAWESOME_VERSION)/fonts/fontawesome-webfont.woff2
	curl -sSL https://raw.githubusercontent.com/FortAwesome/Font-Awesome/v$(FONTAWESOME_VERSION)/css/font-awesome.min.css| sed 's/\.\.\/fonts\//fonts\//g' >> assets/vendor.css

	# Vue.js
	curl -sSL https://unpkg.com/vue@$(VUE_VERSION)/dist/vue.min.js >> assets/vendor.js
	echo "\n" >> assets/vendor.js
	# axios
	curl -sSL https://unpkg.com/axios@$(AXIOS_VERSION)/dist/axios.min.js >> assets/vendor.js
	echo "\n" >> assets/vendor.js

	# Lodash
	cd dist/frontend && \
		$(NODEJS) npm install lodash-cli@$(LODASH_VERSION) && \
		$(NODEJS) node_modules/lodash-cli/bin/lodash include=$(LODASH_INCLUDE)
	cat dist/frontend/lodash.custom.min.js >> assets/vendor.js

	rm -rf dist/frontend

.PHONY: clean
clean:
	rm -rf \
		dist/$(NAME) \
		dist/version \
		dist/static \
		dist/assets \
		dist/templates \
		dist/frontend

.PHONY: test
test:
	$(GO) test -race -v ./api/...
	$(GO) test -race -v ./cmd/...
	$(GO) test -race -v ./pkg/...
	$(GO) test -race -v ./server/...
	$(GO) test -race -v ./services/...
	$(GO) test -race -v *.go

.PHONY: vet
vet:
	$(GO) vet ./api/...
	$(GO) vet ./cmd/...
	$(GO) vet ./pkg/...
	$(GO) vet ./server/...
	$(GO) vet ./services/...
	$(GO) vet *.go

.PHONY: lint
lint:
	$(GOLINT) ./api/...
	$(GOLINT) ./cmd/...
	$(GOLINT) ./pkg/...
	$(GOLINT) ./server/...
	$(GOLINT) ./services/...
	$(GOLINT) *.go

.PHONY: autoreload
autoreload:
	echo -n -e "\033]0;$(NAME) - $@\007"
	rm -rf dist/static dist/assets dist/templates
	ln -s ../static dist/static
	ln -s ../assets dist/assets
	ln -s ../templates dist/templates
	reflex --only-files -s -r '(\.html|\.md|dist/$(NAME))$$' -- ./dist/$(NAME) --debug

.PHONY: autobuild
autobuild:
	echo -n -e "\033]0;$(NAME) - $@\007"
	reflex -r '\.go$$' -- make binary

.PHONY: reflex
reflex:
	$(GO) get github.com/cespare/reflex

.PHONY: develop
develop: all autoreload

.PHONY: package
package: package-linux-amd64 package-linux-arm package-darwin-amd64 package-freebsd-amd64

.PHONY: package-linux-amd64
package-linux-amd64: dist/version dist/assets dist/static dist/templates dist/docker
	GOOS=linux GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS)" -o dist/$(NAME) ./cmd/$(NAME)
	tar -C dist -czf dist/$(NAME)-$(VERSION)-linux-amd64.tar.gz $(NAME) version assets static templates
	cd dist && zip $(NAME)-$(VERSION)-linux-amd64.zip $(NAME) version assets/* static/* templates/*

.PHONY: package-linux-arm
package-linux-arm: dist/version dist/assets dist/static dist/templates dist/docker
	GOOS=linux GOARCH=arm $(GO) build -ldflags "$(LDFLAGS)" -o dist/$(NAME) ./cmd/$(NAME)
	tar -C dist -czf dist/$(NAME)-$(VERSION)-linux-arm.tar.gz $(NAME) version assets static templates
	cd dist && zip $(NAME)-$(VERSION)-linux-arm.zip $(NAME) version assets/* static/* templates/*

.PHONY: package-darwin-amd64
package-darwin-amd64: dist/version dist/assets dist/static dist/templates dist/docker
	GOOS=darwin GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS)" -o dist/$(NAME) ./cmd/$(NAME)
	tar -C dist -czf dist/$(NAME)-$(VERSION)-darwin-amd64.tar.gz $(NAME) version assets static templates
	cd dist && zip $(NAME)-$(VERSION)-darwin-amd64.zip $(NAME) version assets/* static/* templates/*

.PHONY: package-freebsd-amd64
package-freebsd-amd64: dist/version dist/assets dist/static dist/templates dist/docker
	GOOS=freebsd GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS)" -o dist/$(NAME) ./cmd/$(NAME)
	tar -C dist -czf dist/$(NAME)-$(VERSION)-freebsd-amd64.tar.gz $(NAME) version assets static templates
	cd dist && zip $(NAME)-$(VERSION)-freebsd-amd64.zip $(NAME) version assets/* static/* templates/*

.PHONY: package-in-docker
package-in-docker:
	docker run --rm \
	    -v "$(ROOT_DIR)":/go/src/$(GO_PACKAGE_PATH) \
	    -w /go/src/$(GO_PACKAGE_PATH) \
	    "$(GOLANG_DOCKER_IMAGE)" \
	    /bin/sh -c 'apt-get update && apt-get install -y zip && make package'

.PHONY: all-in-docker
all-in-docker:
	docker run --rm \
	    -v "$(ROOT_DIR)":/go/src/$(GO_PACKAGE_PATH) \
	    -w /go/src/$(GO_PACKAGE_PATH) \
	    "$(GOLANG_DOCKER_IMAGE)" \
	    make all

.PHONY: docker-image
docker-image:
	docker build -f dist/docker/Dockerfile -t $(DOCKER_IMAGE):$(VERSION) dist
	docker push $(DOCKER_IMAGE):$(VERSION)

.PHONY: docker
docker: clean all-in-docker docker-image

FORCE:
