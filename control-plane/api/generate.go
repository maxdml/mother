package api

// To regenerate server.gen.go from the OpenAPI spec, run from the project root:
//   make generate
//
// Note: The spec uses OpenAPI 3.1.0, but oapi-codegen requires 3.0.x.
// The Makefile handles creating a temporary 3.0.3-patched copy for generation.
//
//go:generate sh -c "sed 's/openapi: \"3.1.0\"/openapi: \"3.0.3\"/' openapi.yaml > openapi_gen.yaml && go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -config cfg.yaml openapi_gen.yaml && rm openapi_gen.yaml"
