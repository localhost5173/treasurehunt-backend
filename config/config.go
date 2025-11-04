package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	MongoURI           string
	MongoUsername      string
	MongoPassword      string
	DatabaseName       string
	ServerPort         string
	JWTSecret          string
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	FrontendURL        string
	OpenAIAPIKey       string
}

var AppConfig *Config

func LoadConfig() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	AppConfig = &Config{
		MongoURI:           getEnv("MONGO_URI", "mongodb://admin:password123@localhost:27017/treasureHunt_auth?authSource=admin"),
		MongoUsername:      getEnv("MONGO_USERNAME", "admin"),
		MongoPassword:      getEnv("MONGO_PASSWORD", "password123"),
		DatabaseName:       getEnv("DATABASE_NAME", "treasureHunt_auth"),
		ServerPort:         getEnv("SERVER_PORT", "8080"),
		JWTSecret:          getEnv("JWT_SECRET", "your-secret-key"),
		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:  getEnv("GOOGLE_REDIRECT_URL", "http://localhost:8080/api/auth/google/callback"),
		FrontendURL:        getEnv("FRONTEND_URL", "http://localhost:5173"),
		OpenAIAPIKey:       getEnv("OPENAI_API_KEY", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
