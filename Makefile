.PHONY: all build generate test clean base-image

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

base-image:
	./tools/build-base-image.sh

clean:
	rm -f bin/mother bin/coder
	rm -f api/openapi_gen.yaml
