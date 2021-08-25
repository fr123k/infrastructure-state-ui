.PHONY: build shell

VERSION=1.1
IMAGE="app-terraform-plan:$(VERSION)"


export DOCKER_COMMAND_LOCAL=docker run -it \
		-e PORT=8080 \
		-p 8080:8080

build: ## Package the infra-hook go application into a go binary docker image.
	docker build -t $(IMAGE) -f Dockerfile .

pull: ## Pull the go application docker image to the docker registry.
	docker pull ${IMAGE}

release: ## Push the go application docker image to the docker registry.
	docker push ${IMAGE}

local: build ## Build and start the docker container to listen at port 8080
	$(DOCKER_COMMAND_LOCAL) $(IMAGE)

# Absolutely awesome: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
help: ## Print this help
	@grep -E '^[a-zA-Z._-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
