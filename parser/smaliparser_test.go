package parser

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"testing"

	swagger "github.com/go-openapi/spec"
)

//go:embed testdata/FeaturesApi.smali
var featuresApiSmali string

func TestParseSmaliMethods(t *testing.T) {
	// Parse methods from Smali file
	methods := parseSmaliMethods(featuresApiSmali)
	if len(methods) == 0 {
		t.Fatal("Expected at least one parsed method, got none")
	}

	t.Logf("Parsed %d methods from Smali file", len(methods))
	for _, method := range methods {
		t.Logf("Method: %s, Params: %v", method.Name, method.Params)
	}
}

func TestExtractAPIEndpoints(t *testing.T) {
	// Extract API endpoints
	methods := parseSmaliMethods(featuresApiSmali)
	if len(methods) == 0 {
		t.Fatal("Expected at least one parsed method, got none")
	}
	methods = fillRetrofitAnnotations(methods)

	t.Logf("Parsed %d methods from Smali file", len(methods))
}

func TestEncode(t *testing.T) {
	spec := &swagger.Swagger{}
	spec.Swagger = "2.0"
	spec.Info = &swagger.Info{
		InfoProps: swagger.InfoProps{
			Title:   "Test API",
			Version: "1.0",
		},
	}

	spec.Paths = &swagger.Paths{Paths: map[string]swagger.PathItem{}}

	// Build a PathItem with a GET
	pi := swagger.PathItem{}
	pi.Get = &swagger.Operation{
		OperationProps: swagger.OperationProps{
			Description: "A test endpoint",
		},
	}
	spec.Paths.Paths["/test/path"] = pi

	// Marshal
	b, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))
}
