SHELL := /bin/bash

TRIVY_VERSION=0.45.0

ARCH := $(shell uname | tr '[:upper:]' '[:lower:]')

TRIVY = $(realpath ./local-dev/trivy)

CI_BUILD_TAG ?= insights
CORE_REPO=https://github.com/uselagoon/lagoon.git
CORE_TREEISH=main
LAGOON_CORE_IMAGE_REPO=testlagoon
LAGOON_CORE_IMAGE_TAG=main

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

.PHONY: development-api
development-api:
	export LAGOON_CORE=$$(mktemp -d ./lagoon-core.XXX) \
	&& export GRAPHQL_API=http://localhost:3000/graphql \
	&& export KEYCLOAK_API=http://localhost:8088/auth \
	&& git clone $(CORE_REPO) "$$LAGOON_CORE" \
	&& cd "$$LAGOON_CORE" \
	&& git checkout $(CORE_TREEISH) \
	&& IMAGE_REPO=$(LAGOON_CORE_IMAGE_REPO) IMAGE_REPO_TAG=$(LAGOON_CORE_IMAGE_TAG) COMPOSE_STACK_NAME=core-$(CI_BUILD_TAG) docker compose -p core-$(CI_BUILD_TAG) pull \
	&& IMAGE_REPO=$(LAGOON_CORE_IMAGE_REPO) IMAGE_REPO_TAG=$(LAGOON_CORE_IMAGE_TAG) COMPOSE_STACK_NAME=core-$(CI_BUILD_TAG) $(MAKE) compose-api-logs-development

.PHONY: development-api-down
development-api-down:
	docker compose -p core-$(CI_BUILD_TAG) --compatibility down -v --remove-orphans

# clean up any old charts to prevent bloating of old charts from running k3d stacks regularly
.PHONY: clean-core
clean-core: development-api-down
	@for core in $$(ls -1 | grep -o "lagoon-core.*") ; do \
		echo removing core directory $$core ; \
		rm -rf $$core ; \
	done

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

PATH := $(PATH):$(PWD)/local-dev

.PHONY: test
test: fmt vet local-dev/trivy development-api
	go test -v ./...

.PHONY: docker-build
docker-build:
	DOCKER_BUILDKIT=1 docker build --rm -f Dockerfile -t uselagoon/insights-handler:local .

