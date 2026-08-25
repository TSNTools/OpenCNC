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

func CreateRegistry(dirPath string) (*ModuleRegistry, error) {
	files, err := getFilesWithSubdirectories(dirPath)
	if err != nil {
		return nil, fmt.Errorf("error reading files: %w", err)
	}

	registry := &ModuleRegistry{}

	for _, file := range files {
		module := &YangModule{}

		parts := strings.Split(file.FileName, "@")
		module.Name = parts[0]

		if len(parts) == 2 {
			module.Revision = parts[1]
		} else {
			module.Revision = "No Revision tag found."
		}

		module.Structure = file.Directory
		registry.YangModules = append(registry.YangModules, module)
	}

	return registry, nil
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
				FileName:  d.Name(),
			})
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return filesInfo, nil
}
