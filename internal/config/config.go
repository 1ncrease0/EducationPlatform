package config

import (
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env        string     `yaml:"env" env-default:"local"`
	HTTPServer HTTPServer `yaml:"http_server"`
	Postgres   Postgres   `yaml:"postgres"`
	JWT        JWT        `yaml:"jwt"`
	ES         ES         `yaml:"elasticsearch"`
	Minio      Minio      `yaml:"minio"`
}

type Minio struct {
	Endpoint  string                  `yaml:"endpoint" env-default:"minio:9000"`
	AccessKey string                  `yaml:"access_key"`
	SecretKey string                  `yaml:"secret_key"`
	UseSSL    bool                    `yaml:"use_ssl"`
	Buckets   map[string]BucketConfig `yaml:"buckets"`
}

type BucketConfig struct {
	Name       string        `yaml:"name"`
	PresignTTL time.Duration `yaml:"presign_ttl"`
}

type ES struct {
	Hosts    []string `yaml:"hosts"`
	Index    string   `yaml:"index"`
	Password string   `yaml:"password"`
}

type JWT struct {
	SecretKey  string        `yaml:"secret_key"`
	AccessTTL  time.Duration `yaml:"access_token_ttl"`
	RefreshTTL time.Duration `yaml:"refresh_token_ttl"`
}

type Postgres struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
}

type HTTPServer struct {
	Address     string        `yaml:"address" env-default:"localhost:8081"`
	Timeout     time.Duration `yaml:"timeout" env-default:"5s"`
	IdleTimeout time.Duration `yaml:"idle_timeout" env-default:"60s"`
}

func MustLoad() *Config {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {

		log.Fatal("CONFIG_PATH is not set")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("Config file not exist: %s", configPath)
	}

	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("Can not read config file %s", err)
	}

	return &cfg
}
