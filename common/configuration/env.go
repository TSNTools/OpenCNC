package configuration

import (
	"os"
	"strings"
)

func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func ParseCSVEnv(key string, defaults ...string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaults
	}

	parts := strings.Split(raw, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		items = append(items, trimmed)
	}

	if len(items) == 0 {
		return defaults
	}

	return items
}

func ParseEnvBool(key string, defaultValue bool) bool {
	raw, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}

	raw = strings.TrimSpace(strings.ToLower(raw))
	switch raw {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return defaultValue
	}
}
