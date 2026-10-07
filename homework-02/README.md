# Homework 2: product-service and cart

The mandatory part of the Ozon Tech Go homework plus all three optional tasks:

- `product-service`: gRPC on `:50051`, `GetProduct`, paginated `ListProducts`;
- `cart`: HTTP on `:8080`, add/get cart, request logging, OpenAPI at
  `GET /swagger.json`;
- in-memory state with concurrency-safe access;
- generated protobuf code, validation and the specified error mapping.

## Layout

```text
api/product/v1/       protobuf source
cmd/product-service/  gRPC entry point
cmd/cart/             HTTP entry point
internal/product/     JSON repository and gRPC implementation
internal/cart/        cart storage, gRPC client and HTTP transport
pkg/product/v1/       generated protobuf code
products.json         source data supplied with the assignment
```

The Telegram attachment is named `data.json`, while the assignment calls it
`products.json`. The repository keeps the same bytes under the expected name.

## Commands

Requirements: Go 1.24+, `protoc`, `protoc-gen-go`, and
`protoc-gen-go-grpc`.

```bash
make generate
make test
make build
make run-all
```

Configuration:

- `PRODUCT_FILE` (default `products.json`)
- `PRODUCT_GRPC_ADDR` (default `:50051`)
- `PRODUCT_SERVICE_ADDR` (default `localhost:50051`)
- `CART_HTTP_ADDR` (default `:8080`)

## Manual check

Choose an existing SKU from `products.json`, then run:

```bash
curl -i -X POST http://localhost:8080/user/1007/cart/2958025 \
  -H 'Content-Type: application/json' \
  -d '{"count":2}'

curl -i http://localhost:8080/user/1007/cart
```

The POST success response has no body. An unknown SKU returns HTTP 412; invalid
IDs or count return HTTP 400; an absent/empty cart returns HTTP 404. Cart items
are sorted by ascending SKU.

This directory is a local draft only. It has not been committed, pushed, or
submitted to the instructor.

