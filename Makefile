MODULES     := core testing testify ginkgo
MODULE_PATH := github.com/get-skipper/skipper-go

.PHONY: build test lint tidy

build:
	@for m in $(MODULES); do \
	  echo "==> Building $$m"; \
	  (cd $$m && go build ./...); \
	done

# Tests skip automatically when Google Sheets credentials are not available.
test:
	go test -p 1 $(addprefix $(MODULE_PATH)/,$(addsuffix /...,$(MODULES)))

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
