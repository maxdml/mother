.PHONY: all build generate test clean

all: generate build test

build:
	go build -o bin/mother ./cmd/mother
	go build -o bin/coder ./cmd/coder

generate:
	cd api && sed 's/openapi: "3.1.0"/openapi: "3.0.3"/' openapi.yaml > openapi_gen.yaml && \
		go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -config cfg.yaml openapi_gen.yaml && \
		rm openapi_gen.yaml

test:
	go test -v ./...

clean:
	rm -f bin/mother bin/coder
	rm -f api/openapi_gen.yaml
