SHELL := /bin/bash
MODULE := github.com/UnusualNick/ozon-go-homework/homework-03

.PHONY: generate
generate:
	protoc -I . \
		--go_out=. --go_opt=module=$(MODULE) \
		--go-grpc_out=. --go-grpc_opt=module=$(MODULE) \
		api/product/v1/product.proto

.PHONY: test
test:
	go test ./...

.PHONY: test-race
test-race:
	go test -race ./...

.PHONY: test-api
test-api:
	go test -v -count=1 -tags=api ./test/

.PHONY: bench
bench:
	go test -run '^$$' -bench 'Benchmark(GetProduct|AddToCart|GetCart)$$' -benchmem ./internal/product ./internal/cart

.PHONY: build
build:
	go build ./...

.PHONY: run-product
run-product:
	go run ./cmd/product-service

.PHONY: run-cart
run-cart:
	go run ./cmd/cart

.PHONY: run-all
run-all:
	@set -euo pipefail; \
	mkdir -p bin; \
	go build -o bin/product-service ./cmd/product-service; \
	go build -o bin/cart ./cmd/cart; \
	./bin/product-service & product_pid=$$!; \
	./bin/cart & cart_pid=$$!; \
	trap 'kill $$product_pid $$cart_pid 2>/dev/null || true; wait $$product_pid $$cart_pid 2>/dev/null || true' EXIT; \
	wait -n $$product_pid $$cart_pid
