DOCKER:=docker

check-docker-env:
ifeq ($(DOCKER_REGISTRY),)
	$(error DOCKER_REGISTRY environment variable must be set)
endif
ifeq ($(DOCKER_TAG),)
	$(error DOCKER_TAG environment variable must be set)
endif

docker-build: check-docker-env
	$(DOCKER) build . -t $(DOCKER_REGISTRY):$(DOCKER_TAG)

docker-push: check-docker-env
	$(DOCKER) push $(DOCKER_REGISTRY):$(DOCKER_TAG)
