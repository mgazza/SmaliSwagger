package parser

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"runtime"
	"strings"

	swagger "github.com/go-openapi/spec"
)

// --------------------------------------------------------------------------
// 1) GLOBALS & REGEXES
// --------------------------------------------------------------------------

// We'll store fully qualified class names -> absolute file paths
var classToFilePath = make(map[string]string)

// Keep track of object types we've already parsed to avoid recursion loops
var parsedTypes = make(map[string]bool)

// Regex for `.class ... Luk/co/goptions/...;`
var classDefPattern = regexp.MustCompile(`\.class(?:\s+\w+)*\s+([\w/]+);`)

// Regex to capture the method definition
var methodPattern = regexp.MustCompile(
	`(?m)^\.method\s+(public|private|protected)(?:\s+[\w$]+)*\s+([A-Za-z0-9_$]+)\(([^)]*)\)(\S*)\s*([\s\S]*?)\.end method`)

// Regex to find retrofit annotation in the body
var retrofitHTTPAnnotation = regexp.MustCompile(
	`(?s)\.annotation\s+runtime\s+Lretrofit2/http/([A-Z]+);\s*value\s*=\s*"([^"]+)"`)

// Regex to find parameter blocks
var paramBlockPattern = regexp.MustCompile(`(?s)\.param\s+([vp0-9]+)\s+#\s+(\S+)\s*(.*?)\.end param`)

var paramPathAnnotation = regexp.MustCompile(
	`(?s)\.annotation\s+runtime\s+Lretrofit2/http/Path;\s*value\s*=\s*"([^"]+)"`)
var paramQueryAnnotation = regexp.MustCompile(
	`(?s)\.annotation\s+runtime\s+Lretrofit2/http/Query;\s*value\s*=\s*"([^"]+)"`)

// Regex to find the @Signature annotation for the return type
var signatureAnnotation = regexp.MustCompile(
	`(?s)\.annotation\s+system\s+Ldalvik/annotation/Signature;\s*value\s*=\s*{\s*(.*?)\s*}\s*\.end annotation`)

// Regex to find fields in a smali class, e.g. `.field private final name:Ljava/lang/String;`
var fieldPattern = regexp.MustCompile(`(?m)^\.field\s+(?:public|private|protected)?(?:\s+final)?\s+(\w+):([^;]+;)`)

// --------------------------------------------------------------------------
// 2) DATA STRUCTS
// --------------------------------------------------------------------------

type SmaliParam struct {
	Register string // e.g. "p1"
	TypeSig  string // e.g. "Ljava/lang/String;"
	PathVar  string // e.g. "systemId"
	QueryVar string // e.g. "featureName"
}

type SmaliMethod struct {
	AccessLevel     string
	Name            string
	ParamsSig       string // raw smali parameter signature
	ReturnType      string
	Body            string
	HTTPVerb        string
	HTTPPath        string
	Params          []SmaliParam
	ReturnSignature string // from @Signature annotation
}

// APIEndpoint for swagger
type APIEndpoint struct {
	Path            string
	Method          string
	MethodName      string
	ReturnType      string
	Params          []SmaliParam
	ReturnSignature string
}

// --------------------------------------------------------------------------
// 3) SCANNING ALL .SMALI => BUILD classToFilePath
// --------------------------------------------------------------------------

func ScanAllSmaliClasses(files []string) error {
	log.Printf("Scanning %d smali files to build classToFilePath...", len(files))
	for _, path := range files {
		log.Printf("Reading file: %s", path)
		content, err := os.ReadFile(path)
		if err != nil {
			log.Printf("Could not read %s: %v", path, err)
			continue
		}
		matches := classDefPattern.FindStringSubmatch(string(content))
		if matches != nil {
			clsName := matches[1]
			if !strings.HasSuffix(clsName, ";") {
				clsName += ";"
			}
			log.Printf("Found class: %s => file: %s", clsName, path)
			classToFilePath[clsName] = path
		}
	}
	return nil
}

// --------------------------------------------------------------------------
// 4) PARSING .SMALI for METHODS & FIELDS
// --------------------------------------------------------------------------

func parseSmaliMethods(content string) []SmaliMethod {
	all := methodPattern.FindAllStringSubmatch(content, -1)
	log.Printf("parseSmaliMethods: found %d methods", len(all))

	var methods []SmaliMethod
	for _, m := range all {
		method := SmaliMethod{
			AccessLevel: m[1],
			Name:        m[2],
			ParamsSig:   m[3],
			ReturnType:  m[4],
			Body:        m[5],
		}
		methods = append(methods, method)
	}
	return methods
}

func parseMethodParams(method *SmaliMethod) {
	blocks := paramBlockPattern.FindAllStringSubmatch(method.Body, -1)
	log.Printf("parseMethodParams: method %s has %d param blocks", method.Name, len(blocks))

	var results []SmaliParam
	for _, b := range blocks {
		register := b[1]                   // e.g. "p2"
		typeSig := strings.TrimSpace(b[2]) // e.g. "J" or "Ljava/lang/String;"
		body := b[3]                       // everything until .end param

		// base param
		baseParam := SmaliParam{
			Register: register,
			TypeSig:  typeSig,
		}

		// find all path annotations in this block
		pathMatches := paramPathAnnotation.FindAllStringSubmatch(body, -1)
		// find all query annotations
		queryMatches := paramQueryAnnotation.FindAllStringSubmatch(body, -1)

		if len(pathMatches) == 0 && len(queryMatches) == 0 {
			// no path or query => keep as-is (maybe body param)
			results = append(results, baseParam)
			continue
		}

		// for each path annotation => add a new param
		for _, pm := range pathMatches {
			newParam := baseParam
			newParam.PathVar = pm[1] // e.g. "resourceId"
			results = append(results, newParam)
		}

		// for each query annotation => add a new param
		for _, qm := range queryMatches {
			newParam := baseParam
			newParam.QueryVar = qm[1] // e.g. "startDatetime"
			results = append(results, newParam)
		}
	}

	method.Params = results
}

func parseSignatureAnnotation(method *SmaliMethod) {
	if sigMatches := signatureAnnotation.FindStringSubmatch(method.Body); sigMatches != nil {
		sig := mergeSignatureLines(sigMatches[1])
		method.ReturnSignature = sig
		log.Printf("Method %s has ReturnSignature = %s", method.Name, sig)
	}
}

// mergeSignatureLines merges lines from value = { "<line1>", "<line2>", ... }
func mergeSignatureLines(sigBlock string) string {
	lines := strings.Split(sigBlock, "\",\n")
	var tokens []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.Trim(line, `"`) // remove surrounding quotes
		tokens = append(tokens, line)
	}
	return strings.Join(tokens, "")
}

func fillRetrofitAnnotations(methods []SmaliMethod) []SmaliMethod {
	for i := range methods {
		sm := &methods[i]
		if matches := retrofitHTTPAnnotation.FindStringSubmatch(sm.Body); matches != nil {
			sm.HTTPVerb = strings.ToUpper(matches[1])
			sm.HTTPPath = matches[2]
			log.Printf("Method %s => %s %s", sm.Name, sm.HTTPVerb, sm.HTTPPath)
		}
		parseMethodParams(sm)
		parseSignatureAnnotation(sm)
	}
	return methods
}

// parseFieldsFromFile extracts .field lines => map[fieldName] = fieldSig
func parseFieldsFromFile(filePath string) (map[string]string, error) {
	log.Printf("parseFieldsFromFile: %s", filePath)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	content := string(data)
	matches := fieldPattern.FindAllStringSubmatch(content, -1)
	log.Printf("Found %d fields in %s", len(matches), filePath)

	fields := make(map[string]string)
	for _, m := range matches {
		fieldName := m[1]
		fieldSig := m[2]
		log.Printf("  field: %s => %s", fieldName, fieldSig)
		fields[fieldName] = fieldSig
	}
	return fields, nil
}

// --------------------------------------------------------------------------
// 5) EXTRACT API ENDPOINTS
// --------------------------------------------------------------------------

func ExtractAPIEndpoints(content string) ([]*APIEndpoint, error) {
	methods := parseSmaliMethods(content)
	methods = fillRetrofitAnnotations(methods)

	var apis []*APIEndpoint
	for _, m := range methods {
		if m.HTTPVerb != "" && m.HTTPPath != "" {
			log.Printf("Build APIEndpoint for method=%s path=%s verb=%s", m.Name, m.HTTPPath, m.HTTPVerb)
			apis = append(apis, &APIEndpoint{
				Path:            m.HTTPPath,
				Method:          m.HTTPVerb,
				MethodName:      m.Name,
				ReturnType:      m.ReturnType,
				Params:          m.Params,
				ReturnSignature: m.ReturnSignature,
			})
		}
	}
	return apis, nil
}

// --------------------------------------------------------------------------
// 6) SWAGGER GENERATION
// --------------------------------------------------------------------------

func GenerateSwaggerSpec(endpoints []*APIEndpoint) (*swagger.Swagger, error) {
	log.Printf("Generating Swagger spec from %d endpoints...", len(endpoints))
	spec := &swagger.Swagger{
		SwaggerProps: swagger.SwaggerProps{
			Swagger: "2.0",
			Info: &swagger.Info{
				InfoProps: swagger.InfoProps{
					Title:       "Extracted API",
					Version:     "1.0.0",
					Description: "API extracted from Smali files",
				},
			},
			Paths:       &swagger.Paths{Paths: map[string]swagger.PathItem{}},
			Definitions: map[string]swagger.Schema{},
		},
	}

	for _, endpoint := range endpoints {
		if !strings.HasPrefix(endpoint.Path, "/") {
			endpoint.Path = "/" + endpoint.Path
		}

		pathItem := spec.Paths.Paths[endpoint.Path]
		operation := &swagger.Operation{
			OperationProps: swagger.OperationProps{
				Summary:     fmt.Sprintf("Extracted API for %s", endpoint.MethodName),
				Description: fmt.Sprintf("Generated from smali method: %s", endpoint.MethodName),
				Produces:    []string{"application/json"},
				Responses: &swagger.Responses{
					ResponsesProps: swagger.ResponsesProps{
						StatusCodeResponses: map[int]swagger.Response{},
					},
				},
			},
		}

		swaggerParams := buildSwaggerParams(endpoint, spec)
		operation.Parameters = swaggerParams

		if hasBody(swaggerParams) {
			operation.Consumes = []string{"application/json"}
		}

		if endpoint.ReturnSignature != "" {
			log.Printf("Endpoint %s => ReturnSignature: %s", endpoint.MethodName, endpoint.ReturnSignature)

			kind, itemRef, err := interpretTypeAndBuildDefinition(endpoint.ReturnSignature, spec)
			if err != nil {
				return nil, err
			}
			resp, err := buildResponse(kind, itemRef, spec)
			if err != nil {
				return nil, err
			}
			log.Printf("Endpoint %s => kind=%s itemRef=%s => response built", endpoint.MethodName, kind, itemRef)
			operation.Responses.StatusCodeResponses[200] = *resp
		} else {
			log.Printf("Endpoint %s => no ReturnSignature => default string response", endpoint.MethodName)
			operation.Responses.StatusCodeResponses[200] = swagger.Response{
				ResponseProps: swagger.ResponseProps{
					Description: "OK",
					Schema: &swagger.Schema{
						SchemaProps: swagger.SchemaProps{
							Type: []string{"string"},
						},
					},
				},
			}
		}

		switch strings.ToUpper(endpoint.Method) {
		case "GET":
			pathItem.Get = operation
		case "POST":
			pathItem.Post = operation
		case "PUT":
			pathItem.Put = operation
		case "DELETE":
			pathItem.Delete = operation
		default:
			pathItem.Post = operation
		}
		spec.Paths.Paths[endpoint.Path] = pathItem
	}

	return spec, nil
}

func buildParamSchema(kind, itemRef string, spec *swagger.Swagger) (*swagger.Schema, error) {
	return buildPropertySchema(kind, itemRef)
}

func buildSwaggerParams(endpoint *APIEndpoint, spec *swagger.Swagger) []swagger.Parameter {
	methodUpper := strings.ToUpper(endpoint.Method)
	log.Printf("buildSwaggerParams for %s => method=%s", endpoint.MethodName, endpoint.Method)

	if endpoint.Path == "/eventsservice/v2/system/{systemId}/resource/{resourceId}/history" {
		runtime.Breakpoint()
	}
	var params []swagger.Parameter
	for _, p := range endpoint.Params {
		log.Printf("  param register=%s typeSig=%s pathVar=%s queryVar=%s",
			p.Register, p.TypeSig, p.PathVar, p.QueryVar)

		// 1) If we have a path variable => create path param
		if p.PathVar != "" {
			paramType := smaliTypeToSwaggerType(p.TypeSig)
			sp := swagger.Parameter{
				ParamProps: swagger.ParamProps{
					Name:     p.PathVar,
					In:       "path",
					Required: true,
				},
				SimpleSchema: swagger.SimpleSchema{
					Type: paramType,
				},
			}
			params = append(params, sp)
		}

		// 2) If we have a query variable => create query param
		if p.QueryVar != "" {
			paramType := smaliTypeToSwaggerType(p.TypeSig)
			sp := swagger.Parameter{
				ParamProps: swagger.ParamProps{
					Name: p.QueryVar,
					In:   "query",
				},
				SimpleSchema: swagger.SimpleSchema{
					Type: paramType,
				},
			}
			params = append(params, sp)
		}

		// 3) If we have neither path nor query...
		if p.PathVar == "" && p.QueryVar == "" {
			// ...and method is POST/PUT => treat as body param
			if methodUpper == "POST" || methodUpper == "PUT" {
				if isObjectType(p.TypeSig) {
					log.Printf("    param %s => isObject => body param", p.Register)
					k, itemRef, err := interpretTypeAndBuildDefinition(p.TypeSig, spec)
					if err != nil {
						log.Printf("Error interpreting body param: %v", err)
						k = "object"
						itemRef = ""
					}
					paramSchema, err := buildParamSchema(k, itemRef, spec)
					if err != nil {
						log.Printf("Error building schema for body param: %v", err)
						paramSchema = &swagger.Schema{
							SchemaProps: swagger.SchemaProps{Type: []string{"object"}},
						}
					}
					sp := swagger.Parameter{
						ParamProps: swagger.ParamProps{
							Name:     "body",
							In:       "body",
							Required: true,
							Schema:   paramSchema,
						},
					}
					params = append(params, sp)
				} else {
					// If it's a primitive in a POST/PUT with no annotation,
					// you could treat it as a query param or skip. Let's do query:
					t := smaliTypeToSwaggerType(p.TypeSig)
					log.Printf("    param %s => prim => fallback query with type=%s", p.Register, t)
					sp := swagger.Parameter{
						ParamProps: swagger.ParamProps{
							Name: p.Register,
							In:   "query",
						},
						SimpleSchema: swagger.SimpleSchema{
							Type: t,
						},
					}
					params = append(params, sp)
				}
			} else {
				// If method is GET/DELETE and we have no path/query => skip
				// This prevents "result" parameters from showing up in "query"
				log.Printf("    param %s => skipping for %s method", p.Register, methodUpper)
			}
		}
	}
	return params
}

func hasBody(params []swagger.Parameter) bool {
	for _, p := range params {
		if p.In == "body" {
			return true
		}
	}
	return false
}

func isObjectType(sig string) bool {
	sig = strings.TrimSpace(sig)
	return strings.HasPrefix(sig, "L") && strings.HasSuffix(sig, ";")
}

func smaliTypeToSwaggerType(sig string) string {
	sig = strings.TrimSpace(sig)
	switch sig {
	case "J":
		return "integer"
	case "I", "Ljava/lang/Integer;":
		return "integer"
	case "Z", "Ljava/lang/Boolean;":
		return "boolean"
	case "F", "D":
		return "number"
	case "B", "S":
		return "integer"
	case "Ljava/lang/String;":
		return "string"
	default:
		if isObjectType(sig) {
			if strings.Contains(sig, "String;") {
				return "string"
			}
			//return "object"
		}
		return "string"
	}
}

// genericWrapper describes how to handle "wrapper" types like Observable<T>, List<T>, etc.
// prefix: the signature prefix we match (e.g. "Lio/reactivex/rxjava3/core/Observable<")
// parseInside: a function that handles the substring inside < ... >
//
//	returning the final (kind, itemRef, error).
type genericWrapper struct {
	prefix      string
	parseInside func(inside string, spec *swagger.Swagger) (string, string, error)
}

func genericWrapperFunc(preifx, kindOverride string) *genericWrapper {
	return &genericWrapper{
		prefix: preifx,
		parseInside: func(inside string, spec *swagger.Swagger) (string, string, error) {
			// same logic as List<T>
			kind, iRef, err := interpretTypeAndBuildDefinition(inside, spec)
			if err != nil {
				return "", "", err
			}
			if kindOverride != "" {
				return kindOverride, iRef, nil
			}
			return kind, iRef, nil
		},
	}
}

// For example, handle "Observable<T>" by ignoring the wrapper and returning whatever T is.
var wrapperHandlers []*genericWrapper

func init() {
	wrapperHandlers = []*genericWrapper{
		// For Observable<T>, we skip the wrapper, interpret T directly
		genericWrapperFunc("Lio/reactivex/rxjava3/core/Observable<", ""),
		genericWrapperFunc("Lretrofit2/Call<", ""),
		genericWrapperFunc("Lretrofit2/Response<", ""),
		genericWrapperFunc("Lio/reactivex/rxjava3/core/Single<", ""),
		genericWrapperFunc("Ljava/util/List<", "array"),
		genericWrapperFunc("Ljava/util/ArrayList<", "array"),
		{
			prefix: "Ljava/util/HashMap<",
			parseInside: func(inside string, spec *swagger.Swagger) (string, string, error) {
				// HashMap<K,V> => interpret V => "map", itemRef=...
				parts := splitSmaliTypes(inside)
				if len(parts) == 2 {
					_, valRef, err := interpretTypeAndBuildDefinition(parts[1], spec)
					if err != nil {
						return "", "", err
					}
					return "map", valRef, nil
				}
				// fallback => just "map"
				return "map", "object", nil
			},
		},
	}
}

// --------------------------------------------------------------------------
// 7) interpret & build definitions automatically
// --------------------------------------------------------------------------

func interpretTypeAndBuildDefinition(sig string, spec *swagger.Swagger) (string, string, error) {
	sig = strings.TrimSpace(sig)
	log.Printf("interpretTypeAndBuildDefinition sig=%s", sig)

	// skip method param portion
	if strings.HasPrefix(sig, "(") {
		if closeIdx := strings.Index(sig, ")"); closeIdx != -1 {
			sig = strings.TrimSpace(sig[closeIdx+1:])
			log.Printf("  after dropping method params => %s", sig)
		}
	}

	// 1) Check our generic wrapper list first
	for _, wh := range wrapperHandlers {
		if strings.HasPrefix(sig, wh.prefix) {
			log.Printf("  recognized wrapper prefix = %s", wh.prefix)
			// e.g. "Lio/reactivex/rxjava3/core/Observable<..."
			open := strings.Index(sig, "<")
			if open == -1 {
				// malformed => fallback
				return "object", "", nil
			}
			inside := sig[open+1:]
			inside = strings.TrimSuffix(inside, ">;")
			inside = strings.TrimSpace(inside)
			log.Printf("  inside=%s => pass to parseInside", inside)
			return wh.parseInside(inside, spec)
		}
	}

	// 4) If it's one of the known java.lang wrappers
	switch sig {
	case "Ljava/lang/String;":
		log.Printf("  recognized as built-in String => string")
		return "string", "", nil
	case "Ljava/lang/Boolean;":
		log.Printf("  recognized as built-in Boolean => boolean")
		return "boolean", "", nil
	case "Ljava/lang/Integer;", "Ljava/lang/Long;":
		log.Printf("  recognized as built-in Integer/Long => integer")
		return "integer", "", nil
	case "Ljava/lang/Float;", "Ljava/lang/Double;":
		log.Printf("  recognized as built-in Float/Double => number")
		return "number", "", nil
	case "Ljava/lang/Void;", "Lretrofit2/Response<Ljava/lang/Void;>;":
		log.Printf("  recognized as void => no content")
		return "void", "", nil
	case "Ljava/util/Map;":
		log.Printf("  recognized as built-in Map => map")
		return "map", "object", nil
	}

	// 5) If it's an object => parse the smali to build a real definition
	if strings.HasPrefix(sig, "L") {
		if !strings.HasSuffix(sig, ";") {
			sig += ";"
		}
		if parsedTypes[sig] {
			log.Printf("  already parsed type => %s", sig)
		} else {
			log.Printf("  parse new object => %s", sig)
			parsedTypes[sig] = true
			filePath, ok := classToFilePath[sig]
			shortName := typeShortName(sig)
			if !ok {
				log.Printf("  no file found => minimal def => %s", shortName)
				if _, found := spec.Definitions[shortName]; !found {
					spec.Definitions[shortName] = swagger.Schema{
						SchemaProps: swagger.SchemaProps{
							Type:       []string{"object"},
							Properties: map[string]swagger.Schema{},
						},
					}
				}
				return "object", shortName, nil
			}

			fieldsMap, err := parseFieldsFromFile(filePath)
			if err != nil {
				log.Printf("  parseFieldsFromFile failed => minimal def => %s", shortName)
				spec.Definitions[shortName] = swagger.Schema{
					SchemaProps: swagger.SchemaProps{
						Type:       []string{"object"},
						Properties: map[string]swagger.Schema{},
					},
				}
				return "object", shortName, nil
			}
			log.Printf("  building schema with %d fields => %s", len(fieldsMap), shortName)
			schemaProps := map[string]swagger.Schema{}
			for fieldName, fieldSig := range fieldsMap {
				k, iRef, err := interpretTypeAndBuildDefinition(fieldSig, spec)
				if err != nil {
					return "", "", fmt.Errorf("interpretTypeAndBuildDefinition: %w", err)
				}
				b, err := buildPropertySchema(k, iRef)
				if err != nil {
					return "", "", fmt.Errorf("buildPropertySchema: %w", err)
				}
				schemaProps[fieldName] = *b
			}
			spec.Definitions[shortName] = swagger.Schema{
				SchemaProps: swagger.SchemaProps{
					Type:       []string{"object"},
					Properties: schemaProps,
				},
			}
		}
		return "object", typeShortName(sig), nil
	}

	// 5) If it's known primitive
	switch sig {
	case "V":
		log.Printf("  recognized as void => no content")
		return "void", "", nil
	case "I":
		return "integer", "", nil
	case "Z":
		return "boolean", "", nil
	case "F", "D":
		return "number", "", nil
	case "B", "S":
		return "integer", "", nil
	}

	log.Printf("  fallback => string => %s", sig)
	return "string", "", nil
}

func splitSmaliTypes(s string) []string {
	s = strings.TrimSpace(s)
	var result []string
	i := 0
	for i < len(s) {
		c := s[i]
		switch c {
		case 'L':
			end := strings.IndexRune(s[i:], ';')
			if end == -1 {
				result = append(result, s[i:])
				i = len(s)
			} else {
				part := s[i : i+end+1]
				result = append(result, part)
				i += end + 1
			}
		default:
			result = append(result, string(c))
			i++
		}
	}
	return result
}

func buildResponse(kind string, itemRef string, spec *swagger.Swagger) (*swagger.Response, error) {
	log.Printf("buildResponse => kind=%s itemRef=%s", kind, itemRef)
	desc := "OK"
	switch kind {
	case "void":
		return &swagger.Response{
			ResponseProps: swagger.ResponseProps{
				Description: desc,
			},
		}, nil
	case "string", "integer", "boolean", "number":
		return &swagger.Response{
			ResponseProps: swagger.ResponseProps{
				Description: desc,
				Schema: &swagger.Schema{
					SchemaProps: swagger.SchemaProps{
						Type: []string{kind},
					},
				},
			},
		}, nil
	case "array":
		if itemRef == "" {
			return &swagger.Response{
				ResponseProps: swagger.ResponseProps{
					Description: desc,
					Schema: &swagger.Schema{
						SchemaProps: swagger.SchemaProps{
							Type: []string{"array"},
							Items: &swagger.SchemaOrArray{
								Schema: &swagger.Schema{
									SchemaProps: swagger.SchemaProps{
										Type: []string{"string"},
									},
								},
							},
						},
					},
				},
			}, nil
		}
		ref, err := swagger.NewRef("#/definitions/" + itemRef)
		if err != nil {
			return nil, fmt.Errorf("error building ref: %w", err)
		}
		return &swagger.Response{
			ResponseProps: swagger.ResponseProps{
				Description: desc,
				Schema: &swagger.Schema{
					SchemaProps: swagger.SchemaProps{
						Type: []string{"array"},
						Items: &swagger.SchemaOrArray{
							Schema: &swagger.Schema{
								SchemaProps: swagger.SchemaProps{
									Ref: ref,
								},
							},
						},
					},
				},
			},
		}, nil
	case "map":
		return &swagger.Response{
			ResponseProps: swagger.ResponseProps{
				Description: desc,
				Schema: &swagger.Schema{
					SchemaProps: swagger.SchemaProps{
						Type: []string{"object"},
						AdditionalProperties: &swagger.SchemaOrBool{
							Allows: true,
							Schema: &swagger.Schema{
								SchemaProps: swagger.SchemaProps{
									Type: []string{itemRef},
								},
							},
						},
					},
				},
			},
		}, nil
	case "object":
		if itemRef == "" {
			return &swagger.Response{
				ResponseProps: swagger.ResponseProps{
					Description: desc,
					Schema: &swagger.Schema{
						SchemaProps: swagger.SchemaProps{
							Type: []string{"object"},
						},
					},
				},
			}, nil
		}
		if _, ok := spec.Definitions[itemRef]; !ok {
			spec.Definitions[itemRef] = swagger.Schema{
				SchemaProps: swagger.SchemaProps{
					Type:       []string{"object"},
					Properties: map[string]swagger.Schema{},
				},
			}
		}
		ref, err := swagger.NewRef("#/definitions/" + itemRef)
		if err != nil {
			return nil, fmt.Errorf("error building ref: %w", err)
		}
		return &swagger.Response{
			ResponseProps: swagger.ResponseProps{
				Description: desc,
				Schema: &swagger.Schema{
					SchemaProps: swagger.SchemaProps{
						Ref: ref,
					},
				},
			},
		}, nil
	default:
		log.Printf("  fallback => string response")
		return &swagger.Response{
			ResponseProps: swagger.ResponseProps{
				Description: desc,
				Schema: &swagger.Schema{
					SchemaProps: swagger.SchemaProps{
						Type: []string{"string"},
					},
				},
			},
		}, nil
	}
}

func buildPropertySchema(kind, itemRef string) (*swagger.Schema, error) {
	log.Printf("buildPropertySchema => kind=%s itemRef=%s", kind, itemRef)
	switch kind {
	case "string", "integer", "boolean", "number":
		return &swagger.Schema{
			SchemaProps: swagger.SchemaProps{
				Type: []string{kind},
			},
		}, nil
	case "array":
		if itemRef == "" {
			return &swagger.Schema{
				SchemaProps: swagger.SchemaProps{
					Type: []string{"array"},
					Items: &swagger.SchemaOrArray{
						Schema: &swagger.Schema{
							SchemaProps: swagger.SchemaProps{
								Type: []string{"string"},
							},
						},
					},
				},
			}, nil
		}
		ref, err := swagger.NewRef("#/definitions/" + itemRef)
		if err != nil {
			return nil, fmt.Errorf("error building ref: %w", err)
		}
		return &swagger.Schema{
			SchemaProps: swagger.SchemaProps{
				Type: []string{"array"},
				Items: &swagger.SchemaOrArray{
					Schema: &swagger.Schema{
						SchemaProps: swagger.SchemaProps{
							Ref: ref,
						},
					},
				},
			},
		}, nil
	case "map":
		return &swagger.Schema{
			SchemaProps: swagger.SchemaProps{
				Type: []string{"object"},
				AdditionalProperties: &swagger.SchemaOrBool{
					Allows: true,
					Schema: &swagger.Schema{
						SchemaProps: swagger.SchemaProps{
							Type: []string{itemRef},
						},
					},
				},
			},
		}, nil
	case "object":
		if itemRef == "" {
			return &swagger.Schema{
				SchemaProps: swagger.SchemaProps{
					Type: []string{"object"},
				},
			}, nil
		}
		ref, err := swagger.NewRef("#/definitions/" + itemRef)
		if err != nil {
			return nil, fmt.Errorf("error building ref: %w", err)
		}
		return &swagger.Schema{
			SchemaProps: swagger.SchemaProps{
				Ref: ref,
			},
		}, nil
	default:
		log.Printf("  fallback => string property")
		return &swagger.Schema{
			SchemaProps: swagger.SchemaProps{
				Type: []string{"string"},
			},
		}, nil
	}
}

// typeShortName: from Luk/co/goptions/.../SomeClass; => "SomeClass"
func typeShortName(sig string) string {
	tmp := strings.TrimPrefix(sig, "L")
	tmp = strings.TrimSuffix(tmp, ";")
	parts := strings.Split(tmp, "/")
	p := parts[len(parts)-1]
	p = strings.ReplaceAll(p, "$", "_")
	return p
}
