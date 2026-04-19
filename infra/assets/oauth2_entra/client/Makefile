# Podman image configuration
ifeq ($(strip $(REGISTRY_USERNAME)),)
$(error REGISTRY_USERNAME is not set. Please set the REGISTRY_USERNAME environment variable.)
endif
IMAGE_REGISTRY := docker.io
IMAGE_NAME := oauth2-entra-client
IMAGE_TAG := latest
FULL_IMAGE_NAME := $(REGISTRY_USERNAME)/$(IMAGE_NAME):$(IMAGE_TAG)

.PHONY: build run clean login push

# Build the Podman image
build:
	podman build --platform linux/amd64 -t $(FULL_IMAGE_NAME) .

# Run the container (requires environment variables)
run:
	podman run --rm \
		-e API_SCOPE=$(API_SCOPE) \
		-e API_ENDPOINT=$(API_ENDPOINT) \
		$(FULL_IMAGE_NAME)

# Login to the registry
login:
	podman login $(IMAGE_REGISTRY)

# Push the image to the registry
push:
	podman push $(FULL_IMAGE_NAME)

# Clean up the image
clean:
	podman rmi $(FULL_IMAGE_NAME)
