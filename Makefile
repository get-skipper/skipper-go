MODULES := core testing testify ginkgo

.PHONY: build test test-integration lint tidy

build:
	@for m in $(MODULES); do \
	  echo "==> Building $$m"; \
	  (cd $$m && go build ./...); \
	done

# Unit tests only (no Google Sheets API calls).
test:
	@for m in $(MODULES); do \
	  echo "==> Testing $$m"; \
	  (cd $$m && go test ./...); \
	done

# Integration tests: require GOOGLE_CREDS_B64 or service-account-skipper-bot.json.
# Only runs against the core module; integration with framework modules is covered
# by their respective test suites when credentials are present.
test-integration:
	@echo "==> Integration tests (core)"
	(cd core && go test -tags integration ./...)

lint:
	@for m in $(MODULES); do \
	  echo "==> Linting $$m"; \
	  (cd $$m && go vet ./...); \
	done

tidy:
	@for m in $(MODULES); do \
	  echo "==> Tidying $$m"; \
	  (cd $$m && go mod tidy); \
	done
