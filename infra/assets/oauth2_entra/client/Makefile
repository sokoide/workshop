# Docker image configuration
ifeq ($(strip $(DOCKERHUB_USERNAME)),)
$(error DOCKERHUB_USERNAME is not set. Please set the DOCKERHUB_USERNAME environment variable.)
endif
IMAGE_REGISTRY := docker.io
IMAGE_NAME := oauth2-entra-client
IMAGE_TAG := latest
FULL_IMAGE_NAME := $(DOCKERHUB_USERNAME)/$(IMAGE_NAME):$(IMAGE_TAG)

.PHONY: build run clean login push

# Build the Docker image
build:
	docker build --platform linux/amd64 -t $(FULL_IMAGE_NAME) .

# Run the container (requires environment variables)
run:
	docker run --rm \
		-e API_SCOPE=$(API_SCOPE) \
		-e API_ENDPOINT=$(API_ENDPOINT) \
		$(FULL_IMAGE_NAME)

# Login to the registry
login:
	docker login $(IMAGE_REGISTRY)

# Push the image to the registry
push:
	docker push $(FULL_IMAGE_NAME)

# Clean up the image
clean:
	docker rmi $(FULL_IMAGE_NAME)
