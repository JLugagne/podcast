// Package config loads the service configuration from built-in defaults, an
// optional YAML file and PODCAST_ environment variables.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// envPrefix is the prefix of every environment variable read by Load.
const envPrefix = "PODCAST_"

// defaultPublicURL is the base URL used to build absolute links such as the
// RSS feed enclosures.
const defaultPublicURL = "http://localhost:8080"

// ErrInvalid marks a configuration that fails validation.
var ErrInvalid = errors.New("config: invalid configuration")

// Config is the fully resolved service configuration.
type Config struct {
	HTTP     HTTP     `koanf:"http"`
	Database Database `koanf:"database"`
	Storage  Storage  `koanf:"storage"`
	Upload   Upload   `koanf:"upload"`
	Log      Log      `koanf:"log"`
}

// HTTP holds the public HTTP server settings.
type HTTP struct {
	Addr string `koanf:"addr"`
	// PublicURL is the externally reachable base URL, without a trailing slash.
	PublicURL string `koanf:"public_url"`
}

// Database holds the PostgreSQL connection settings.
type Database struct {
	URL string `koanf:"url"`
}

// Storage holds the blob storage settings.
type Storage struct {
	Dir string `koanf:"dir"`
}

// Upload holds the maximum accepted upload sizes, in bytes.
type Upload struct {
	MaxAudioBytes int64 `koanf:"max_audio_bytes"`
	MaxImageBytes int64 `koanf:"max_image_bytes"`
}

// Log holds the logger settings.
type Log struct {
	Format string `koanf:"format"`
	Level  string `koanf:"level"`
}

// Defaults returns the built-in configuration defaults.
func Defaults() Config {
	return Config{
		HTTP:    HTTP{Addr: ":8080", PublicURL: defaultPublicURL},
		Storage: Storage{Dir: "./data/blobs"},
		Upload: Upload{
			MaxAudioBytes: 2 << 30,  // 2 GiB
			MaxImageBytes: 10 << 20, // 10 MiB
		},
		Log: Log{Format: "text", Level: "info"},
	}
}

// Load reads the optional YAML file at path (skipped when empty), then the
// PODCAST_ environment variables, and validates the result. Environment keys
// use a double underscore for nesting, so PODCAST_DATABASE__URL maps to
// database.url.
func Load(path string) (*Config, error) {
	k := koanf.New(".")
	if err := k.Load(confmap.Provider(defaultsMap(), "."), nil); err != nil {
		return nil, fmt.Errorf("config: load defaults: %w", err)
	}
	if path != "" {
		if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
			return nil, fmt.Errorf("config: load %s: %w", path, err)
		}
	}
	if err := k.Load(env.Provider(".", env.Opt{Prefix: envPrefix, TransformFunc: transformEnv}), nil); err != nil {
		return nil, fmt.Errorf("config: load environment: %w", err)
	}
	return decode(k)
}

// decode merges the loaded keys onto the defaults and validates them.
func decode(k *koanf.Koanf) (*Config, error) {
	cfg := Defaults()
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("config: decode: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// defaultsMap renders the defaults as a koanf conf map.
func defaultsMap() map[string]any {
	c := Defaults()
	return map[string]any{
		"http":     map[string]any{"addr": c.HTTP.Addr, "public_url": c.HTTP.PublicURL},
		"database": map[string]any{"url": c.Database.URL},
		"storage":  map[string]any{"dir": c.Storage.Dir},
		"upload": map[string]any{
			"max_audio_bytes": c.Upload.MaxAudioBytes,
			"max_image_bytes": c.Upload.MaxImageBytes,
		},
		"log": map[string]any{"format": c.Log.Format, "level": c.Log.Level},
	}
}

// transformEnv turns PODCAST_DATABASE__URL into database.url.
func transformEnv(key, value string) (string, any) {
	name := strings.TrimPrefix(key, envPrefix)
	name = strings.ReplaceAll(name, "__", ".")
	return strings.ToLower(name), value
}

// Validate reports the first invalid setting. It normalises the public URL by
// trimming its trailing slash.
func (c *Config) Validate() error {
	switch {
	case strings.TrimSpace(c.HTTP.Addr) == "":
		return fmt.Errorf("%w: http.addr is required", ErrInvalid)
	case strings.TrimSpace(c.Database.URL) == "":
		return fmt.Errorf("%w: database.url is required", ErrInvalid)
	case strings.TrimSpace(c.Storage.Dir) == "":
		return fmt.Errorf("%w: storage.dir is required", ErrInvalid)
	case c.Upload.MaxAudioBytes <= 0:
		return fmt.Errorf("%w: upload.max_audio_bytes must be positive", ErrInvalid)
	case c.Upload.MaxImageBytes <= 0:
		return fmt.Errorf("%w: upload.max_image_bytes must be positive", ErrInvalid)
	case c.Log.Format != "text" && c.Log.Format != "json":
		return fmt.Errorf("%w: log.format must be text or json", ErrInvalid)
	}
	var level slog.Level
	if err := level.UnmarshalText([]byte(c.Log.Level)); err != nil {
		return fmt.Errorf("%w: log.level %q is not a valid level", ErrInvalid, c.Log.Level)
	}
	publicURL := strings.TrimRight(strings.TrimSpace(c.HTTP.PublicURL), "/")
	parsed, err := url.Parse(publicURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("%w: http.public_url must be an absolute http(s) URL", ErrInvalid)
	}
	c.HTTP.PublicURL = publicURL
	return nil
}
