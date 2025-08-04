package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	// NetSapiens API Configuration
	NetsapiensBaseURL  string
	NetsapiensToken    string
	NetsapiensClientID string
	NetsapiensSecret   string

	// Application Configuration
	AppEnv  string
	AppPort string

	// Database Configuration
	DatabasePath string
}

// LoadConfig loads configuration from environment variables and .env file
func LoadConfig() *Config {
	// Load .env file if it exists (for local development)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	config := &Config{
		// NetSapiens Configuration
		NetsapiensBaseURL:  getEnv("NETSAPIENS_BASE_URL", "https://ns-api.com"),
		NetsapiensToken:    getEnv("NETSAPIENS_ACCESS_TOKEN", ""), // Can be empty now
		NetsapiensClientID: getEnv("NETSAPIENS_CLIENT_ID", ""),
		NetsapiensSecret:   getEnv("NETSAPIENS_CLIENT_SECRET", ""),

		// Application Configuration
		AppEnv:  getEnv("APP_ENV", "development"),
		AppPort: getEnv("APP_PORT", "8080"),

		// Database Configuration
		DatabasePath: getEnv("DATABASE_PATH", "./data/odango.db"),
	}

	// Remove the validation since tokens come from users now
	// if config.NetsapiensToken == "" {
	//     log.Fatal("NETSAPIENS_ACCESS_TOKEN is required but not set")
	// }

	return config
}

// getEnv gets an environment variable with a fallback default value
func getEnv(key string, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt gets an environment variable as integer with fallback
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvAsBool gets an environment variable as boolean with fallback
func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// IsProduction checks if we're running in production
func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}

// IsDevelopment checks if we're running in development
func (c *Config) IsDevelopment() bool {
	return c.AppEnv == "development"
}
