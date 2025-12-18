package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppURL          string
	Username        string
	Password        string
	OTP             string
	Headless        bool
	DailyLimit      int
	CooldownSeconds int
	DBPath          string
}

func Load() (*Config, error) {
	godotenv.Load()

	return &Config{
		AppURL:     getEnv("APP_URL", "http://localhost:8080"),
		Username:   getEnv("APP_USERNAME", "admin"),
		Password:   getEnv("APP_PASSWORD", "password123"),
		OTP:        getEnv("APP_OTP", "123456"),
		Headless:   getEnv("HEADLESS", "false") == "true",
		DailyLimit: 20,
		CooldownSeconds: func() int {
			val, _ := strconv.Atoi(getEnv("COOLDOWN_SECONDS", "60"))
			return val
		}(),
		DBPath: getEnv("DB_PATH", "autolink.db"),
	}, nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
