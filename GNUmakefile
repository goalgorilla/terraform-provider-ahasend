default: fmt build test

build:
	go build -v -o bin/terraform-provider-ahasend .

install: build
	go install -v .

# Lint is optional; golangci-lint is not required for local/dev builds.
lint:
	@command -v golangci-lint >/dev/null && golangci-lint run || echo "golangci-lint not installed; skipping"

# Requires Terraform CLI on PATH. Exports schema via local binary + dev_overrides
# (provider is unpublished, so tfplugindocs cannot terraform-init from the Registry).
generate: build
	@tmpdir=$$(mktemp -d); \
	absbin=$$(cd bin && pwd); \
	printf '%s\n' \
		'terraform {' \
		'  required_providers {' \
		'    ahasend = { source = "goalgorilla/ahasend" }' \
		'  }' \
		'}' \
		'provider "ahasend" {}' \
		> "$$tmpdir/main.tf"; \
	printf '%s\n' \
		'provider_installation {' \
		'  dev_overrides {' \
		'    "goalgorilla/ahasend" = "'"$$absbin"'"' \
		'  }' \
		'  direct {}' \
		'}' \
		> "$$tmpdir/terraformrc"; \
	TF_CLI_CONFIG_FILE="$$tmpdir/terraformrc" terraform -chdir="$$tmpdir" providers schema -json > "$$tmpdir/schema.json"; \
	python3 -c 'import json,sys; p=sys.argv[1]; d=json.load(open(p)); k="registry.terraform.io/goalgorilla/ahasend";\
d["provider_schemas"]["ahasend"]=d["provider_schemas"].pop(k); json.dump(d, open(p,"w"))' "$$tmpdir/schema.json"; \
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@v0.22.0 generate \
		--provider-name ahasend \
		--rendered-provider-name ahasend \
		--providers-schema "$$tmpdir/schema.json"; \
	rm -rf "$$tmpdir"

# Refresh the pinned OpenAPI contract snapshot from AhaSend.
# After syncing: review the diff, bump github.com/AhaSend/ahasend-go if needed
# (go get + go mod tidy), then fix resources for any breaking schema changes.
openapi-sync:
	curl -fsSL -o openapi/openapi.yaml https://ahasend.com/docs/openapi.yaml
	@echo "Updated openapi/openapi.yaml. Review the diff and bump ahasend-go if the SDK changed."

fmt:
	gofmt -s -w -e ./main.go ./internal

test:
	go test -v -cover -timeout=120s -parallel=10 ./...

# Acceptance tests need a live AhaSend account (AHASEND_API_KEY, AHASEND_ACCOUNT_ID).
# Not wired yet; unit tests cover the provider package for v0.1.0.
testacc:
	@echo "TF_ACC acceptance tests are not configured yet."
	@exit 1

.PHONY: fmt lint test testacc build install generate openapi-sync
