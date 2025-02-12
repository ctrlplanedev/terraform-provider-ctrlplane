default: fmt lint install generate

build:
	go build -v ./...

install: build
	go install -v ./...

lint:
	golangci-lint run

generate:
	cd tools; go generate ./...

generate-client:
	cd client; go generate ./...

fmt:
	gofmt -s -w -e .

test:
	go test -v -cover -timeout=120s -parallel=10 ./...

testacc:
	@if [ ! -f .env ]; then \
		echo "Error: .env file not found. Copy .env.example to .env and configure it."; \
		exit 1; \
	fi
	@echo "Running acceptance tests..."
	source .env && \
	TF_ACC=1 \
		CTRLPLANE_TOKEN="$${CTRLPLANE_PROVIDER_TESTING_API_KEY}" \
		CTRLPLANE_WORKSPACE="$${CTRLPLANE_PROVIDER_TESTING_WORKSPACE}" \
		CTRLPLANE_BASE_URL="$${CTRLPLANE_PROVIDER_TESTING_BASE_URL}" \
		go test -v \
		-timeout=$${GO_TEST_TIMEOUT:-120m} \
		-parallel=$${GO_TEST_PARALLEL:-4} \
		-cover \
		./internal/provider/...

# Clean test artifacts
clean:
	rm -f terraform.log
	rm -rf .terraform
	rm -f .terraform.lock.hcl
	rm -f terraform.tfstate*

.PHONY: fmt lint test testacc build install generate clean install-local