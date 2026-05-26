package config

import (
	"errors"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                int
	DatabaseURL         string
	JWTSecret           string
	FrontendURL         string
	CronSecret          string
	ZohoEmail           string
	ZohoAppPassword     string
	NewsDataAPIKey      string
	TheNewsAPIKey       string
	CloudinaryCloudName string
	CloudinaryPreset    string
	NodeEnv             string
	RunMigrations       bool
}

func Load() (*Config, error) {
	// load .env file -silently skip if not found

	_ = godotenv.Load()

	port, err := strconv.Atoi(getEnv("PORT", "5000"))

	if err != nil {
		return nil, errors.New("invalid PORT: must be a number")
	}

	cfg := &Config{
		Port:                port,
		DatabaseURL:         getEnv("DATABASE_URL", ""),
		JWTSecret:           getEnv("JWT_SECRET", ""),
		FrontendURL:         getEnv("FRONTEND_URL", "http://localhost:3000"),
		CronSecret:          getEnv("CRON_SECRET", ""),
		ZohoEmail:           getEnv("ZOHO_EMAIL", ""),
		ZohoAppPassword:     getEnv("ZOHO_APP_PASSWORD", ""),
		NewsDataAPIKey:      getEnv("NEWSDATA_API_KEY", ""),
		TheNewsAPIKey:       getEnv("THENEWS_API_KEY", ""),
		CloudinaryCloudName: getEnv("CLOUDINARY_CLOUD_NAME", ""),
		CloudinaryPreset:    getEnv("CLOUDINARY_UPLOAD_PRESET", ""),
		NodeEnv:             getEnv("NODE_ENV", "development"),
		RunMigrations:       getEnv("RUN_MIGRATIONS", "true") == "true",
	}

	if cfg.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}

	if cfg.JWTSecret == "" {
		return nil, errors.New("JWT_SECRET is required")
	}

	if cfg.FrontendURL == "" {
		return nil, errors.New("FRONTEND_URL is required")
	}

	if cfg.CronSecret == "" {
		return nil, errors.New("CRON_SECRET is required")
	}

	if cfg.ZohoEmail == "" {
		return nil, errors.New("ZOHO_EMAIL is required")
	}

	if cfg.ZohoAppPassword == "" {
		return nil, errors.New("ZOHO_APP_PASSWORD is required")
	}

	if cfg.NewsDataAPIKey == "" {
		return nil, errors.New("NEWSDATA_API_KEY is required")
	}


	if cfg.TheNewsAPIKey == "" {
		return nil, errors.New("THENEWS_API_KEY is required")
	}


	if cfg.CloudinaryCloudName == "" {
		return nil, errors.New("CLOUDINARY_CLOUD_NAME is required")
	}


	if cfg.CloudinaryPreset == "" {
		return nil, errors.New("CLOUDINARY_UPLOAD_PRESET is required")
	}

	
	return cfg, nil
}

func getEnv(key, fallback string) string {

	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
