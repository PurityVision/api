package config

import (
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

	StripeKey string

	// Name on email license delivery
	EmailName string

	// From address for email license delivery
	EmailFrom string
)

func Init() {
	// DefaultPort is the default port to expose the API server.
	DefaultPort = 8080

	// DBHost is the host machine running the postgres instance.
	DBHost = getEnvWithDefault("PURITY_DB_HOST", "localhost")

	// DBPort is the port that exposes the db server.
	DBPort = getEnvWithDefault("PURITY_DB_PORT", "5432")

	// DBName is the postgres database name.
	DBName = getEnvWithDefault("PURITY_DB_NAME", DefaultDBName)

	// DBUser is the postgres user account.
	DBUser = getEnvWithDefault("PURITY_DB_USER", "postgres")

	// DBPassword is the password for the DBUser postgres account.
	DBPassword = getEnvWithDefault("PURITY_DB_PASS", "")

	// DBSSLMode sets the SSL mode of the postgres client.
	DBSSLMode = getEnvWithDefault("PURITY_DB_SSL_MODE", "disable")

	// LogLevel is the level of logging for the application.
	LogLevel = getEnvWithDefault("PURITY_LOG_LEVEL", strconv.Itoa(int(zerolog.InfoLevel)))

	StripeKey = os.Getenv("STRIPE_KEY")

	// Name on email license delivery
	EmailName = getEnvWithDefault("EMAIL_NAME", "John Doe")

	// From address for email license delivery
	EmailFrom = getEnvWithDefault("EMAIL_FROM", "test@example.com")

}

func getEnvWithDefault(name string, def string) string {
	res, found := os.LookupEnv(name)
	if !found {
		return def
	}
	return res
}
