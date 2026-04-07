package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	app "github.com/shurco/mycart/internal"
	"github.com/shurco/mycart/pkg/update"
)

var (
	version   = "v0.0.1"
	gitCommit = "00000000"
	buildDate = "14.07.2023"
)

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

	rootCmd.AddCommand(cmdInit())
	rootCmd.AddCommand(cmdServe())
	rootCmd.AddCommand(cmdUpdate())
	rootCmd.AddCommand(cmdMigrate())

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

// cmdServe creates and returns the serve command.
func cmdServe() *cobra.Command {
	var noSite, devMode bool
	var httpAddr, httpsAddr string

	cmd := &cobra.Command{
		Use:   "serve [flags]",
		Short: "Starts the web server (default to 0.0.0.0:8080)",
		Run: func(_ *cobra.Command, _ []string) {
			handleCommandError(app.NewApp(httpAddr, httpsAddr, noSite, devMode))
		},
	}

	cmd.PersistentFlags().StringVar(&httpAddr, "http", "0.0.0.0:8080", "server address")
	cmd.PersistentFlags().StringVar(&httpsAddr, "https", "", "https server address (auto TLS)")
	cmd.PersistentFlags().BoolVar(&noSite, "no-site", false, "disable create site")
	cmd.PersistentFlags().BoolVar(&devMode, "dev", false, "develop mode")

	if err := cmd.PersistentFlags().MarkHidden("dev"); err != nil {
		fmt.Println("warning: failed to hide dev flag:", err)
	}

	return cmd
}

// cmdInit creates and returns the init command.
func cmdInit() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Creating the basic structure",
		Run: func(_ *cobra.Command, _ []string) {
			handleCommandError(app.Init())
		},
	}
}

// cmdUpdate creates and returns the update command.
func cmdUpdate() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Updating the application to the latest version",
		Run: func(_ *cobra.Command, _ []string) {
			cfg := &update.Config{
				Owner:             "shurco",
				Repo:              "mycart",
				CurrentVersion:    version,
				ArchiveExecutable: "mycart",
			}

			if err := update.Init(cfg); err != nil {
				handleCommandError(err)
				return
			}

			handleCommandError(app.Migrate())
		},
	}
}

// cmdMigrate creates and returns the migrate command.
func cmdMigrate() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "Migrate on the latest version of database schema",
		Run: func(_ *cobra.Command, _ []string) {
			handleCommandError(app.Migrate())
		},
	}
}
