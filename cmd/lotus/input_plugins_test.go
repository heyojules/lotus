package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildInputPlugins_RegistersPrimitives(t *testing.T) {
	t.Parallel()

	plugins := buildInputPlugins(InputPluginConfig{
		TCPEnabled: true,
		TCPAddr:    "127.0.0.1:4000",
	})

	if len(plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(plugins))
	}
	if plugins[0].Name() != "tcp" {
		t.Fatalf("plugins[0] name = %q, want %q", plugins[0].Name(), "tcp")
	}
	if plugins[1].Name() != "stdin" {
		t.Fatalf("plugins[1] name = %q, want %q", plugins[1].Name(), "stdin")
	}
	if !plugins[0].Enabled() {
		t.Fatal("expected tcp plugin to be enabled when TCPEnabled=true")
	}
}

func TestBuildInputPlugins_TCPDisabled(t *testing.T) {
	t.Parallel()

	plugins := buildInputPlugins(InputPluginConfig{
		TCPEnabled: false,
		TCPAddr:    "127.0.0.1:4000",
	})

	if len(plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(plugins))
	}
	if plugins[0].Enabled() {
		t.Fatal("expected tcp plugin to be disabled when TCPEnabled=false")
	}
}

func TestLoadConfig_AddressResolutionAndProcessorModes(t *testing.T) {
	resetLotusEnv(t)

	tests := []struct {
		name         string
		configYAML   string
		wantErr      bool
		wantHost     string
		wantTCPAddr  string
		wantAPIAddr  string
		wantMode     string
		errSubstring string
	}{
		{
			name: "defaults to localhost host and parse mode",
			configYAML: `
tcp-port: 4100
api-port: 3100
`,
			wantHost:    "127.0.0.1",
			wantTCPAddr: "127.0.0.1:4100",
			wantAPIAddr: "127.0.0.1:3100",
			wantMode:    "parse",
		},
		{
			name: "host applies to derived tcp and api addresses",
			configYAML: `
host: 0.0.0.0
processor: passthrough
tcp-port: 4200
api-port: 3200
`,
			wantHost:    "0.0.0.0",
			wantTCPAddr: "0.0.0.0:4200",
			wantAPIAddr: "0.0.0.0:3200",
			wantMode:    "passthrough",
		},
		{
			name: "explicit addresses override host and ports",
			configYAML: `
host: 0.0.0.0
processor: parse
tcp-port: 4300
api-port: 3300
tcp-addr: 10.0.0.5:9999
api-addr: 10.0.0.5:8888
`,
			wantHost:    "0.0.0.0",
			wantTCPAddr: "10.0.0.5:9999",
			wantAPIAddr: "10.0.0.5:8888",
			wantMode:    "parse",
		},
		{
			name: "invalid processor mode rejected",
			configYAML: `
processor: fastpath
tcp-port: 4400
api-port: 3400
`,
			wantErr:      true,
			errSubstring: "invalid processor mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := writeTempConfig(t, tt.configYAML)
			cfg, err := loadConfig(configPath)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errSubstring != "" && !strings.Contains(err.Error(), tt.errSubstring) {
					t.Fatalf("error = %q, want substring %q", err.Error(), tt.errSubstring)
				}
				return
			}

			if err != nil {
				t.Fatalf("loadConfig returned error: %v", err)
			}

			if cfg.Host != tt.wantHost {
				t.Fatalf("Host = %q, want %q", cfg.Host, tt.wantHost)
			}
			if cfg.TCPAddr != tt.wantTCPAddr {
				t.Fatalf("TCPAddr = %q, want %q", cfg.TCPAddr, tt.wantTCPAddr)
			}
			if cfg.APIAddr != tt.wantAPIAddr {
				t.Fatalf("APIAddr = %q, want %q", cfg.APIAddr, tt.wantAPIAddr)
			}
			if cfg.Processor != tt.wantMode {
				t.Fatalf("Processor = %q, want %q", cfg.Processor, tt.wantMode)
			}
		})
	}
}

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)+"\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func resetLotusEnv(t *testing.T) {
	t.Helper()

	original := make(map[string]string)
	existed := make(map[string]bool)

	for _, kv := range os.Environ() {
		key, value, ok := strings.Cut(kv, "=")
		if !ok || !strings.HasPrefix(key, "LOTUS_") {
			continue
		}
		original[key] = value
		existed[key] = true
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset %s: %v", key, err)
		}
	}

	t.Cleanup(func() {
		for key := range existed {
			if err := os.Unsetenv(key); err != nil {
				t.Fatalf("cleanup unset %s: %v", key, err)
			}
		}
		for key, value := range original {
			if err := os.Setenv(key, value); err != nil {
				t.Fatalf("cleanup restore %s: %v", key, err)
			}
		}
	})
}
