package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/proxikal/hydra/internal/config"
	"github.com/proxikal/hydra/internal/logger"
	"github.com/proxikal/hydra/internal/proxy"
	"github.com/proxikal/hydra/internal/recorder"
	"github.com/proxikal/hydra/internal/sanitizer"
	"github.com/proxikal/hydra/internal/security"
	"github.com/proxikal/hydra/internal/statestore"
	"github.com/proxikal/hydra/internal/supervisor"
	"github.com/proxikal/hydra/internal/transport"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run Hydra supervisor for a named server",
		RunE:  runCommand,
	}

	cmd.Flags().String("name", "", "Server name from registry")
	_ = cmd.MarkFlagRequired("name")
	_ = viper.BindPFlag("name", cmd.Flags().Lookup("name"))

	cmd.Flags().String("config", "~/.hydra/config.json", "Path to registry config")
	_ = viper.BindPFlag("config", cmd.Flags().Lookup("config"))

	cmd.Flags().String("override", "./hydra.json", "Local override path")
	_ = viper.BindPFlag("override", cmd.Flags().Lookup("override"))

	return cmd
}

func runCommand(cmd *cobra.Command, args []string) error {
	serverName, _ := cmd.Flags().GetString("name")
	registryPath, _ := cmd.Flags().GetString("config")
	overridePath, _ := cmd.Flags().GetString("override")

	log := logger.New("info")
	loader := config.NewLoader(log)

	reg, err := loader.LoadRegistry(registryPath)
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	srvCfg, ok := reg.Servers[serverName]
	if !ok {
		return fmt.Errorf("server %s not found in registry", serverName)
	}

	// Apply defaults
	merged := config.DefaultServerConfig()
	mergeServer := *srvCfg
	config.MergeServerConfig(merged, &mergeServer)

	// Apply local override if present
	if localCfg, err := loader.LoadServerConfig(overridePath); err == nil && localCfg != nil {
		config.MergeServerConfig(merged, localCfg)
	}

	if err := loader.Validate(merged); err != nil {
		return fmt.Errorf("invalid server config: %w", err)
	}

	sup := supervisor.NewSupervisor(
		append([]string{merged.Command}, merged.Args...),
		merged.Behavior.MaxRestarts,
		time.Duration(merged.Behavior.RestartWindowSeconds)*time.Second,
		log,
	)

	store := statestore.New()
	redactor := security.NewRedactorWithReplacement(merged.Security.RedactReplacement)
	rec := recorder.NewRecorder(recorder.Options{
		Enabled:               merged.Recorder.Enabled,
		BufferSize:            merged.Recorder.BufferSize,
		IncludeRequestBodies:  merged.Recorder.IncludeRequestBodies,
		IncludeResponseBodies: merged.Recorder.IncludeResponseBodies,
		RedactPatterns:        merged.Security.RedactPatterns,
	}, redactor, log)

	san := sanitizer.New()

	stdin := os.Stdin
	stdout := os.Stdout
	childTransport := transport.NewStdio(stdin, stdout, log)
	clientTransport := transport.NewStdio(stdin, stdout, log)

	p := proxy.New(
		proxy.Dependencies{
			Logger:              log,
			Sanitizer:           san,
			Supervisor:          sup,
			StateStore:          store,
			Recorder:            rec,
			Redactor:            redactor,
			MaxRestartsInWindow: func() int { return merged.Behavior.MaxRestarts },
			MaxRestarts:         func() int { return merged.Behavior.MaxRestarts },
			Child:               childTransport,
			Client:              clientTransport,
		},
		proxy.Options{
			CollisionPolicy:    merged.InjectableTools.OnCollision,
			CrashExportPath:    merged.Recorder.ExportPath,
			CrashExportEnabled: merged.Recorder.ExportOnCrash,
		},
	)

	return p.Run()
}
