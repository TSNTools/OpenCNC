package yang_modules

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
)

func TreeParser(filePath string) {
	// Extract the file name without the extension
	fileName := strings.TrimSuffix(path.Base(filePath), path.Ext(filePath))

	// Define the desired directory where the output file will be created
	outputDir := "pkg/yang_modules/trees"

	// Load the YANG modules
	modules := yang.NewModules()
	if err := modules.Read(filePath); err != nil {
		log.Fatalf("Failed to read YANG file: %v", err)
	}

	// Process the modules
	if errs := modules.Process(); len(errs) > 0 {
		log.Fatalf("Failed to process YANG modules: %v", errs)
	}

	// Retrieve the root module
	module, found := modules.Modules[fileName]
	if !found {
		log.Fatalf("Module not found: %v", fileName)
	}

	// Convert the module into an entry for traversal
	rootEntry := yang.ToEntry(module)
	if rootEntry == nil {
		log.Fatalf("Failed to convert module to entry")
	}

	// Define the output file path
	outputFilePath := path.Join(outputDir, fileName+"_tree_output.txt")
	file, err := os.Create(outputFilePath)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer file.Close()

	// Print the module tree to the file
	fmt.Fprintln(file, "Module:", fileName)
	printModuleTree(rootEntry, "", file)
	fmt.Printf("Module tree has been written to %s\n", outputFilePath)
}

// Recursive function to print the module tree and write to a file
func printModuleTree(entry *yang.Entry, indent string, file *os.File) {
	for _, child := range entry.Dir {
		accessType := getAccessType(child)
		dataType := getTypeY(child)
		// Now indent the +-- and access type correctly but keep the columns aligned
		// Ensure that the columns (access type, name, data type) start at the same position
		fmt.Fprintf(file, "%s+--%-5s %-20s %-15s\n", indent, accessType, child.Name, dataType)

		// Recursively print child nodes if any
		if len(child.Dir) > 0 {
			printModuleTree(child, indent+"  ", file)
		}
	}
}

// Determine access type (rw or ro)
func getAccessType(entry *yang.Entry) string {
	if entry.Config == yang.TSFalse {
		return "ro" // Read-Only
	}
	return "rw" // Read-Write
}

// Get the YANG data type (for now, we just return the type name)
func getTypeY(entry *yang.Entry) string {
	if entry.Type == nil {
		return "unknown"
	}
	// Return the name of the type
	return entry.Type.Name
}
