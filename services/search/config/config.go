package config

import "os"

type Config struct {
	Port         string
	CatalogURL   string
	EmbeddingURL string
}

func Load() *Config {
	return &Config{
		Port:         getEnv("SEARCH_PORT", "8083"),
		CatalogURL:   getEnv("CATALOG_URL", "http://localhost:8081"),
		EmbeddingURL: getEnv("EMBEDDING_URL", "http://localhost:8082"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
