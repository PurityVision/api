package src

import (
	"fmt"
	"os"
	"strconv"

	"github.com/rs/zerolog"
)

type Config struct {
	DBHost               string // DBHost is the host machine running the postgres instance.
	DBPort               string // DBPort is the port that exposes the db server.
	DBName               string // DBName is the postgres database name.
	DBUser               string // DBUser is the postgres user account.
	DBPassword           string // DBPassword is the password for the DBUser postgres account.
	DBSSLMode            string // DBSSLMode sets the SSL mode of the postgres client.
	LogLevel             string // LogLevel is the level of logging for the application.
	StripeKey            string // StripeKey is for making Stripe API requests.
	EmailName            string // Name on email license delivery.
	SendgridAPIKey       string // SendgridAPIKey is for sending emails.
	StripeWebhookSecret  string // Stripe webhook secret.
	EmailFrom            string // From address for email license delivery.
	TrialLicenseMaxUsage int    // TrialLicenseMaxUsage is the maximum image filters for a trial license.
}

func missingEnvErr(envVar string) error {
	return fmt.Errorf("%s not found in environment", envVar)
}

func newConfig() (Config, error) {
	var (
		StripeKey           = os.Getenv("STRIPE_KEY")
		StripeWebhookSecret = os.Getenv("STRIPE_WEBHOOK_SECRET")
		EmailName           = getEnvWithDefault("EMAIL_NAME", "John Doe")
		EmailFrom           = getEnvWithDefault("EMAIL_FROM", "test@example.com")
		SendgridAPIKey      = os.Getenv("SENDGRID_API_KEY")
	)

	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" {
		return Config{}, missingEnvErr("GOOGLE_APPLICATION_CREDENTIALS")
	}

	if StripeKey == "" {
		return Config{}, missingEnvErr("STRIPE_KEY")
	}

	if StripeWebhookSecret == "" {
		return Config{}, missingEnvErr("STRIPE_WEBHOOK_SECRET")
	}

	if EmailName == "" {
		return Config{}, missingEnvErr("EMAIL_NAME")
	}

	if EmailFrom == "" {
		return Config{}, missingEnvErr("EMAIL_FROM")
	}

	if SendgridAPIKey == "" {
		return Config{}, missingEnvErr("SENDGRID_API_KEY")
	}

	return Config{
		DBHost:              getEnvWithDefault("PURITY_DB_HOST", "localhost"),
		DBPort:              getEnvWithDefault("PURITY_DB_PORT", "5432"),
		DBName:              getEnvWithDefault("PURITY_DB_NAME", "purity"),
		DBUser:              getEnvWithDefault("PURITY_DB_USER", "postgres"),
		DBPassword:          getEnvWithDefault("PURITY_DB_PASS", ""),
		DBSSLMode:           getEnvWithDefault("PURITY_DB_SSL_MODE", "disable"),
		LogLevel:            getEnvWithDefault("PURITY_LOG_LEVEL", strconv.Itoa(int(zerolog.InfoLevel))),
		StripeKey:           StripeKey,
		StripeWebhookSecret: StripeWebhookSecret,
		EmailName:           EmailName,
		EmailFrom:           EmailFrom,
		SendgridAPIKey:      SendgridAPIKey,
	}, nil
}

func getEnvWithDefault(name string, def string) string {
	res, found := os.LookupEnv(name)
	if !found {
		return def
	}
	return res
}
