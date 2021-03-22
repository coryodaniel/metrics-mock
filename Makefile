VERSION=$(shell cat VERSION)
BIN_NAME=metricsbin
REPO_NAME=quay.io/coryodaniel/$(BIN_NAME)

##@ Build
$(BIN_NAME): ## Compile binary
	go build

.PHONY: clean
clean: ## Cleanup
	rm -f $(BIN_NAME)	

##@ Docker
.PHONY: docker.build
docker.build: ## Build docker image
	docker build -t $(REPO_NAME):$(VERSION) .

.PHONY: docker.run
docker.run: ## Run docker container
docker.run: docker.stop
	docker run --name $(BIN_NAME) \
		-p 8080:8080 -it $(REPO_NAME):$(VERSION)

.PHONY: docker.debug
docker.debug: ## Run a debug shell
docker.debug: docker.stop
	docker build --build-arg RUN_IMG=gcr.io/distroless/base:debug \
		-t $(REPO_NAME):debug .
	docker run --name $(BIN_NAME) --entrypoint=sh \
		-p 8080:8080 -it $(REPO_NAME):debug

.PHONY: docker.stop
docker.stop: ## Stop and rm container
	-docker stop $(BIN_NAME)
	-docker rm $(BIN_NAME)

.PHONY: docker.push
docker.push: ## Push image to quay
	docker push $(REPO_NAME):$(VERSION)

##@ Utility
.PHONY: help
help:  ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m\033[0m\n"} /^[$$()% a-zA-Z_.-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)