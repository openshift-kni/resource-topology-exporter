COMMONENVVAR=GOOS=linux GOARCH=amd64
BUILDENVVAR=CGO_ENABLED=0
TOPOLOGYAPI_MANIFESTS=https://raw.githubusercontent.com/k8stopologyawareschedwg/noderesourcetopology-api/master/manifests

KUBECLI ?= kubectl
RUNTIME ?= podman
REPOOWNER ?= k8stopologyawarewg
IMAGENAME ?= resource-topology-exporter
IMAGETAG ?= latest
RTE_CONTAINER_IMAGE ?= quay.io/$(REPOOWNER)/$(IMAGENAME):$(IMAGETAG)

VERSION := $(shell git tag --sort=committerdate | head -n 1)
ifeq ($(VERSION),)
	VERSION = v0.0.0
endif

LDFLAGS = -ldflags "-s -w -X github.com/k8stopologyawareschedwg/resource-topology-exporter/pkg/version.version=$(VERSION)"

.PHONY: all
all: build

.PHONY: build
build: outdir
	go version
	$(COMMONENVVAR) $(BUILDENVVAR) go build $(LDFLAGS) -o _out/resource-topology-exporter cmd/resource-topology-exporter/main.go

.PHONY: gofmt
gofmt:
	@echo "Running gofmt"
	gofmt -s -w `find . -path ./vendor -prune -o -type f -name '*.go' -print`

.PHONY: govet
govet:
	@echo "Running go vet"
	go vet

outdir:
	mkdir -p _out || :

.PHONY: deps-update
deps-update:
	go mod tidy && go mod vendor

.PHONY: deps-clean
deps-clean:
	rm -rf vendor

.PHONY: binaries
binaries: outdir deps-update build

.PHONY: clean
clean:
	rm -rf _out

.PHONY: image
image: binaries
	@echo "building image"
	$(RUNTIME) build -f images/Dockerfile -t $(RTE_CONTAINER_IMAGE) .

.PHONY: push
push: image
	@echo "pushing image"
	$(RUNTIME) push $(RTE_CONTAINER_IMAGE)

.PHONY: test-unit
test-unit:
	[ -d ./pkg ] && go test ./pkg/... ./cmd/... || :

build-e2e: outdir
	# need to use makefile rules in a better way
	[ -x _out/rte-e2e.test ] || go test -v -c -o _out/rte-e2e.test ./test/e2e/

.PHONY: test-e2e
test-e2e: binaries build-e2e
	_out/rte-e2e.test

.PHONY: test-e2e-full
	go test -v ./test/e2e/

# a specific target for running e2e tests under the KNI's CI
# this assumes to be running on a vanilla OCP cluster
.PHONY: test-e2e-kni
test-e2e-kni: build-e2e
	hack/e2e-kni.sh

.PHONY: deploy
deploy:
	hack/deploy.sh

.PHONY: undeploy
undeploy:
	hack/undeploy.sh

.PHONY: gen-manifests
gen-manifests:
	@cat manifests/crd.yaml
	@hack/get-manifests.sh

.PHONY: label-custom-kubelet
label-custom-kubelet:
	hack/label-custom-kubelet.sh

.PHONY: kube-update
kube-update: label-custom-kubelet
	hack/kube-update.sh

.PHONY: wait-for-mcp
wait-for-mcp:
	hack/wait-for-mcp.sh

.PHONY: update-golden-files
update-golden-files:
	@go test ./cmd/... -update
