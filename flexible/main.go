package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	managerconfig "github.com/slackmgr/core/config"
	managerpkg "github.com/slackmgr/core/manager"
	api "github.com/slackmgr/core/restapi"
	"github.com/slackmgr/examples/flexible/config"
	common "github.com/slackmgr/types"
	"golang.org/x/sync/errgroup"
)

func main() {
	exitMain(mainImpl())
}

func mainImpl() (retErr error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	defer func() {
		if r := recover(); r != nil {
			retErr = fmt.Errorf("panic: %v\n%s", r, debug.Stack())
		}
	}()

	go handleSignals(ctx, cancel)

	cfg := config.New()
	logger := newLogger(cfg)

	var metrics common.Metrics

	if cfg.EnableMetrics {
		metrics = NewPrometheusMetrics()
		go func() {
			http.Handle("/metrics", promhttp.Handler())
			if err := http.ListenAndServe(":"+cfg.MetricsPort, nil); err != nil {
				logger.Errorf("Metrics server error: %s", err)
			}
		}()
	} else {
		metrics = &common.NoopMetrics{}
	}

	redisClient, err := newRedisClient(&cfg.Redis)
	if err != nil {
		return fmt.Errorf("failed to create redis client: %w", err)
	}

	// Create a new cache store with redis as the backend.
	cacheStore := newCacheStore(redisClient)

	// Create a new channel locker with redis as the backend.
	// This is used to prevent multiple manager instances from processing the same channel simultaneously.
	// In a single instance setup, the channel locker is not necessary. Just set it to nil, and the manager will skip locking.
	// In a multi-instance setup (e.g in k8s), the channel locker is very much necessary.
	channelLocker := managerpkg.NewRedisChannelLocker(redisClient)

	// Create an alert queue. The type of queue created depends on the QueueMode setting in the config.
	alertQueue, err := newAlertQueue(ctx, redisClient, channelLocker, cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to create alert queue: %w", err)
	}

	// Create a command queue. The type of queue created depends on the QueueMode setting in the config.
	commandQueue, err := newCommandQueue(ctx, redisClient, channelLocker, cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to create command queue: %w", err)
	}

	// Create the database client. The type of database created depends on the DatabaseMode setting in the config.
	db, err := newDatabase(ctx, cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to create database client: %w", err)
	}

	// Create the manager configuration, using the defaults and overriding with values from the config.
	managerCfg := managerconfig.NewDefaultManagerConfig()
	managerCfg.SlackClient.BotToken = cfg.Slack.BotToken
	managerCfg.SlackClient.AppToken = cfg.Slack.AppToken
	managerCfg.EncryptionKey = cfg.EncryptionKey
	managerCfg.Location = getLocation(cfg)
	managerCfg.SkipDatabaseCache = cfg.SkipDatabaseCache

	// Validate the manager configuration.
	if err := managerCfg.Validate(); err != nil {
		return fmt.Errorf("invalid manager configuration: %w", err)
	}

	// Read the manager settings from the yaml file specified in the config.
	// Unlike the config, these settings can be changed at runtime and hot-reloaded.
	managerSettings, managerSettingsHash, err := readManagerSettings(cfg.ManagerSettingsFilename)
	if err != nil {
		return fmt.Errorf("failed to read manager settings: %w", err)
	}

	// Create the API configuration, using the defaults and overriding with values from the config.
	apiCfg := managerconfig.NewDefaultAPIConfig()
	apiCfg.Verbose = cfg.Verbose
	apiCfg.LogJSON = cfg.LogJSON
	apiCfg.RestPort = cfg.RestPort
	apiCfg.SlackClient.BotToken = cfg.Slack.BotToken
	apiCfg.SlackClient.AppToken = cfg.Slack.AppToken
	apiCfg.EncryptionKey = cfg.EncryptionKey

	// Validate the API configuration.
	if err := apiCfg.Validate(); err != nil {
		return fmt.Errorf("invalid API configuration: %w", err)
	}

	// Read the API settings from the yaml file specified in the config.
	// Unlike the config, these settings can be changed at runtime and hot-reloaded.
	apiSettings, apiSettingsHash, err := readAPISettings(cfg.APISettingsFilename)
	if err != nil {
		return fmt.Errorf("failed to read API settings: %w", err)
	}

	// Create the manager instance. This is the main application component, which handles alert processing.
	manager := managerpkg.New(db, alertQueue, commandQueue, cacheStore, channelLocker, logger, metrics, managerCfg, managerSettings)

	// Create the API server instance. This provides the REST API, where clients send alerts.
	apiServer := api.New(alertQueue, cacheStore, logger, metrics, apiCfg, apiSettings)

	// Start the manager and API server in separate goroutines.
	// Also start a goroutine to periodically check for changes in the settings files and hot-reload them.
	//
	// Note! In a production system, you may want to separate the API server and manager into two different services.
	// This allows for better scaling and isolation. We combine them here for simplicity.
	errg, ctx := errgroup.WithContext(ctx)

	// Start the API server.
	errg.Go(func() error {
		return apiServer.Run(ctx)
	})

	// Start the manager.
	errg.Go(func() error {
		return manager.Run(ctx)
	})

	// Start the settings refresher.
	errg.Go(func() error {
		return refreshSettings(ctx, cfg, manager, managerSettingsHash, apiServer, apiSettingsHash)
	})

	return errg.Wait()
}

// refreshSettings periodically checks for changes in the manager and API settings files.
// If changes are detected, it hot-reloads the settings into the running manager and API server.
func refreshSettings(ctx context.Context, cfg *config.Config, manager *managerpkg.Manager, managerSettingsHash string, apiServer *api.Server, apiSettingsHash string) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Second):
			managerSettings, hash, err := readManagerSettings(cfg.ManagerSettingsFilename)
			if err != nil {
				log.Error().Msgf("Failed to read manager settings: %s", err)
			} else if hash != managerSettingsHash {
				if err := manager.UpdateSettings(managerSettings); err != nil {
					log.Error().Msgf("Failed to update manager settings: %s", err)
				}

				managerSettingsHash = hash
			}

			apiSettings, hash, err := readAPISettings(cfg.APISettingsFilename)
			if err != nil {
				log.Error().Msgf("Failed to read API settings: %s", err)
			} else if hash != apiSettingsHash {
				if err := apiServer.UpdateSettings(apiSettings); err != nil {
					log.Error().Msgf("Failed to update API settings: %s", err)
				}

				apiSettingsHash = hash
			}
		}
	}
}

// handleSignals listens for OS signals and cancels the context when a termination signal is received.
func handleSignals(ctx context.Context, cancel context.CancelFunc) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)

	select {
	case <-ctx.Done():
	case sig := <-signals:
		log.Info().Msgf("Signal %s received", sig)
		cancel()
	}
}

// exitMain handles the application exit logic based on the provided error.
func exitMain(err error) {
	var returnCode int

	switch {
	case err == nil:
		returnCode = 0
	case errors.Is(err, context.Canceled):
		returnCode = 0
		log.Info().Msgf("Application canceled: %s", err)
	default:
		returnCode = 1
		log.Error().Msgf("Application failed: %s", err)
	}

	os.Exit(returnCode)
}
