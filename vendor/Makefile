COMMONENVVAR=GOOS=linux GOARCH=amd64
BUILDENVVAR=CGO_ENABLED=0

RUNTIME ?= podman
REPOOWNER ?= swsehgal
IMAGENAME ?= resource-topology-exporter
IMAGETAG ?= latest

.PHONY: all
all: deps-update build

.PHONY: deps-update
deps-update:
	go mod tidy && \
	go mod vendor

.PHONY: build
build:
	$(COMMONENVVAR) $(BUILDENVVAR) go build -ldflags '-w' -o bin/resource-topology-exporter cmd/resource-topology-exporter/main.go

.PHONY: gofmt
gofmt:
	@echo "Running gofmt"
	gofmt -s -w `find . -path ./vendor -prune -o -type f -name '*.go' -print`

.PHONY: govet
govet:
	@echo "Running go vet"
	go vet

.PHONY: config
config:
	@echo "deploying configmap"
	kubectl create -f config/examples/sriovdp-configmap.yaml

.PHONY: image
image: build
	@echo "building image"
	$(RUNTIME) build -f images/Dockerfile -t quay.io/$(REPOOWNER)/$(IMAGENAME):$(IMAGETAG) .

.PHONY: crd
crd:
	@echo "deploying crd"
	kubectl create -f manifests/crd-apiextension-v1beta1.yaml

.PHONY: push
push: image
	@echo "pushing image"
	$(RUNTIME) push quay.io/$(REPOOWNER)/$(IMAGENAME):$(IMAGETAG)

.PHONY: deploy
deploy: push
	@echo "deploying Resource Topology Exporter"
	kubectl create -f manifests/resource-topology-exporter-ds.yaml

.PHONY: deploy-pod
deploy-pod:
	@echo "deploying Pods"
	kubectl create -f manifests/sample-devices/test-pod-deviceA.yaml
	kubectl create -f manifests/sample-devices/test-pod-deviceA-2.yaml
	kubectl create -f manifests/sample-devices/test-pod-deviceA-3.yaml

.PHONY: deploy-taerror
deploy-taerror:
	@echo "deploying Pod"
	kubectl create -f manifests/test-deployment-taerror.yaml

clean-binaries:
	rm -f bin/resource-topology-exporter

clean: clean-binaries
	kubectl delete -f manifests/resource-topology-exporter-ds.yaml
	kubectl delete -f manifests/sample-devices/test-pod-deviceA.yaml
	kubectl delete -f manifests/sample-devices/test-pod-deviceA-2.yaml
	kubectl delete -f manifests/sample-devices/test-pod-deviceA-3.yaml
