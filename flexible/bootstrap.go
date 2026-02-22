//nolint:ireturn
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/eko/gocache/lib/v4/store"
	redis_store "github.com/eko/gocache/store/rediscluster/v4"
	redis "github.com/redis/go-redis/v9"
	managerconfig "github.com/slackmgr/core/config"
	manager "github.com/slackmgr/core/manager"
	"github.com/slackmgr/examples/flexible/config"
	dynamodb "github.com/slackmgr/plugins/dynamodb"
	postgres "github.com/slackmgr/plugins/postgres"
	sqs "github.com/slackmgr/plugins/sqs"
	"github.com/slackmgr/types"
	"gopkg.in/yaml.v2"
)

// newRedisClient creates a new Redis client based on the provided configuration.
// In this case, we create a very basic Redis client. For more complex setups (e.g., clusters, sentinel),
// additional logic would be needed.
func newRedisClient(cfg *config.RedisConfig) (*redis.Client, error) {
	if cfg.Addr == "" {
		return nil, errors.New("redis address is empty")
	}

	options := &redis.Options{
		Addr:     cfg.Addr,
		Username: cfg.Username,
		Password: cfg.Password,
		DB:       cfg.DB,
	}

	return redis.NewClient(options), nil
}

// newCacheStore creates a new cache store using the provided Redis client.
// We accept a redis.UniversalClient to allow for more flexibility (e.g., cluster, sentinel).
func newCacheStore(client redis.UniversalClient) store.StoreInterface {
	return redis_store.NewRedisCluster(client)
}

// newAlertQueue creates a new alert queue based on the provided configuration.
// It supports SQS, Redis, and in-memory queue modes, depending on the QueueMode setting in the config.
func newAlertQueue(ctx context.Context, redisClient redis.UniversalClient, channelLocker manager.ChannelLocker, cfg *config.Config, logger *Logger) (manager.FifoQueue, error) {
	switch strings.ToLower(cfg.QueueMode) {
	case "sqs":
		return newSQSClient(ctx, &cfg.Aws, &cfg.Aws.AlertQueue, logger)
	case "redis":
		return manager.NewRedisFifoQueue(redisClient, channelLocker, "alerts", logger).Init()
	case "in-memory":
		return types.NewInMemoryFifoQueue("alerts", 1000, 5*time.Second), nil
	default:
		return nil, fmt.Errorf("unknown queue mode: %s", cfg.QueueMode)
	}
}

// newCommandQueue creates a new command queue based on the provided configuration.
// It supports SQS, Redis, and in-memory queue modes, depending on the QueueMode setting in the config.
func newCommandQueue(ctx context.Context, redisClient redis.UniversalClient, channelLocker manager.ChannelLocker, cfg *config.Config, logger *Logger) (manager.FifoQueue, error) {
	switch strings.ToLower(cfg.QueueMode) {
	case "sqs":
		return newSQSClient(ctx, &cfg.Aws, &cfg.Aws.CommandQueue, logger)
	case "redis":
		return manager.NewRedisFifoQueue(redisClient, channelLocker, "commands", logger).Init()
	case "in-memory":
		return types.NewInMemoryFifoQueue("commands", 1000, 5*time.Second), nil
	case "":
		return nil, errors.New("queue mode is not set (QUEUE_MODE=<mode>)")
	default:
		return nil, fmt.Errorf("unknown queue mode: %s", cfg.QueueMode)
	}
}

// newSQSClient creates a new SQS client based on the provided AWS and SQS queue configuration.
// Only relevant if SQS is used as the queue mode.
func newSQSClient(ctx context.Context, cfg *config.AwsConfig, queueCfg *config.SqsQueueConfig, logger *Logger) (*sqs.Client, error) {
	awsCfg, err := createAwsCfg(ctx, cfg, logger)
	if err != nil {
		return nil, err
	}

	opts := []sqs.Option{
		sqs.WithSqsVisibilityTimeout(queueCfg.VisibilityTimeoutSeconds),
		sqs.WithSqsReceiveMaxNumberOfMessages(queueCfg.MaxNumberOfMessages),
		sqs.WithSqsReceiveWaitTimeSeconds(queueCfg.WaitTimeSeconds),
		sqs.WithSqsAPIMaxRetryAttempts(cfg.MaxRetryAttempts),
		sqs.WithSqsAPIMaxRetryBackoffDelay(cfg.MaxRetryBackoffDelay),
	}

	return sqs.New(awsCfg, queueCfg.QueueName, logger, opts...).Init(ctx)
}

// newDatabase creates a new database client based on the provided configuration.
// It supports DynamoDB and Postgres, depending on the DatabaseMode setting in the config.
func newDatabase(ctx context.Context, cfg *config.Config, logger *Logger) (types.DB, error) {
	switch strings.ToLower(cfg.DatabaseMode) {
	case "dynamodb":
		return newDynamoDBClient(ctx, &cfg.Aws, logger)
	case "postgres":
		return newPostgresClient(ctx, &cfg.Postgres, logger)
	case "":
		return nil, errors.New("database mode is not set (DATABASE_MODE=<mode>)")
	default:
		return nil, fmt.Errorf("unknown database mode: %s", cfg.DatabaseMode)
	}
}

// newPostgresClient creates a new Postgres client based on the provided Postgres configuration.
// Only relevant if Postgres is used as the database.
func newPostgresClient(ctx context.Context, cfg *config.PostgresConfig, logger *Logger) (*postgres.Client, error) {
	if cfg.Host == "" {
		return nil, errors.New("postgres host is empty")
	}

	opts := []postgres.Option{
		postgres.WithHost(cfg.Host),
		postgres.WithPort(cfg.Port),
		postgres.WithUser(cfg.User),
		postgres.WithPassword(cfg.Password),
		postgres.WithDatabase(cfg.Database),
		postgres.WithSSLMode(postgres.SSLMode(cfg.SSLMode)),
		postgres.WithIssuesTable(cfg.IssuesTable),
		postgres.WithAlertsTable(cfg.AlertsTable),
		postgres.WithMoveMappingsTable(cfg.MoveMappingsTable),
		postgres.WithChannelProcessingStateTable(cfg.ChannelProcessingStateTable),
	}

	client := postgres.New(opts...)

	if err := client.Connect(ctx); err != nil {
		return nil, err
	}

	logger.Infof("Connected to Postgres on %s:%d", cfg.Host, cfg.Port)

	if err := client.Init(ctx, false); err != nil {
		return nil, err
	}

	logger.Infof("Initialized Postgres database %s", cfg.Database)

	return client, nil
}

// newDynamoDBClient creates a new DynamoDB client based on the provided AWS configuration.
// Only relevant if DynamoDB is used as the database.
func newDynamoDBClient(ctx context.Context, cfg *config.AwsConfig, logger *Logger) (*dynamodb.Client, error) {
	awsCfg, err := createAwsCfg(ctx, cfg, logger)
	if err != nil {
		return nil, err
	}

	client := dynamodb.New(awsCfg, cfg.DynamoDB.TableName)

	if err := client.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to DynamoDB: %w", err)
	}

	logger.Infof("Connected to DynamoDB table %s", cfg.DynamoDB.TableName)

	if err := client.Init(ctx, false); err != nil {
		return nil, fmt.Errorf("failed to initialize DynamoDB client: %w", err)
	}

	logger.Infof("Initialized DynamoDB client for table %s", cfg.DynamoDB.TableName)

	return client, nil
}

// createAwsCfg creates an AWS configuration based.
// It handles static credentials, assumed roles, and default credentials.
// Only relevant if AWS services (e.g., SQS, DynamoDB) are used.
func createAwsCfg(ctx context.Context, c *config.AwsConfig, logger *Logger) (*aws.Config, error) {
	if c.Region == "" {
		return &aws.Config{}, errors.New("cannot create AWS config with empty region")
	}

	cfg, err := awscfg.LoadDefaultConfig(ctx, awscfg.WithRegion(c.Region))
	if err != nil {
		return &aws.Config{}, err
	}

	// If we don't have a key or a role, assume we're running on EC2 and use the instance metadata service to get credentials
	if c.Key == "" && c.AssumeRole == "" {
		logger.Info("Using default AWS credentials")

		return &cfg, nil
	}

	// If we have a key, use static credentials
	if c.Key != "" {
		cfg.Credentials = aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(c.Key, c.SecretKey, c.SessionToken))
		logger.Infof("Using static AWS credentials with key %s", c.Key)
	}

	// If we have a role, assume it (possibly using the static credentials from the previous step)
	if c.AssumeRole != "" {
		credCacheOpts := func(opt *aws.CredentialsCacheOptions) {
			opt.ExpiryWindow = time.Minute
			opt.ExpiryWindowJitterFrac = 0.5
		}
		cfg.Credentials = aws.NewCredentialsCache(stscreds.NewAssumeRoleProvider(sts.NewFromConfig(cfg), c.AssumeRole), credCacheOpts)
		logger.Infof("Using assumed AWS credentials with role %s", c.AssumeRole)
	}

	return &cfg, nil
}

// readManagerSettings reads and unmarshals the manager settings from the specified yaml file.
// It also returns a hash of the settings for change detection, for hot-reloading purposes.
func readManagerSettings(filename string) (*managerconfig.ManagerSettings, string, error) {
	settingsYaml, err := os.ReadFile(filepath.Clean(filename))
	if err != nil {
		return nil, "", fmt.Errorf("failed to read manager settings file %s: %w", filename, err)
	}

	var settings managerconfig.ManagerSettings

	if err := yaml.Unmarshal(settingsYaml, &settings); err != nil {
		return nil, "", fmt.Errorf("failed to unmarshal manager settings from %s: %w", filename, err)
	}

	return &settings, hash(settingsYaml), nil
}

// readAPISettings reads and unmarshals the API settings from the specified yaml file.
// It also returns a hash of the settings for change detection, for hot-reloading purposes.
func readAPISettings(filename string) (*managerconfig.APISettings, string, error) {
	settingsYaml, err := os.ReadFile(filepath.Clean(filename))
	if err != nil {
		return nil, "", fmt.Errorf("failed to read API settings file %s: %w", filename, err)
	}

	var settings managerconfig.APISettings

	if err := yaml.Unmarshal(settingsYaml, &settings); err != nil {
		return nil, "", fmt.Errorf("failed to unmarshal API settings from %s: %w", filename, err)
	}

	return &settings, hash(settingsYaml), nil
}

// getLocation loads and returns a time.Location based on the provided configuration.
func getLocation(cfg *config.Config) *time.Location {
	loc, err := time.LoadLocation(cfg.Location)
	if err != nil {
		panic(fmt.Errorf("failed to load location %s: %w", cfg.Location, err))
	}
	return loc
}

func hash(input []byte) string {
	h := sha256.New()
	h.Write(input)
	bs := h.Sum(nil)
	return hex.EncodeToString(bs)
}
