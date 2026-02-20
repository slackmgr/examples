package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
	managerconfig "github.com/slackmgr/core/config"
	managerpkg "github.com/slackmgr/core/manager"
	api "github.com/slackmgr/core/restapi"
	"github.com/slackmgr/types"
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

	logger := newLogger()

	// Create an in-memory alert queue. Do not use this in production!
	alertQueue := types.NewInMemoryFifoQueue("alerts", 1000, 5*time.Second)

	// Create an in-memory command queue. Do not use this in production!
	commandQueue := types.NewInMemoryFifoQueue("commands", 1000, 5*time.Second)

	// Create an in-memory database. Do not use this in production!
	db := types.NewInMemoryDB()

	// Create a no-op channel locker. This in fine when testing, and when running in a non-distributed environment.
	// In a distributed environment, such as k8s, a proper channel locker MUST be used.
	channelLocker := &managerpkg.NoopChannelLocker{}

	// Create a minimal manager config.
	managerCfg := managerconfig.NewDefaultManagerConfig()
	managerCfg.SlackClient.BotToken = os.Getenv("SLACK_BOT_TOKEN")
	managerCfg.SlackClient.AppToken = os.Getenv("SLACK_APP_TOKEN")

	// Create a minimal API config.
	apiCfg := managerconfig.NewDefaultAPIConfig()
	apiCfg.SlackClient.BotToken = os.Getenv("SLACK_BOT_TOKEN")
	apiCfg.SlackClient.AppToken = os.Getenv("SLACK_APP_TOKEN")

	// Minimal API settings. We need at least one routing rule if we want to accept alerts with route keys.
	// In this case, we use a single fallback rule that will match any route key. The Slack channel ID is read from the environment.
	apiSettings := &managerconfig.APISettings{
		RoutingRules: []*managerconfig.RoutingRule{
			{
				Name:        "Default rule",
				Description: "Fallback rule for every kind of alert",
				MatchAll:    true,
				Channel:     os.Getenv("ALERT_CHANNEL_ID"),
			},
		},
	}

	// Create the manager instance. This is the main application component, which handles alert processing.
	// We set the cache store and metrics to nil. The manager will use default in-memory implementations.
	// The manager settings are also nil, which means the manager will use default values for everything.
	manager := managerpkg.New(db, alertQueue, commandQueue, nil, channelLocker, logger, nil, managerCfg, nil)

	// Create the API server instance. This provides the REST API, where clients send alerts.
	// We set the cache store and metrics to nil. The api will use default in-memory implementations.
	apiServer := api.New(alertQueue, nil, logger, nil, apiCfg, apiSettings)

	// Start the manager and API server in separate goroutines.
	errg, ctx := errgroup.WithContext(ctx)

	// Start the API server.
	errg.Go(func() error {
		return apiServer.Run(ctx)
	})

	// Start the manager.
	errg.Go(func() error {
		return manager.Run(ctx)
	})

	return errg.Wait()
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
