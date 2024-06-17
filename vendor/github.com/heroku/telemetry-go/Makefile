default: help

.PHONY: help
help: # Prints out help
	@IFS=$$'\n' ; \
	help_lines=(`fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$$//' | sed -e 's/##/:/'`); \
	printf "%-30s %s\n" "target" "help" ; \
	printf "%-30s %s\n" "------" "----" ; \
	for help_line in $${help_lines[@]}; do \
			IFS=$$':' ; \
			help_split=($$help_line) ; \
			help_command=`echo $${help_split[0]} | sed -e 's/^ *//' -e 's/ *$$//'` ; \
			help_info=`echo $${help_split[2]} | sed -e 's/^ *//' -e 's/ *$$//'` ; \
			printf '\033[36m'; \
			printf "%-30s %s" $$help_command ; \
			printf '\033[0m'; \
			printf "%s\n" $$help_info; \
	done

.PHONY: install-linter
install-linter: ## Installs linter
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.50.0

.PHONY: lint
lint: ## Lints with default set of linters (See https://golangci-lint.run/usage/linters/#enabled-by-default)
	golangci-lint run ./...

.PHONY: test
test: ## Runs go test with race flag.
	go test -v -timeout 30s -race -coverprofile=coverage.out -covermode=atomic ./...
	cd ./examples/server && go test -v -race ./...


.PHONY: coverage
COVER_PROFILE ?= coverage.out
coverage: ## Generates a coverage profile and opens a web browser with the results
	go test -coverprofile=$(COVER_PROFILE) -covermode=atomic
	go tool cover -html=$(COVER_PROFILE)

.PHONY: setup-example-server
setup-example-server:
	cp examples/server/.env.example examples/server/.env
	heroku config -s -a telemetry-go-example >> examples/server/.env

.PHONY: run-example-server
run-example-server:
	CGO_ENABLED=0 go build -o build/server ./examples/server
	./build/server --env ./examples/server/.env --data ./examples/server/testdata/todos.json

EXTRA_ARGS ?=
EXAMPLE_SERVER_HEROKU_APP ?= telemetry-go-example
EXAMPLE_SERVER_DOCKER_FILE ?= Dockerfile.example.server
EXAMPLE_SERVER_PROCESS_TYPE ?= web
export ORG ?= heroku
export IMAGE ?= $(EXAMPLE_SERVER_HEROKU_APP)
export IMAGE_TAG ?= $(shell git rev-parse --short HEAD)
export VERSION ?= $(shell git describe --tags --always --dirty)
export DOCKER_BUILDKIT := 1
export DOCKER_DEFAULT_PLATFORM=linux/amd64
TAG = $(ORG)/$(IMAGE):$(IMAGE_TAG)
.PHONY: build-example-server-image
build-example-server-image:
	docker build $(EXTRA_ARGS) . \
		--file=$(EXAMPLE_SERVER_DOCKER_FILE) \
		--build-arg OBJ=$(EXAMPLE_SERVER_HEROKU_APP) \
		--build-arg IMAGE_TAG=$(IMAGE_TAG) \
		--build-arg VERSION=$(VERSION) \
		--pull \
		--ssh default \
		-t $(TAG)
		docker scan --accept-license --version
		docker scan --file $(EXAMPLE_SERVER_DOCKER_FILE) $(TAG)
		docker tag $(TAG) registry.heroku.com/$(EXAMPLE_SERVER_HEROKU_APP)/$(EXAMPLE_SERVER_PROCESS_TYPE)

.PHONY: deploy-example-server
deploy-example-server: build-example-server-image
	heroku container:login
	docker push registry.heroku.com/$(EXAMPLE_SERVER_HEROKU_APP)/$(EXAMPLE_SERVER_PROCESS_TYPE)
	heroku container:release web -a $(EXAMPLE_SERVER_HEROKU_APP)

.PHONY: setup-example-client
setup-example-client:
	cp examples/client/.env.example examples/client/.env
	heroku config -s -a telemetry-go-example >> examples/client/.env

.PHONY: run-example-client
run-example-client:
	CGO_ENABLED=0 go run examples/client/*.go --env ./examples/client/.env --endpoints ./examples/client/endpoints.json

.PHONY: docs
docs:
	go install github.com/posener/goreadme/cmd/goreadme@v1.4.2
	README_FILE=API.md goreadme \
		-title="Package telemery-go API Docs" \
		-recursive=false \
		-constants=true \
		-variabless=true \
		-functions=true \
		-methods=true \
		-types=true \
		-factories=true \
		-credit=false
