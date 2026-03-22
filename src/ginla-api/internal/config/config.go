package config

import "os"

// Config holds all application configuration loaded from environment variables.
type Config struct {
	MongoURI      string
	MongoDatabase string
	Port          string
}

// Load reads configuration from environment variables, applying defaults where unset.
func Load() *Config {
	return &Config{
		MongoURI:      getEnv("MONGO_URI", "mongodb://saturn.local:27017"),
		MongoDatabase: getEnv("MONGO_DATABASE", "ginla"),
		Port:          getEnv("PORT", "8080"),
	}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
