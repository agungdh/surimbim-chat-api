.PHONY: help build build-prod clean

help: ## Show available commands
	@echo "Usage: make [target]"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' Makefile | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build binary (dev)
	go build -o surimbim-chat-api .

build-prod: ## Build optimized binary (prod)
	CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o surimbim-chat-api .

clean: ## Remove built binary
	rm -f surimbim-chat-api
