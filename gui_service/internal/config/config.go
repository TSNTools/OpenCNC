package config

import (
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Port      string
	StaticDir string
}

func Load() (Config, error) {
	port := os.Getenv("GUI_SERVICE_PORT")
	if port == "" {
		port = "8080"
	}

	staticDir, err := findStaticDir()
	if err != nil {
		return Config{}, fmt.Errorf("failed to find static GUI directory: %w", err)
	}

	return Config{Port: port, StaticDir: staticDir}, nil
}

func findStaticDir() (string, error) {
	candidates := []string{
		"gui_service/frontend/dist",
		"frontend/dist",
		filepath.Join("..", "frontend", "dist"),
		"gui_service/static",
		"static",
		filepath.Join("..", "static"),
	}

	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}
	}

	return "", os.ErrNotExist
}
