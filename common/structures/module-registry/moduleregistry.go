package moduleregistry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type FileInfo struct {
	Directory string
	FileName  string
}

func (registry *ModuleRegistry) CreateRegistry(dirPath string) {
	// Read file names from the directory
	files, err := getFilesWithSubdirectories(dirPath)
	if err != nil {
		fmt.Printf("Error reading files: %v\n", err)
		return
	}

	// Create store prexies from file names and revisions
	for _, file := range files {
		// Create a new instance of YangModule for each iteration
		module := &YangModule{} // Ensure each iteration gets a fresh instance

		// Split the filename on the "@" character
		parts := strings.Split(file.FileName, "@")

		module.Name = parts[0]

		// Set the Revision based on the split parts
		if len(parts) == 2 {
			module.Revision = parts[1]
		} else {
			module.Revision = "No Revision tag found."
		}

		// Set the Structure (Directory)
		module.Structure = file.Directory

		// Append the new module to the registry
		registry.YangModules = append(registry.YangModules, module)
	}
	registry.PrintModuleRegistry()
}

// Method to print the ModuleRegistry (as part of ModuleRegistry)
func (registry *ModuleRegistry) PrintModuleRegistry() {
	if registry == nil || len(registry.YangModules) == 0 {
		fmt.Println("ModuleRegistry is empty.")
		return
	}

	// Print each YangModule in the registry
	for i, module := range registry.YangModules {
		fmt.Printf("Module %d:\n", i+1)
		fmt.Printf("  Name: %s\n", module.Name)
		fmt.Printf("  Structure: %s\n", module.Structure)
		fmt.Printf("  Revision: %s\n", module.Revision)
		fmt.Println() // For spacing between modules
	}
}

func getFilesWithSubdirectories(dirPath string) ([]FileInfo, error) {
	var filesInfo []FileInfo

	// Walk through the directory and its subdirectories
	err := filepath.WalkDir(dirPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories but note their names
		if d.IsDir() {
			return nil
		}

		if filepath.Ext(d.Name()) == ".yang" {
			// Get the subdirectory name (relative to the root directory)
			subdirectory := filepath.Base(filepath.Dir(path))

			// Add the file info with the subdirectory name
			filesInfo = append(filesInfo, FileInfo{
				Directory: subdirectory,
				FileName:  strings.TrimSuffix(d.Name(), filepath.Ext(d.Name())),
			})
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return filesInfo, nil
}
