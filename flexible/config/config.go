package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	LogJSON                 bool
	Verbose                 bool
	Location                string
	RestPort                string
	EncryptionKey           string
	SkipDatabaseCache       bool
	EnableMetrics           bool
	MetricsPort             string
	QueueMode               string
	DatabaseMode            string
	ManagerSettingsFilename string
	APISettingsFilename     string
	Aws                     AwsConfig
	Postgres                PostgresConfig
	Slack                   SlackConfig
	Redis                   RedisConfig
}

type AwsConfig struct {
	Region               string
	Key                  string
	SecretKey            string
	SessionToken         string
	AssumeRole           string
	MaxRetryAttempts     int
	MaxRetryBackoffDelay time.Duration
	Concurrency          int64
	SqsEndpoint          string
	AlertQueue           SqsQueueConfig
	CommandQueue         SqsQueueConfig
	DynamoDB             DynamoDBConfig
}

type SqsQueueConfig struct {
	QueueName                string
	VisibilityTimeoutSeconds int32
	MaxNumberOfMessages      int32
	WaitTimeSeconds          int32
}

type DynamoDBConfig struct {
	TableName string
}

type PostgresConfig struct {
	Host                        string
	Port                        int
	User                        string
	Password                    string
	Database                    string
	SSLMode                     string
	IssuesTable                 string
	AlertsTable                 string
	MoveMappingsTable           string
	ChannelProcessingStateTable string
}

type SlackConfig struct {
	AppToken string
	BotToken string
}

type RedisConfig struct {
	Addr     string
	Username string
	Password string
	DB       int
}

func New() *Config {
	return &Config{
		LogJSON:                 GetEnvBoolIfSet("LOG_JSON", true),
		Verbose:                 GetEnvBoolIfSet("VERBOSE", false),
		Location:                GetEnvIfSet("LOCATION", "Europe/Oslo"),
		RestPort:                GetEnvIfSet("REST_PORT", "8080"),
		EncryptionKey:           GetEnvIfSet("ENCRYPTION_KEY", ""),
		SkipDatabaseCache:       GetEnvBoolIfSet("SKIP_DATABASE_CACHE", false),
		EnableMetrics:           GetEnvBoolIfSet("ENABLE_METRICS", true),
		MetricsPort:             GetEnvIfSet("METRICS_PORT", "9090"),
		QueueMode:               GetEnvIfSet("QUEUE_MODE", "redis"),
		DatabaseMode:            GetEnvIfSet("DATABASE_MODE", "postgres"),
		ManagerSettingsFilename: GetEnvIfSet("MANAGER_SETTINGS_FILENAME", "manager-settings.yaml"),
		APISettingsFilename:     GetEnvIfSet("API_SETTINGS_FILENAME", "api-settings.yaml"),
		Aws: AwsConfig{
			Region:               GetEnvIfSet("AWS_REGION", ""),
			Key:                  GetEnvIfSet("AWS_ACCESS_KEY_ID", ""),
			SecretKey:            GetEnvIfSet("AWS_SECRET_ACCESS_KEY", ""),
			SessionToken:         GetEnvIfSet("AWS_SESSION_TOKEN", ""),
			AssumeRole:           GetEnvIfSet("AWS_ASSUME_ROLE", ""),
			MaxRetryAttempts:     GetEnvIntIfSet("AWS_MAX_RETRY_ATTEMPTS", 5),
			MaxRetryBackoffDelay: GetEnvSecondsIfSet("AWS_MAX_RETRY_BACKOFF_DELAY", 30),
			Concurrency:          int64(GetEnvIntIfSet("AWS_CONCURRENCY", 10)),
			SqsEndpoint:          GetEnvIfSet("AWS_SQS_ENDPOINT", ""),
			AlertQueue: SqsQueueConfig{
				QueueName:                GetEnvIfSet("AWS_SQS_ALERT_QUEUE_NAME", ""),
				VisibilityTimeoutSeconds: int32(GetEnvIntIfSet("AWS_SQS_ALERT_QUEUE_VISIBILITY_TIMEOUT_SECONDS", 30)),
				MaxNumberOfMessages:      int32(GetEnvIntIfSet("AWS_SQS_ALERT_QUEUE_MAX_NUMBER_OF_MESSAGES", 10)),
				WaitTimeSeconds:          int32(GetEnvIntIfSet("AWS_SQS_ALERT_QUEUE_WAIT_TIME_SECONDS", 20)),
			},
			CommandQueue: SqsQueueConfig{
				QueueName:                GetEnvIfSet("AWS_SQS_COMMAND_QUEUE_NAME", ""),
				VisibilityTimeoutSeconds: int32(GetEnvIntIfSet("AWS_SQS_COMMAND_QUEUE_VISIBILITY_TIMEOUT_SECONDS", 30)),
				MaxNumberOfMessages:      int32(GetEnvIntIfSet("AWS_SQS_COMMAND_QUEUE_MAX_NUMBER_OF_MESSAGES", 10)),
				WaitTimeSeconds:          int32(GetEnvIntIfSet("AWS_SQS_COMMAND_QUEUE_WAIT_TIME_SECONDS", 20)),
			},
			DynamoDB: DynamoDBConfig{
				TableName: GetEnvIfSet("AWS_DYNAMODB_TABLE_NAME", ""),
			},
		},
		Postgres: PostgresConfig{
			Host:                        GetEnvIfSet("POSTGRES_HOST", ""),
			Port:                        GetEnvIntIfSet("POSTGRES_PORT", 0),
			User:                        GetEnvIfSet("POSTGRES_USER", ""),
			Password:                    GetEnvIfSet("POSTGRES_PASSWORD", ""),
			Database:                    GetEnvIfSet("POSTGRES_DATABASE", ""),
			SSLMode:                     GetEnvIfSet("POSTGRES_SSL_MODE", "disable"),
			IssuesTable:                 GetEnvIfSet("POSTGRES_ISSUES_TABLE", "issues"),
			AlertsTable:                 GetEnvIfSet("POSTGRES_ALERTS_TABLE", "alerts"),
			MoveMappingsTable:           GetEnvIfSet("POSTGRES_MOVE_MAPPINGS_TABLE", "move_mappings"),
			ChannelProcessingStateTable: GetEnvIfSet("POSTGRES_CHANNEL_PROCESSING_STATE_TABLE", "channel_processing_state"),
		},
		Slack: SlackConfig{
			AppToken: GetEnvIfSet("SLACK_APP_TOKEN", ""),
			BotToken: GetEnvIfSet("SLACK_BOT_TOKEN", ""),
		},
		Redis: RedisConfig{
			Addr:     GetEnvIfSet("REDIS_ADDR", ""),
			Password: GetEnvIfSet("REDIS_PASSWORD", ""),
			Username: GetEnvIfSet("REDIS_USERNAME", ""),
			DB:       GetEnvIntIfSet("REDIS_DB", 0),
		},
	}
}

func GetEnvIfSet(envVar, defaultValue string) string {
	if val, ok := os.LookupEnv(envVar); ok {
		return val
	}

	return defaultValue
}

func GetEnvIntIfSet(envVar string, defaultValue int) int {
	str := os.Getenv(envVar)

	if str != "" {
		val, err := strconv.Atoi(str)
		if err != nil {
			panic(fmt.Sprintf("failed to parse environment variable %s as int: %v", envVar, err))
		}

		return val
	}

	return defaultValue
}

func GetEnvSecondsIfSet(envVar string, defaultValue int) time.Duration {
	val := GetEnvIntIfSet(envVar, defaultValue)

	return time.Duration(val) * time.Second
}

func GetEnvBoolIfSet(envVar string, defaultValue bool) bool {
	str := os.Getenv(envVar)

	if str != "" {
		val, err := strconv.ParseBool(str)
		if err != nil {
			panic(fmt.Sprintf("failed to parse environment variable %s as bool: %v", envVar, err))
		}

		return val
	}

	return defaultValue
}
