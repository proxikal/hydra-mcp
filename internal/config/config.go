package config

// Registry represents the global Hydra configuration (~/.hydra/config.json).
// This is the single source of truth for all MCP servers.
type Registry struct {
	Version  string                   `json:"version"`
	Defaults Defaults                 `json:"defaults"`
	Servers  map[string]*ServerConfig `json:"servers"`
}

// ServerConfig represents a single MCP server configuration.
// Can exist in registry or local override (./hydra.json).
type ServerConfig struct {
	Command         string                `json:"command"`
	Args            []string              `json:"args,omitempty"`
	CWD             string                `json:"cwd,omitempty"`
	EnvFile         string                `json:"env_file,omitempty"`
	Environment     map[string]string     `json:"environment,omitempty"`
	Watch           WatchConfig           `json:"watch"`
	Behavior        BehaviorConfig        `json:"behavior"`
	InjectableTools InjectableToolsConfig `json:"injectable_tools"`
	Recorder        RecorderConfig        `json:"recorder"`
}

// Defaults holds system-wide default values for all servers.
type Defaults struct {
	DebounceMS              int `json:"debounce_ms"`
	GracefulShutdownMS      int `json:"graceful_shutdown_ms"`
	MaxOutputSizeKB         int `json:"max_output_size_kb"`
	LogRateLimitPerSecond   int `json:"log_rate_limit_per_second"`
	RestartQueueMaxMessages int `json:"restart_queue_max_messages"`
	RestartQueueTTLSeconds  int `json:"restart_queue_ttl_seconds"`
	RestartTimeoutSeconds   int `json:"restart_timeout_seconds"`
}

// WatchConfig configures file watching for hot-reload.
type WatchConfig struct {
	Enabled     bool     `json:"enabled"`
	Paths       []string `json:"paths"`
	Extensions  []string `json:"extensions"`
	IgnoreFiles []string `json:"ignore_files,omitempty"`
	Ignore      []string `json:"ignore,omitempty"`
}

// BehaviorConfig controls restart and crash loop behavior.
type BehaviorConfig struct {
	DebounceMS           int    `json:"debounce_ms"`
	RestartDelayMS       int    `json:"restart_delay_ms"`
	MaxRestarts          int    `json:"max_restarts"`
	GracefulShutdownMS   int    `json:"graceful_shutdown_ms"`
	RestartWindowSeconds int    `json:"restart_window_seconds"`
	PreRestartCommand    string `json:"pre_restart_command,omitempty"`
}

// InjectableToolsConfig configures hydra_* tool injection.
type InjectableToolsConfig struct {
	Enabled     bool     `json:"enabled"`
	Tools       []string `json:"tools"`
	OnCollision string   `json:"on_collision"`
}

// RecorderConfig configures traffic recorder.
type RecorderConfig struct {
	Enabled               bool   `json:"enabled"`
	BufferSize            int    `json:"buffer_size"`
	IncludeRequestBodies  bool   `json:"include_request_bodies"`
	IncludeResponseBodies bool   `json:"include_response_bodies"`
	ExportOnCrash         bool   `json:"export_on_crash"`
	ExportPath            string `json:"export_path"`
}

// DefaultRegistry returns a registry with safe defaults.
func DefaultRegistry() *Registry {
	return &Registry{
		Version: "1.0",
		Defaults: Defaults{
			DebounceMS:              500,
			GracefulShutdownMS:      2000,
			MaxOutputSizeKB:         50,
			LogRateLimitPerSecond:   10,
			RestartQueueMaxMessages: 100,
			RestartQueueTTLSeconds:  30,
			RestartTimeoutSeconds:   10,
		},
		Servers: make(map[string]*ServerConfig),
	}
}

// DefaultServerConfig returns a server config with safe defaults.
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Command: "",
		Args:    []string{},
		Watch: WatchConfig{
			Enabled:     false,
			Paths:       []string{"."},
			Extensions:  []string{".py", ".js", ".ts", ".go"},
			IgnoreFiles: []string{".gitignore"},
			Ignore:      []string{".git", "node_modules", "__pycache__", ".hydra"},
		},
		Behavior: BehaviorConfig{
			DebounceMS:           500,
			RestartDelayMS:       0,
			MaxRestarts:          10,
			GracefulShutdownMS:   2000,
			RestartWindowSeconds: 60,
		},
		InjectableTools: InjectableToolsConfig{
			Enabled:     true,
			Tools:       []string{"hydra_restart", "hydra_logs", "hydra_status", "hydra_force_restart"},
			OnCollision: "error",
		},
		Recorder: RecorderConfig{
			Enabled:               false,
			BufferSize:            50,
			IncludeRequestBodies:  false,
			IncludeResponseBodies: false,
			ExportOnCrash:         true,
			ExportPath:            "/tmp/hydra-traffic-{timestamp}.json",
		},
	}
}
