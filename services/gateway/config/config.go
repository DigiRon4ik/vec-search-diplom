package config

import "os"

type Config struct {
	Port         string
	CatalogURL   string
	EmbeddingURL string
	SearchURL    string
	FrontendDir  string
}

func Load() *Config {
	return &Config{
		Port:         getEnv("GATEWAY_PORT", "8080"),
		CatalogURL:   getEnv("CATALOG_URL", "http://localhost:8081"),
		EmbeddingURL: getEnv("EMBEDDING_URL", "http://localhost:8082"),
		SearchURL:    getEnv("SEARCH_URL", "http://localhost:8083"),
		FrontendDir:  getEnv("FRONTEND_DIR", "./frontend"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
