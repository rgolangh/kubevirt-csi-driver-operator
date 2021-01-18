SHELL :=/bin/bash

all: build
.PHONY: all

IMAGE_REF?=quay.io/kubevirt/csi-driver-operator:latest
GO_TEST_PACKAGES :=./pkg/... ./cmd/...

# You can customize go tools depending on the directory layout.
# example:
#GO_BUILD_PACKAGES :=./pkg/...
# You can list all the golang related variables by:
#   $ make -n --print-data-base | grep ^GO

# Include the library makefile
include $(addprefix ./vendor/github.com/openshift/build-machinery-go/make/, \
	golang.mk \
	targets/openshift/deps-gomod.mk \
)

.PHONY: image
image:
	docker build . -f Dockerfile -t $(IMAGE_REF)

.PHONY: push
push: image
	docker push $(IMAGE_REF)

# make target aliases
fmt: verify-gofmt
		
vet: verify-govet

.PHONY: vendor
vendor: verify-deps
