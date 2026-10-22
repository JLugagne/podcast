package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/v2"
)

func loadInto(t *testing.T, providers ...func(*koanf.Koanf) error) *Config {
	t.Helper()
	k := koanf.New(".")
	for _, load := range providers {
		if err := load(k); err != nil {
			t.Fatalf("load provider: %v", err)
		}
	}
	cfg, err := decode(k)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	return cfg
}

func withDefaults() func(*koanf.Koanf) error {
	return func(k *koanf.Koanf) error {
		return k.Load(confmap.Provider(defaultsMap(), "."), nil)
	}
}

func withEnv(environ ...string) func(*koanf.Koanf) error {
	return func(k *koanf.Koanf) error {
		provider := env.Provider(".", env.Opt{
			Prefix:        envPrefix,
			TransformFunc: transformEnv,
			EnvironFunc:   func() []string { return environ },
		})
		return k.Load(provider, nil)
	}
}

func withMap(m map[string]any) func(*koanf.Koanf) error {
	return func(k *koanf.Koanf) error {
		return k.Load(confmap.Provider(m, "."), nil)
	}
}

func TestDecodeDefaults(t *testing.T) {
	cfg := loadInto(t, withDefaults(), withMap(map[string]any{
		"database": map[string]any{"url": "postgres://localhost/podcast"},
	}))

	if cfg.HTTP.Addr != ":8080" {
		t.Errorf("http.addr = %q, want :8080", cfg.HTTP.Addr)
	}
	if cfg.HTTP.PublicURL != "http://localhost:8080" {
		t.Errorf("http.public_url = %q, want http://localhost:8080", cfg.HTTP.PublicURL)
	}
	if cfg.Storage.Dir != "./data/blobs" {
		t.Errorf("storage.dir = %q, want ./data/blobs", cfg.Storage.Dir)
	}
	if cfg.Upload.MaxAudioBytes != 2<<30 {
		t.Errorf("upload.max_audio_bytes = %d, want %d", cfg.Upload.MaxAudioBytes, int64(2<<30))
	}
	if cfg.Upload.MaxImageBytes != 10<<20 {
		t.Errorf("upload.max_image_bytes = %d, want %d", cfg.Upload.MaxImageBytes, int64(10<<20))
	}
	if cfg.Log.Format != "text" || cfg.Log.Level != "info" {
		t.Errorf("log = %+v, want text/info", cfg.Log)
	}
}

func TestDecodeEnvNesting(t *testing.T) {
	cfg := loadInto(t, withDefaults(), withEnv(
		"PODCAST_DATABASE__URL=postgres://db/podcast",
		"PODCAST_HTTP__ADDR=:9999",
		"PODCAST_UPLOAD__MAX_AUDIO_BYTES=1024",
		"PODCAST_LOG__LEVEL=debug",
		"PATH=/usr/bin",
	))

	if cfg.Database.URL != "postgres://db/podcast" {
		t.Errorf("database.url = %q", cfg.Database.URL)
	}
	if cfg.HTTP.Addr != ":9999" {
		t.Errorf("http.addr = %q, want :9999", cfg.HTTP.Addr)
	}
	if cfg.Upload.MaxAudioBytes != 1024 {
		t.Errorf("upload.max_audio_bytes = %d, want 1024", cfg.Upload.MaxAudioBytes)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("log.level = %q, want debug", cfg.Log.Level)
	}
}

func TestDecodeEnvOverridesYAMLAndDefaults(t *testing.T) {
	cfg := loadInto(t,
		withDefaults(),
		withMap(map[string]any{"http": map[string]any{"addr": ":7000"}, "database": map[string]any{"url": "postgres://yaml/podcast"}}),
		withEnv("PODCAST_HTTP__ADDR=:9999"),
	)
	if cfg.HTTP.Addr != ":9999" {
		t.Errorf("http.addr = %q, want :9999", cfg.HTTP.Addr)
	}
}

func TestDecodeValidation(t *testing.T) {
	tests := map[string]map[string]any{
		"missing database url": {},
		"bad log format":       {"database": map[string]any{"url": "postgres://x"}, "log": map[string]any{"format": "xml"}},
		"bad log level":        {"database": map[string]any{"url": "postgres://x"}, "log": map[string]any{"level": "loud"}},
		"bad audio size":       {"database": map[string]any{"url": "postgres://x"}, "upload": map[string]any{"max_audio_bytes": 0}},
	}
	for name, m := range tests {
		t.Run(name, func(t *testing.T) {
			k := koanf.New(".")
			if err := k.Load(confmap.Provider(defaultsMap(), "."), nil); err != nil {
				t.Fatalf("load defaults: %v", err)
			}
			if err := k.Load(confmap.Provider(m, "."), nil); err != nil {
				t.Fatalf("load map: %v", err)
			}
			if _, err := decode(k); !errors.Is(err, ErrInvalid) {
				t.Fatalf("decode error = %v, want ErrInvalid", err)
			}
		})
	}
}

func TestLoadYAMLFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "podcast.yaml")
	content := "http:\n  addr: \":7777\"\ndatabase:\n  url: postgres://file/podcast\nlog:\n  format: json\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write yaml: %v", err)
	}
	t.Setenv("PODCAST_LOG__LEVEL", "warn")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.HTTP.Addr != ":7777" || cfg.Database.URL != "postgres://file/podcast" || cfg.Log.Format != "json" {
		t.Errorf("unexpected config: %+v", cfg)
	}
	if cfg.Log.Level != "warn" {
		t.Errorf("log.level = %q, want env override warn", cfg.Log.Level)
	}
}

func TestLoadMissingFile(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Fatal("Load with missing file = nil error, want error")
	}
}

// TestPublicURL covers the http.public_url default, its normalisation and its
// validation.
func TestPublicURL(t *testing.T) {
	tests := map[string]struct {
		value   string
		want    string
		wantErr bool
	}{
		"default":        {want: "http://localhost:8080"},
		"https":          {value: "https://podcast.example", want: "https://podcast.example"},
		"trailing slash": {value: "https://podcast.example/", want: "https://podcast.example"},
		"relative":       {value: "/feeds", wantErr: true},
		"wrong scheme":   {value: "ftp://podcast.example", wantErr: true},
		"missing host":   {value: "https://", wantErr: true},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			k := koanf.New(".")
			if err := k.Load(confmap.Provider(defaultsMap(), "."), nil); err != nil {
				t.Fatalf("load defaults: %v", err)
			}
			environ := []string{"PODCAST_DATABASE__URL=postgres://db/podcast"}
			if tc.value != "" {
				environ = append(environ, "PODCAST_HTTP__PUBLIC_URL="+tc.value)
			}
			if err := k.Load(env.Provider(".", env.Opt{
				Prefix:        envPrefix,
				TransformFunc: transformEnv,
				EnvironFunc:   func() []string { return environ },
			}), nil); err != nil {
				t.Fatalf("load environment: %v", err)
			}
			cfg, err := decode(k)
			if tc.wantErr {
				if !errors.Is(err, ErrInvalid) {
					t.Fatalf("decode error = %v, want ErrInvalid", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if cfg.HTTP.PublicURL != tc.want {
				t.Fatalf("http.public_url = %q, want %q", cfg.HTTP.PublicURL, tc.want)
			}
		})
	}
}
