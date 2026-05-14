// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/telekom/sparrow/internal/logger"
	"github.com/telekom/sparrow/pkg/config"
	"github.com/telekom/sparrow/pkg/sparrow"
)

const (
	defaultLoaderHttpTimeout = 30 * time.Second
	defaultLoaderInterval    = 300 * time.Second
	defaultHttpRetryCount    = 3
	defaultHttpRetryDelay    = 1 * time.Second
)

// NewRunCmd creates a new run command
func NewRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run sparrow",
		Long:  `Sparrow will be started with the provided configuration`,
		RunE:  run,
	}

	NewFlag("api.address", "apiAddress").String().Bind(cmd, ":8080", "api: The address the server is listening on")
	NewFlag("name", "sparrowName").String().Bind(cmd, "", "The DNS name of the sparrow")
	NewFlag("loader.type", "loaderType").StringP("l").Bind(cmd, "http", "Defines the loader type that will load the checks configuration during the runtime. The fallback is the fileLoader")
	NewFlag("loader.interval", "loaderInterval").Duration().Bind(cmd, defaultLoaderInterval, "defines the interval the loader reloads the configuration in seconds")
	NewFlag("loader.http.url", "loaderHttpUrl").String().Bind(cmd, "", "http loader: The url where to get the remote configuration")
	NewFlag("loader.http.token", "loaderHttpToken").String().Bind(cmd, "", "http loader: Bearer token to authenticate the http endpoint")
	NewFlag("loader.http.timeout", "loaderHttpTimeout").Duration().Bind(cmd, defaultLoaderHttpTimeout, "http loader: The timeout for the http request in seconds")
	NewFlag("loader.http.retry.count", "loaderHttpRetryCount").Int().Bind(cmd, defaultHttpRetryCount, "http loader: Amount of retries trying to load the configuration")
	NewFlag("loader.http.retry.delay", "loaderHttpRetryDelay").Duration().Bind(cmd, defaultHttpRetryDelay, "http loader: The initial delay between retries in seconds")
	NewFlag("loader.file.path", "loaderFilePath").String().Bind(cmd, "config.yaml", "file loader: The path to the file to read the runtime config from")

	return cmd
}

// run is the entry point to start the sparrow
func run(cmd *cobra.Command, _ []string) error {
	ctx := logger.IntoContext(cmd.Context(), logger.NewLogger())
	cfg := &config.Config{}
	err := viper.Unmarshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	if err = cfg.Validate(ctx); err != nil {
		return fmt.Errorf("error while validating the config: %w", err)
	}

	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	log := logger.FromContext(ctx)

	s := sparrow.New(ctx, cfg)
	cErr := make(chan error, 1)
	log.InfoContext(ctx, "Running sparrow")
	go func() {
		cErr <- s.Run(ctx)
	}()

	select {
	case <-ctx.Done():
		log.InfoContext(ctx, "Context is done, shutting down", "reason", ctx.Err())
		<-cErr
	case err = <-cErr:
		log.InfoContext(ctx, "Sparrow was shut down", "error", err)
		return err
	}

	return nil
}
