package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/rs/zerolog"
)

var (
	// DefaultDBName is the default name of the database.
	DefaultDBName = "purity"

	// DefaultDBTestName is the default name of the test database.
	DefaultDBTestName = "purity_test"

	// DefaultPort is the default port to expose the API server.
	DefaultPort int = 8080

	// DBHost is the host machine running the postgres instance.
	DBHost string

	// DBPort is the port that exposes the db server.
	DBPort string

	// DBName is the postgres database name.
	DBName string

	// DBUser is the postgres user account.
	DBUser string

	// DBPassword is the password for the DBUser postgres account.
	DBPassword string

	// DBSSLMode sets the SSL mode of the postgres client.
	DBSSLMode string

	// LogLevel is the level of logging for the application.
	LogLevel string

	// StripeKey is for making Stripe API requests.
	StripeKey string

	// Name on email license delivery.
	EmailName string

	// SendgridAPIKey is for sending emails.
	SendgridAPIKey string

	// Stripe webhook secret.
	StripeWebhookSecret string

	// From address for email license delivery.
	EmailFrom string
)

func Init() error {
	DefaultPort = 8080

	DBHost = getEnvWithDefault("PURITY_DB_HOST", "localhost")
	DBPort = getEnvWithDefault("PURITY_DB_PORT", "5432")
	DBName = getEnvWithDefault("PURITY_DB_NAME", DefaultDBName)
	DBUser = getEnvWithDefault("PURITY_DB_USER", "postgres")
	DBPassword = getEnvWithDefault("PURITY_DB_PASS", "")
	DBSSLMode = getEnvWithDefault("PURITY_DB_SSL_MODE", "disable")

	LogLevel = getEnvWithDefault("PURITY_LOG_LEVEL", strconv.Itoa(int(zerolog.InfoLevel)))

	missingEnvErr := func(envVar string) error {
		return fmt.Errorf("%s not found in environment", envVar)
	}

	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" {
		return missingEnvErr("GOOGLE_APPLICATION_CREDENTIALS")
	}

	if StripeKey = os.Getenv("STRIPE_KEY"); StripeKey == "" {
		return missingEnvErr("STRIPE_KEY")
	}

	if StripeWebhookSecret = os.Getenv("STRIPE_WEBHOOK_SECRET"); StripeWebhookSecret == "" {
		return missingEnvErr("STRIPE_WEBHOOK_SECRET")
	}

	if EmailName = getEnvWithDefault("EMAIL_NAME", "John Doe"); EmailName == "" {
		return missingEnvErr("EMAIL_NAME")
	}

	if EmailFrom = getEnvWithDefault("EMAIL_FROM", "test@example.com"); EmailFrom == "" {
		return missingEnvErr("EMAIL_FROM")
	}

	if SendgridAPIKey = os.Getenv("SENDGRID_API_KEY"); SendgridAPIKey == "" {
		return missingEnvErr("SENDGRID_API_KEY")
	}

	return nil
}

func getEnvWithDefault(name string, def string) string {
	res, found := os.LookupEnv(name)
	if !found {
		return def
	}
	return res
}
