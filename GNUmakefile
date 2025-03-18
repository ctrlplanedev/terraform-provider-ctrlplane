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

update:
	go get -u ./...

test:
	go test -v -cover -timeout=120s -parallel=10 ./... -skip TestIntegration TestAcc

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
		go test \
		-timeout=$${GO_TEST_TIMEOUT:-120m} \
		-parallel=$${GO_TEST_PARALLEL:-4} \
		-cover \
		$${TEST:-./internal/provider/... ./internal/resources/...} $${TESTARGS}

testacc-quiet:
	@if [ ! -f .env ]; then \
		echo "Error: .env file not found. Copy .env.example to .env and configure it."; \
		exit 1; \
	fi
	@echo "Running acceptance tests with reduced verbosity..."
	source .env && \
	TF_ACC=1 \
	TF_LOG=ERROR \
		CTRLPLANE_TOKEN="$${CTRLPLANE_PROVIDER_TESTING_API_KEY}" \
		CTRLPLANE_WORKSPACE="$${CTRLPLANE_PROVIDER_TESTING_WORKSPACE}" \
		CTRLPLANE_BASE_URL="$${CTRLPLANE_PROVIDER_TESTING_BASE_URL}" \
		go test \
		-timeout=$${GO_TEST_TIMEOUT:-120m} \
		-parallel=$${GO_TEST_PARALLEL:-4} \
		-cover \
		-v=0 \
		$${TEST:-./internal/provider/... ./internal/resources/...} $${TESTARGS}

testint:
	@if [ ! -f .env ]; then \
		echo "Error: .env file not found. Copy .env.example to .env and configure it."; \
		exit 1; \
	fi
	@echo "Running integration tests..."
	source .env && \
	INTEGRATION_TEST_MODE="$${INTEGRATION_TEST_MODE:-autocleanup}" \
	CTRLPLANE_TOKEN="$${CTRLPLANE_PROVIDER_TESTING_API_KEY}" \
	CTRLPLANE_WORKSPACE="$${CTRLPLANE_PROVIDER_TESTING_WORKSPACE}" \
	CTRLPLANE_BASE_URL="$${CTRLPLANE_PROVIDER_TESTING_BASE_URL}" \
	go run github.com/onsi/ginkgo/v2/ginkgo run -v ./internal/integration

testexamples: build
	for dir in examples/resources/*; do \
		cd $$dir && \
		terraform init && \
		terraform plan -out=plan.tfplan && \
		terraform apply plan.tfplan -auto-approve; \
		terraform destroy -auto-approve; \
	done

# Clean test artifacts
clean:
	rm -f terraform.log
	rm -rf .terraform
	rm -f .terraform.lock.hcl
	rm -f terraform.tfstate*

.PHONY: fmt lint test testacc testacc-quiet testint testexamples build install generate clean install-local
