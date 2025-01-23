SHELL := /bin/bash

TRIVY_VERSION=0.45.0

ARCH := $(shell uname | tr '[:upper:]' '[:lower:]')

TRIVY = $(realpath ./local-dev/trivy)

.PHONY: local-dev/trivy
local-dev/trivy:
ifeq ($(TRIVY_VERSION), $(shell trivy version 2>/dev/null | sed -nE 's/^Version: ([0-9.]+).*/\1/p'))
	$(info linking local trivy version $(TRIVY_VERSION))
	ln -sf $(shell command -v trivy) ./local-dev/trivy
else
ifneq ($(TRIVY_VERSION), $(shell ./local-dev/trivy version 2>/dev/null | sed -nE 's/^Version: ([0-9.]+).*/\1/p'))
	$(info downloading trivy version $(TRIVY_VERSION) for $(ARCH))
	mkdir -p local-dev
	rm local-dev/trivy || true
	TMPDIR=$$(mktemp -d) \
		&& curl -sSL https://github.com/aquasecurity/trivy/releases/download/v$(TRIVY_VERSION)/trivy_$(TRIVY_VERSION)_$(ARCH)-64bit.tar.gz -o $$TMPDIR/trivy.tar.gz \
		&& (cd $$TMPDIR && tar -zxf trivy.tar.gz) && cp $$TMPDIR/trivy ./local-dev/trivy && rm -rf $$TMPDIR
	chmod a+x local-dev/trivy
endif
endif

.PHONY: runlocal
runlocal:
	go run main.go --problems-from-sbom=true \
		--rabbitmq-username=guest  \
		--rabbitmq-password=guest \
		--lagoon-api-host=http://localhost:8888/graphql \
		--jwt-token-signing-key=secret \
		--access-key-id=minio \
		--secret-access-key=minio123 \
		--disable-s3-upload=true \
		--debug=true

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: fmt vet local-dev/trivy
	PATH=$$PATH:$(PWD)/local-dev
	go test -v ./...