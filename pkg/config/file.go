// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/caas-team/sparrow/internal/logger"
	"github.com/caas-team/sparrow/pkg/checks/runtime"
	"gopkg.in/yaml.v3"
)

var _ Loader = (*FileLoader)(nil)

type FileLoader struct {
	config   LoaderConfig
	cRuntime chan<- runtime.Config
	done     chan struct{}
	fsys     fs.FS
}

func NewFileLoader(cfg *Config, cRuntime chan<- runtime.Config) *FileLoader {
	return &FileLoader{
		config:   cfg.Loader,
		cRuntime: cRuntime,
		done:     make(chan struct{}, 1),
		fsys:     os.DirFS(filepath.Dir(cfg.Loader.File.Path)),
	}
}

// Run gets the runtime configuration from the local file.
// The config will be loaded periodically defined by the loader interval configuration.
// If the interval is 0, the configuration is only fetched once and the loader is disabled.
func (f *FileLoader) Run(ctx context.Context) error {
	ctx, cancel := logger.NewContextWithLogger(ctx)
	defer cancel()
	log := logger.FromContext(ctx)

	// Get the runtime configuration once on startup
	cfg, err := f.getRuntimeConfig(ctx)
	if err != nil {
		log.Warn("Could not get local runtime configuration", "error", err)
		err = fmt.Errorf("could not get local runtime configuration: %w", err)
	}
	f.cRuntime <- cfg

	if f.config.Interval == 0 {
		log.Info("File Loader disabled")
		return err
	}

	tick := time.NewTicker(f.config.Interval)
	defer tick.Stop()

	for {
		select {
		case <-f.done:
			log.Info("File Loader terminated")
			return nil
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
			runtimeCfg, err := f.getRuntimeConfig(ctx)
			if err != nil {
				log.Warn("Could not get local runtime configuration", "error", err)
				tick.Reset(f.config.Interval)
				continue
			}

			log.Info("Successfully got local runtime configuration")
			f.cRuntime <- runtimeCfg
			tick.Reset(f.config.Interval)
		}
	}
}

// getRuntimeConfig gets the local runtime configuration from the specified file.
func (f *FileLoader) getRuntimeConfig(ctx context.Context) (cfg runtime.Config, err error) {
	log := logger.FromContext(ctx).With("path", f.config.File.Path)

	file, err := f.fsys.Open(filepath.Base(f.config.File.Path))
	if err != nil {
		log.Error("Failed to open config file", "error", err)
		return cfg, fmt.Errorf("failed to open config file: %w", err)
	}
	defer func() {
		cerr := file.Close()
		if cerr != nil {
			log.Error("Failed to close config file", "error", cerr)
		}
		err = errors.Join(cerr, err)
	}()

	b, err := io.ReadAll(file)
	if err != nil {
		log.Error("Failed to read config file", "error", err)
		return cfg, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(b, &cfg); err != nil {
		log.Error("Failed to parse config file", "error", err)
		return cfg, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}

func (f *FileLoader) Shutdown(ctx context.Context) {
	log := logger.FromContext(ctx)
	select {
	case f.done <- struct{}{}:
		log.Debug("Sending signal to shut down file loader")
	default:
	}
}
