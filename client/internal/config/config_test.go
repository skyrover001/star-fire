package config

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func writeTestConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "starfire_config.json")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func TestSaveJoinTokenPreservesOtherSettings(t *testing.T) {
	configPath := writeTestConfig(t, `{"token":"old-token","host":"https://server.test","registered_models":["model-a"]}`)

	if err := SaveJoinToken(configPath, "new-token"); err != nil {
		t.Fatalf("save join token: %v", err)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var saved struct {
		Token            string   `json:"token"`
		Host             string   `json:"host"`
		RegisteredModels []string `json:"registered_models"`
	}
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("parse saved config: %v", err)
	}
	if saved.Token != "new-token" || saved.Host != "https://server.test" || !reflect.DeepEqual(saved.RegisteredModels, []string{"model-a"}) {
		t.Fatalf("unexpected saved config: %#v", saved)
	}
}

func TestLoadConfigExplicitFlagsBeatEnvironmentAndFile(t *testing.T) {
	configPath := writeTestConfig(t, `{"host":"https://file.test","token":"file-token"}`)
	t.Setenv("STARFIRE_HOST", "https://env.test")
	t.Setenv("STARFIRE_TOKEN", "env-token")

	originalFlags := flag.CommandLine
	originalArgs := os.Args
	t.Cleanup(func() {
		flag.CommandLine = originalFlags
		os.Args = originalArgs
	})
	flag.CommandLine = flag.NewFlagSet("config-test", flag.ContinueOnError)
	os.Args = []string{
		"starfire", "-host", "https://flag.test", "-token", "flag-token",
		"-engine", "ollama", "-config", configPath,
	}

	cfg := LoadConfig()
	if cfg.StarFireHost != "https://flag.test" || cfg.JoinToken != "flag-token" {
		t.Fatalf("explicit flags lost precedence: host=%q token=%q", cfg.StarFireHost, cfg.JoinToken)
	}
}

func TestLoadConfigFileSupportsDesktopSchema(t *testing.T) {
	for _, name := range []string{
		"STARFIRE_HOST", "STARFIRE_TOKEN", "STARFIRE_ENGINE", "OLLAMA_HOST",
		"OPENAI_API_KEY", "OPENAI_API_BASE", "STAR_FIRE_INPUT_TOKEN_PRICE_PER_M",
		"STAR_FIRE_OUTPUT_TOKEN_PRICE_PER_M", "STAR_FIRE_CACHED_INPUT_TOKEN_PRICE_PER_M",
	} {
		t.Setenv(name, "")
	}
	cfg := &Config{ConfigFile: writeTestConfig(t, `{
		"host":"https://server.test","token":"file-token","model_mode":"proxy",
		"ollama_host":"http://ollama.test:11434","proxy_base_url":"https://legacy.test/v1",
		"proxy_api_key":"legacy-key","app_port":29527,"ippm":"3.8","oppm":8.3,"cippm":"1.0",
		"proxy_backends":[{"name":"a","base_url":"https://a.test/api","api_key":"key-a"}],
		"model_prices":{"shared":{"engine":"openai","ippm":"4.2","oppm":8.4,"cippm":"1.2"}},
		"registered_models":["shared","local-running"]
	}`)}

	loadConfigFile(cfg, map[string]bool{})

	if cfg.StarFireHost != "https://server.test" || cfg.JoinToken != "file-token" || cfg.LocalInferenceType != "all" {
		t.Fatalf("desktop fields not loaded: %+v", cfg)
	}
	if cfg.APPPort != 29527 || cfg.InputTokenPricePerMillion != 3.8 || cfg.OutputTokenPricePerMillion != 8.3 || cfg.CachedInputTokenPricePerMillion != 1.0 {
		t.Fatalf("numeric fields not loaded: %+v", cfg)
	}
	if len(cfg.ProxyBackends) != 1 || !cfg.ProxyBackends[0].Enabled || cfg.ProxyBackends[0].BaseURL != "https://a.test/api/v1" {
		t.Fatalf("proxy backends not normalized: %+v", cfg.ProxyBackends)
	}
	if cfg.ModelPrices["shared"].InputPrice != 4.2 {
		t.Fatalf("model prices not loaded: %+v", cfg.ModelPrices)
	}
	if len(cfg.RegisteredModels) != 2 || cfg.RegisteredModels[0] != "shared" || cfg.RegisteredModels[1] != "local-running" {
		t.Fatalf("registered models not loaded: %+v", cfg.RegisteredModels)
	}
}

func TestLoadConfigFileDoesNotOverrideEnvironmentOrExplicitFlags(t *testing.T) {
	t.Setenv("STARFIRE_HOST", "https://env.test")
	t.Setenv("STAR_FIRE_INPUT_TOKEN_PRICE_PER_M", "6.5")
	cfg := &Config{
		ConfigFile:                writeTestConfig(t, `{"host":"https://file.test","token":"file-token","ippm":"3.8","oppm":"8.3"}`),
		StarFireHost:              "https://env.test",
		JoinToken:                 "flag-token",
		InputTokenPricePerMillion: 6.5,
	}

	loadConfigFile(cfg, map[string]bool{"token": true})

	if cfg.StarFireHost != "https://env.test" {
		t.Fatalf("file overrode environment host: %q", cfg.StarFireHost)
	}
	if cfg.JoinToken != "flag-token" {
		t.Fatalf("file overrode flag token: %q", cfg.JoinToken)
	}
	if cfg.InputTokenPricePerMillion != 6.5 {
		t.Fatalf("file overrode environment price: %v", cfg.InputTokenPricePerMillion)
	}
	if cfg.OutputTokenPricePerMillion != 8.3 {
		t.Fatalf("file fallback was not applied: %v", cfg.OutputTokenPricePerMillion)
	}
}

func TestValidateConfigAcceptsProxyBackendKey(t *testing.T) {
	cfg := &Config{
		StarFireHost:       "http://server.test",
		JoinToken:          "token",
		LocalInferenceType: "openai",
		ProxyBackends: []ProxyBackend{
			{Name: "a", BaseURL: "https://a.test/v1", APIKey: "key-a", Enabled: true},
		},
	}
	if err := ValidateConfig(cfg); err != nil {
		t.Fatalf("proxy backend key should satisfy OpenAI requirement: %v", err)
	}
}

func TestValidateConfigRequiresEnabledBackendKey(t *testing.T) {
	cfg := &Config{
		StarFireHost:       "http://server.test",
		JoinToken:          "token",
		LocalInferenceType: "openai",
		ProxyBackends: []ProxyBackend{
			{Name: "a", BaseURL: "https://a.test/v1", APIKey: "", Enabled: true},
			{Name: "b", BaseURL: "https://b.test/v1", APIKey: "key-b", Enabled: false},
		},
	}
	if err := ValidateConfig(cfg); err == nil {
		t.Fatalf("expected validation to fail for empty/disabled backend keys")
	}
}

func TestLoadConfigOnlyFileSatisfiesRequiredParams(t *testing.T) {
	for _, name := range []string{
		"STARFIRE_HOST", "STARFIRE_TOKEN", "STARFIRE_ENGINE", "OLLAMA_HOST",
		"OPENAI_API_KEY", "OPENAI_API_BASE", "STAR_FIRE_INPUT_TOKEN_PRICE_PER_M",
		"STAR_FIRE_OUTPUT_TOKEN_PRICE_PER_M", "STAR_FIRE_CACHED_INPUT_TOKEN_PRICE_PER_M",
	} {
		t.Setenv(name, "")
	}

	originalFlags := flag.CommandLine
	originalArgs := os.Args
	t.Cleanup(func() {
		flag.CommandLine = originalFlags
		os.Args = originalArgs
	})
	flag.CommandLine = flag.NewFlagSet("config-test", flag.ContinueOnError)
	configPath := writeTestConfig(t, `{
		"host":"http://server.test","token":"file-token","engine":"openai",
		"proxy_base_url":"https://proxy.test/v1","proxy_api_key":"file-key"
	}`)
	os.Args = []string{"starfire", "-config", configPath}

	cfg := LoadConfig()
	if err := ValidateConfig(cfg); err != nil {
		t.Fatalf("config-file-only invocation should satisfy all required params: %v", err)
	}
	if cfg.LocalInferenceType != "openai" || cfg.OpenAIKey != "file-key" {
		t.Fatalf("proxy key not loaded from file: engine=%q key=%q", cfg.LocalInferenceType, cfg.OpenAIKey)
	}
}
