package cmd

import (
	"context"
	"log"

	"github.com/caas-team/sparrow/pkg/config"
	"github.com/caas-team/sparrow/pkg/sparrow"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewCmdRun creates a new run command
func NewCmdRun() *cobra.Command {
	flagMapping := config.RunFlagsNameMapping{
		LoaderType:           "loaderType",
		LoaderInterval:       "loaderInterval",
		LoaderHttpUrl:        "loaderHttpUrl",
		LoaderHttpToken:      "loaderHttpToken",
		LoaderHttpTimeout:    "loaderHttpTimeout",
		LoaderHttpRetryCount: "loaderHttpRetryCount",
		LoaderHttpRetryDelay: "loaderHttpRetryDelay",
	}

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run sparrow",
		Long:  `Sparrow will be started with the provided configuration`,
		Run:   run(&flagMapping),
	}

	cmd.PersistentFlags().StringP(flagMapping.LoaderType, "l", "http", "defines the loader type that will load the checks configuration during the runtime")
	cmd.PersistentFlags().Int(flagMapping.LoaderInterval, 300, "defines the interval the loader reloads the configuration in seconds")
	cmd.PersistentFlags().String(flagMapping.LoaderHttpUrl, "", "http loader: The url where to get the remote configuration")
	cmd.PersistentFlags().String(flagMapping.LoaderHttpToken, "", "http loader: Bearer token to authenticate the http endpoint")
	cmd.PersistentFlags().Int(flagMapping.LoaderHttpTimeout, 30, "http loader: The timeout for the http request in seconds")
	cmd.PersistentFlags().Int(flagMapping.LoaderHttpRetryCount, 3, "http loader: Amount of retries trying to load the configuration")
	cmd.PersistentFlags().Int(flagMapping.LoaderHttpRetryDelay, 1, "http loader: The initial delay between retries in seconds")

	viper.BindPFlag(flagMapping.LoaderType, cmd.PersistentFlags().Lookup(flagMapping.LoaderType))
	viper.BindPFlag(flagMapping.LoaderInterval, cmd.PersistentFlags().Lookup(flagMapping.LoaderInterval))
	viper.BindPFlag(flagMapping.LoaderHttpUrl, cmd.PersistentFlags().Lookup(flagMapping.LoaderHttpUrl))
	viper.BindPFlag(flagMapping.LoaderHttpToken, cmd.PersistentFlags().Lookup(flagMapping.LoaderHttpToken))
	viper.BindPFlag(flagMapping.LoaderHttpTimeout, cmd.PersistentFlags().Lookup(flagMapping.LoaderHttpTimeout))
	viper.BindPFlag(flagMapping.LoaderHttpRetryCount, cmd.PersistentFlags().Lookup(flagMapping.LoaderHttpRetryCount))
	viper.BindPFlag(flagMapping.LoaderHttpRetryDelay, cmd.PersistentFlags().Lookup(flagMapping.LoaderHttpRetryDelay))

	return cmd
}

// run is the entry point to start the sparrow
func run(fm *config.RunFlagsNameMapping) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		cfg := config.NewConfig()

		cfg.SetLoaderType(viper.GetString(fm.LoaderType))
		cfg.SetLoaderInterval(viper.GetInt(fm.LoaderInterval))
		cfg.SetLoaderHttpUrl(viper.GetString(fm.LoaderHttpUrl))
		cfg.SetLoaderHttpToken(viper.GetString(fm.LoaderHttpToken))
		cfg.SetLoaderHttpTimeout(viper.GetInt(fm.LoaderHttpTimeout))
		cfg.SetLoaderHttpRetryCount(viper.GetInt(fm.LoaderHttpRetryCount))
		cfg.SetLoaderHttpRetryDelay(viper.GetInt(fm.LoaderHttpRetryDelay))

		if err := cfg.Validate(fm); err != nil {
			log.Panic(err)
		}

		sparrow := sparrow.New(cfg)

		log.Println("running sparrow")
		if err := sparrow.Run(context.Background()); err != nil {
			log.Panic(err)
		}
	}
}
