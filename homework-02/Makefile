MODULE := github.com/UnusualNick/ozon-go-homework/homework-02

.PHONY: generate
generate:
	protoc -I . \
		--go_out=. --go_opt=module=$(MODULE) \
		--go-grpc_out=. --go-grpc_opt=module=$(MODULE) \
		api/product/v1/product.proto

.PHONY: test
test:
	go test -race ./...

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
	@set -eu; \
	go run ./cmd/product-service & product_pid=$$!; \
	go run ./cmd/cart & cart_pid=$$!; \
	trap 'kill $$product_pid $$cart_pid 2>/dev/null || true' INT TERM EXIT; \
	wait $$product_pid $$cart_pid
