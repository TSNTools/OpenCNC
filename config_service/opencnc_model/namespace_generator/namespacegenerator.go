package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
)

func main() {
	// Create a new Modules instance.
	ms := yang.NewModules()

	// Get all .yang files in the folder "yang_modules"
	yangDir := "yang_modules"
	pattern := filepath.Join(yangDir, "*.yang")
	yangFiles, err := filepath.Glob(pattern)
	if err != nil {
		log.Fatalf("Failed to glob YANG files: %v", err)
	}
	if len(yangFiles) == 0 {
		log.Fatalf("No YANG files found in %q", yangDir)
	}

	// Read each YANG file.
	for _, file := range yangFiles {
		cleanFile := filepath.Clean(file)
		if err := ms.Read(cleanFile); err != nil {
			log.Fatalf("Failed to read YANG file %s: %v", cleanFile, err)
		}
	}

	// Process all loaded modules to resolve includes and imports.
	if errs := ms.Process(); len(errs) > 0 {
		log.Fatalf("YANG processing errors: %v", errs)
	}

	// Build a string builder for the output Go file.
	var b strings.Builder
	b.WriteString("package openCNC_model\n\n")
	b.WriteString("// NamespaceByModule maps YANG module names to their XML namespaces.\n")
	b.WriteString("var NamespaceByModule = map[string]string{\n")

	seen := map[string]bool{}

	// Iterate over modules and write the mapping.
	for name, mod := range ms.Modules {
		if mod == nil || mod.Namespace == nil {
			continue
		}

		// Remove the version part after "@"
		baseName := strings.SplitN(name, "@", 2)[0]

		if seen[baseName] {
			continue
		}
		seen[baseName] = true

		b.WriteString(fmt.Sprintf("\t%q: %q,\n", baseName, mod.Namespace.Name))
	}

	b.WriteString("}\n")

	// Write the built string into a file "namespace_map.go"
	outFile, err := os.Create("namespace_map.go")
	if err != nil {
		log.Fatalf("Failed to create namespace_map.go: %v", err)
	}
	defer outFile.Close()

	if _, err := outFile.WriteString(b.String()); err != nil {
		log.Fatalf("Failed to write namespace map: %v", err)
	}

	fmt.Println("✅ namespace_map.go has been written successfully.")
}
