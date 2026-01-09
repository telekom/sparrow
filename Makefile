# SPDX-FileCopyrightText: 2024 Deutsche Telekom IT GmbH
#
# SPDX-License-Identifier: Apache-2.0

PORT := 5000
BINARY := sparrow

#############
### Build ###
#############

.PHONY: build-linux-amd64
build-linux-amd64:
	@echo "Building $(BINARY) for linux/amd64..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BINARY) .

.PHONY: build-linux-arm64
build-linux-arm64:
	@echo "Building $(BINARY) for linux/arm64..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o $(BINARY) .

#############
### E2E  ###
#############

.PHONY: e2e-cluster
e2e-cluster:
	@echo "Creating local registry..."
	@k3d registry create registry.localhost --port $(PORT) || echo "Registry may already exist"
	@echo "Creating k3d cluster using local registry..."
	@uname | grep -q 'Darwin' && export K3D_FIX_DNS=0; k3d cluster create sparrow-tests --registry-use k3d-registry.localhost:$(PORT)
	@echo "Create test namespace..."
	@kubectl create namespace sparrow || echo "Namespace already exists"

.PHONY: dev-image-linux-amd64
dev-image-linux-amd64: build-linux-amd64
	@echo "Building dev image (linux/amd64)..."
	@DOCKER_DEFAULT_PLATFORM=linux/amd64 docker build -t k3d-registry.localhost:$(PORT)/sparrow:dev .
	@echo "Pushing dev image..."
	@docker push k3d-registry.localhost:$(PORT)/sparrow:dev

.PHONY: dev-image-linux-arm64
dev-image-linux-arm64: build-linux-arm64
	@echo "Building dev image (linux/arm64)..."
	@DOCKER_DEFAULT_PLATFORM=linux/arm64 docker build -t k3d-registry.localhost:$(PORT)/sparrow:dev .
	@echo "Pushing dev image..."
	@docker push k3d-registry.localhost:$(PORT)/sparrow:dev

.PHONY: dev-image
dev-image: dev-image-linux-amd64

.PHONY: e2e-images
e2e-images: dev-image
	@echo "Importing dev image to cluster..."
	@k3d image import k3d-registry.localhost:$(PORT)/sparrow:dev --cluster sparrow-tests

.PHONY: e2e-deploy
e2e-deploy:
	@echo "Deploying sparrow via Helm..."
	@helm upgrade -i sparrow chart -n sparrow --create-namespace \
		--set image.repository=k3d-registry.localhost:$(PORT)/sparrow \
		--set image.tag=dev \
		--wait --debug --atomic

.PHONY: e2e-prep
e2e-prep: e2e-cluster e2e-images e2e-deploy

.PHONY: e2e-cleanup
e2e-cleanup:
	@echo "Cleaning up test env..."
	@k3d registry delete registry.localhost || echo "Deleting k3d registry failed. Continuing..."
	@helm uninstall sparrow -n sparrow || echo "Uninstalling sparrow helm release failed. Continuing..."
	@k3d cluster delete sparrow-tests || echo "Deleting k3d cluster failed. Continuing..."
	@echo "Done."
