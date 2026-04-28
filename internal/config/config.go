package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port string `yaml:"port"`
		URL  string `yaml:"url"`
	} `yaml:"server"`
	DBPath    string `yaml:"db_path"`
	JWTSecret string `yaml:"jwt_secret"`
	FFmpeg    struct {
		MaxDuration int    `yaml:"max_duration"`
		OutputRatio string `yaml:"output_ratio"`
	} `yaml:"ffmpeg"`
	Storage struct {
		Path string `yaml:"path"`
	} `yaml:"storage"`
}

func Load() *Config {
	cfg := Config{}
	cfg.Server.Port = "8080"
	cfg.Server.URL = "http://192.168.178.85:8080"
	cfg.DBPath = "./timmygram.db"
	cfg.JWTSecret = "your-strong-secret-key-here"
	cfg.FFmpeg.MaxDuration = 60
	cfg.FFmpeg.OutputRatio = "9:16"
	cfg.Storage.Path = "./videos"

	if data, err := os.ReadFile("internal/config/config.yaml"); err == nil {
		yaml.Unmarshal(data, &cfg)
	}

	if port := os.Getenv("PORT"); port != "" {
		cfg.Server.Port = port
	}
	if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" {
		cfg.JWTSecret = jwtSecret
	}
	return &cfg
}
