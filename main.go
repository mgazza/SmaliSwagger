package SmaliSwagger

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/mgazza/SmaliSwagger/parser"
)

func glob(dir string, ext string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		if filepath.Ext(path) == ext {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}

func main() {
	// Define CLI flags
	pathFlag := flag.String("path", "", "Directory containing Smali files (default: current working directory)")
	outputFlag := flag.String("output", "swagger.json", "Path to the output Swagger JSON file")

	// Parse command-line flags
	flag.Parse()

	// Determine Smali directory
	smaliDir := *pathFlag
	if smaliDir == "" {
		// Use first non-flag argument if available
		if flag.NArg() > 0 {
			smaliDir = flag.Arg(0)
		} else {
			// Default to current working directory
			var err error
			smaliDir, err = os.Getwd()
			if err != nil {
				log.Fatalf("Error getting current directory: %v", err)
			}
		}
	}

	log.Printf("Using Smali directory: %s", smaliDir)
	log.Printf("Output file: %s", *outputFlag)

	log.Println("Starting scanning...")

	// 1) Gather all .smali in a directory
	files, err := glob(smaliDir, ".smali")
	if err != nil {
		log.Fatalf("error parsing glob: %v", err)
	}
	log.Printf("Glob found %d files", len(files))

	// 2) Build map from class->file
	err = parser.ScanAllSmaliClasses(files)
	if err != nil {
		log.Fatalf("Error scanning smali: %v", err)
	}

	// 3) Parse each smali for endpoints
	var allEndpoints []*parser.APIEndpoint
	for _, path := range files {
		log.Printf("Extracting endpoints in %s", path)
		content, err := os.ReadFile(path)
		if err != nil {
			log.Printf("error reading file: %v", err)
			continue
		}
		apis, err := parser.ExtractAPIEndpoints(string(content))
		if err == nil {
			if len(apis) > 0 {
				log.Printf("Found %d endpoints in %s", len(apis), path)
			}
			allEndpoints = append(allEndpoints, apis...)
		} else {
			fmt.Printf("Error parsing %s: %v\n", path, err)
		}
	}
	log.Printf("Total endpoints found: %d", len(allEndpoints))

	// 4) Build swagger spec
	spec, err := parser.GenerateSwaggerSpec(allEndpoints)
	if err != nil {
		log.Fatalf("Error generating Swagger spec: %v", err)
	}

	// 5) Output swagger.json
	fw, err := os.Create("swagger.json")
	if err != nil {
		log.Fatalf("Error creating swagger.json: %v", err)
	}
	defer fw.Close()

	e := json.NewEncoder(fw)
	e.SetIndent("", "  ")
	if err = e.Encode(spec); err != nil {
		log.Printf("Error encoding Swagger spec: %v", err)
	}
	log.Println("Done. Wrote swagger.json")
}
