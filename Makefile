MODULES := core testing testify ginkgo

.PHONY: build test test-integration lint tidy

build:
	go build ./...

# Unit tests only (no Google Sheets API calls).
test:
	go test ./...

# Integration tests: require GOOGLE_CREDS_B64 or service-account-skipper-bot.json.
test-integration:
	go test -tags integration ./core/...

lint:
	go vet ./...

tidy:
	go work sync
	@for m in $(MODULES); do \
	  echo "==> Tidying $$m"; \
	  (cd $$m && go mod tidy); \
	done
