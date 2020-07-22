package cmd

import (
	"fmt"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	defaultLogLevel     = "info"
	defaultOutputFormat = "stdout"
)

var (
	cfgFile      string
	outputFormat string
	logLevel     string
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:               "rgctl",
	Short:             "Cloud procedures made easy",
	Long:              `rgctl is a cli application that executes procedures`,
	PersistentPreRunE: setupLogging,
	TraverseChildren:  true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {

	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().StringVarP(&logLevel, "verbosity", "v", defaultLogLevel, "Log level (debug, info, warn, error, fatal, panic")
	RootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", defaultOutputFormat, "Output format(stdout, json)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".rgctl" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".rgctl")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func setupLogging(cmd *cobra.Command, args []string) error {
	log.SetOutput(os.Stdout)
	if outputFormat == "json" {
		log.SetFormatter(&log.JSONFormatter{})
	} else {
		log.SetFormatter(&log.TextFormatter{})
	}

	lvl, err := log.ParseLevel(logLevel)
	if err != nil {
		return err
	}
	log.SetLevel(lvl)
	return nil
}
