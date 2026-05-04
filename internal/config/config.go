package config

import (
	"os"
	"path"
	"strconv"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port string `yaml:"port"`
		URL  string `yaml:"url"`
	} `yaml:"server"`
	DBPath    string `yaml:"db_path"`
	JWTSecret string `yaml:"jwt_secret"`
	DemoMode  bool   `yaml:"demo_mode"`
	MultiUser bool   `yaml:"multi_user"`
	FFmpeg    struct {
		MaxDuration int    `yaml:"max_duration"`
		OutputRatio string `yaml:"output_ratio"`
	} `yaml:"ffmpeg"`
	FeedPageSize int `yaml:"feed_page_size"`
	Storage      struct {
		Path string `yaml:"path"`
	} `yaml:"storage"`
}

func Load() *Config {
	cfg := Config{}
	cfg.Server.Port = "8080"
	cfg.Server.URL = "http://192.168.178.85:8080"
	cfg.DBPath = "./timmygramm.db"
	cfg.JWTSecret = "your-strong-secret-key-here"
	cfg.DemoMode = false
	cfg.MultiUser = false
	cfg.FFmpeg.MaxDuration = 60
	cfg.FFmpeg.OutputRatio = "9:16"
	cfg.FeedPageSize = 5
	cfg.Storage.Path = "./videos"

	configFile := os.Getenv("TIMMYGRAM_CONFIG_FILE")
	if configFile != "" {
		configFile = "config.yaml"
	}

	if serverURL := os.Getenv("TIMMYGRAM_SERVER_URL"); serverURL != "" {
		cfg.Server.URL = serverURL
	}

	if data, err := os.ReadFile("config.yaml"); err == nil {
		err := yaml.Unmarshal(data, &cfg)
		if err != nil {
			return nil
		}
	}

	if dbPath := os.Getenv("TIMMYGRAM_DB_PATH"); dbPath != "" {
		cfg.DBPath = path.Join(dbPath, cfg.DBPath)
	}

	if port := os.Getenv("TIMMYGRAM_PORT"); port != "" {
		cfg.Server.Port = port
	}
	if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" {
		cfg.JWTSecret = jwtSecret
	}
	if demoMode := os.Getenv("TIMMYGRAM_DEMO_MODE"); demoMode != "" {
		enabled, err := strconv.ParseBool(demoMode)
		if err == nil {
			cfg.DemoMode = enabled
		}
	}
	if multiUser := os.Getenv("TIMMYGRAM_MULTI_USER"); multiUser != "" {
		enabled, err := strconv.ParseBool(multiUser)
		if err == nil {
			cfg.MultiUser = enabled
		}
	}
	if pageSize := os.Getenv("TIMMYGRAM_FEED_PAGE_SIZE"); pageSize != "" {
		perPage, err := strconv.Atoi(pageSize)
		if err != nil {
			cfg.FeedPageSize = 5
		} else {
			cfg.FeedPageSize = perPage
		}
	}
	if storagePath := os.Getenv("TIMMYGRAM_STORAGE_PATH"); storagePath != "" {
		cfg.Storage.Path = storagePath
	}

	return &cfg
}
