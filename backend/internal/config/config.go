package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	JWT      JWTConfig      `yaml:"jwt"`
	Auth     AuthConfig     `yaml:"auth"`
	Database DatabaseConfig `yaml:"database"`
	Core     CoreConfig     `yaml:"core"`
	Eval     EvalConfig     `yaml:"eval"`
	Dataset  DatasetConfig  `yaml:"dataset"`
	Log      LogConfig      `yaml:"log"`
}

// 占位符常量：example yaml 与代码共享，启动时根据这两个值判断是否仍在用默认值并打 WARN。
const (
	DefaultJWTSecretPlaceholder     = "replace-with-local-secret"
	DefaultAdminPasswordPlaceholder = "admin123"
)

// AuthConfig 控制初始管理员账号的种子。如果库里还没有该 username，启动时按这里
// 生成 bcrypt 哈希并写入 users 表；后续可通过 POST /auth/change-password 修改。
type AuthConfig struct {
	DefaultAdminUsername string `yaml:"default_admin_username"`
	DefaultAdminPassword string `yaml:"default_admin_password"`
}

type ServerConfig struct {
	Host                string `yaml:"host"`
	Port                int    `yaml:"port"`
	ReadTimeoutSeconds  int    `yaml:"read_timeout_seconds"`
	WriteTimeoutSeconds int    `yaml:"write_timeout_seconds"`
}

type JWTConfig struct {
	Issuer      string `yaml:"issuer"`
	Secret      string `yaml:"secret"`
	ExpireHours int    `yaml:"expire_hours"`
}

type DatabaseConfig struct {
	Driver             string `yaml:"driver"`
	Path               string `yaml:"path"`
	MaxOpenConnections int    `yaml:"max_open_connections"`
	MaxIdleConnections int    `yaml:"max_idle_connections"`
}

type CoreConfig struct {
	GRPCAddress           string `yaml:"grpc_address"`
	ConnectTimeoutSeconds int    `yaml:"connect_timeout_seconds"`
	CallTimeoutSeconds    int    `yaml:"call_timeout_seconds"`
	RetryCount            int    `yaml:"retry_count"`
}

type EvalConfig struct {
	DefaultDatasetType   string `yaml:"default_dataset_type"`
	DefaultTimeoutSecs   int    `yaml:"default_timeout_seconds"`
	OutputDir            string `yaml:"output_dir"`
	SaveAPIKeyEnabled    bool   `yaml:"save_api_key_enabled"`
	MaskAPIKeyOnResponse bool   `yaml:"mask_api_key_on_response"`
}

type DatasetConfig struct {
	OpenCompassDemoDir string `yaml:"opencompass_demo_dir"`
	HuggingFaceMirror  string `yaml:"huggingface_mirror"`  // HF 镜像地址
	DatasetStorageDir  string `yaml:"dataset_storage_dir"` // 数据集本地存储目录
}

type LogConfig struct {
	Level string `yaml:"level"`
	Dir   string `yaml:"dir"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	applyDefaults(&cfg)
	return &cfg, nil
}

func (c ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func (c ServerConfig) ReadTimeout() time.Duration {
	return time.Duration(c.ReadTimeoutSeconds) * time.Second
}

func (c ServerConfig) WriteTimeout() time.Duration {
	return time.Duration(c.WriteTimeoutSeconds) * time.Second
}

func (c JWTConfig) ExpireDuration() time.Duration {
	return time.Duration(c.ExpireHours) * time.Hour
}

func (c CoreConfig) ConnectTimeout() time.Duration {
	return time.Duration(c.ConnectTimeoutSeconds) * time.Second
}

func (c CoreConfig) CallTimeout() time.Duration {
	return time.Duration(c.CallTimeoutSeconds) * time.Second
}

func applyDefaults(cfg *Config) {
	if cfg.Server.Host == "" {
		cfg.Server.Host = "127.0.0.1"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Server.ReadTimeoutSeconds == 0 {
		cfg.Server.ReadTimeoutSeconds = 30
	}
	if cfg.Server.WriteTimeoutSeconds == 0 {
		cfg.Server.WriteTimeoutSeconds = 30
	}
	if cfg.JWT.ExpireHours == 0 {
		cfg.JWT.ExpireHours = 24
	}
	if cfg.Auth.DefaultAdminUsername == "" {
		cfg.Auth.DefaultAdminUsername = "admin"
	}
	if cfg.Auth.DefaultAdminPassword == "" {
		cfg.Auth.DefaultAdminPassword = DefaultAdminPasswordPlaceholder
	}
	if cfg.Database.Driver == "" {
		cfg.Database.Driver = "sqlite"
	}
	if cfg.Database.MaxOpenConnections == 0 {
		cfg.Database.MaxOpenConnections = 1
	}
	if cfg.Eval.DefaultTimeoutSecs == 0 {
		cfg.Eval.DefaultTimeoutSecs = 3600
	}
}
