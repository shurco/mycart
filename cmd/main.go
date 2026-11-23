// @title           myCart API
// @version         1.0
// @description     Open source shopping-cart backend API - a single-binary e-commerce solution
// @termsOfService  https://github.com/shurco/mycart

// @contact.name   API Support
// @contact.url    https://github.com/shurco/mycart/issues
// @contact.email  support@mycart.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/shurco/mycart/internal/database"
	"github.com/shurco/mycart/pkg/update"

	_ "github.com/shurco/mycart/docs/swagger"
)

var (
	version   = "v0.0.1"
	gitCommit = "00000000"
	buildDate = "14.07.2023"
)

// dbOverrides carries the persistent --db-* flags. Values left empty fall
// through to the environment, lc_base/config.json and finally the built-in
// SQLite default.
var dbOverrides database.Overrides

// resolveDB resolves the database configuration for a command. The resolution
// order is: flags, environment, lc_base/config.json, SQLite default.
func resolveDB() database.Config {
	cfg, err := database.Resolve(dbOverrides)
	if err != nil {
		handleCommandError(err)
	}
	return cfg
}

var rootCmd = &cobra.Command{
	Use:                "mycart",
	Short:              "myCart CLI",
	Long:               "🛒 myCart - shopping-cart in 1 file",
	Version:            fmt.Sprintf("myCart %s (%s) from %s", version, gitCommit, buildDate),
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	CompletionOptions:  cobra.CompletionOptions{DisableDefaultCmd: true},
}

func main() {
	update.SetVersion(&update.Version{
		CurrentVersion: version,
		GitCommit:      gitCommit,
		BuildDate:      buildDate,
	})

	rootCmd.SetHelpCommand(&cobra.Command{
		Use:    "no-help",
		Hidden: true,
	})

	rootCmd.PersistentFlags().StringVar(&dbOverrides.Driver, "db", "",
		`database: "sqlite" (default) or "postgres"`)
	rootCmd.PersistentFlags().StringVar(&dbOverrides.DSN, "db-dsn", "",
		`database connection string: a PostgreSQL DSN, or a SQLite file path`)

	rootCmd.AddCommand(cmdInit())
	rootCmd.AddCommand(cmdInstall())
	rootCmd.AddCommand(cmdServe())
	rootCmd.AddCommand(cmdUpdate())
	rootCmd.AddCommand(cmdMigrate())
	rootCmd.AddCommand(cmdDB())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// handleCommandError handles command execution errors uniformly.
func handleCommandError(err error) {
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
}
