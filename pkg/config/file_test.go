// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"io/fs"
	"reflect"
	"testing"
	"time"

	"github.com/telekom/sparrow/pkg/checks/health"
	"github.com/telekom/sparrow/pkg/checks/runtime"
	"github.com/telekom/sparrow/pkg/config/test"
	"gopkg.in/yaml.v3"
)

func TestNewFileLoader(t *testing.T) {
	l := NewFileLoader(&Config{Loader: LoaderConfig{File: FileLoaderConfig{Path: "config.yaml"}}}, make(chan runtime.Config, 1))

	if l.config.File.Path != "config.yaml" {
		t.Errorf("Expected path to be config.yaml, got %s", l.config.File.Path)
	}
	if l.cRuntime == nil {
		t.Errorf("Expected channel to be not nil")
	}
	if l.fsys == nil {
		t.Errorf("Expected filesystem to be not nil")
	}
}

func TestFileLoader_Run(t *testing.T) {
	tests := []struct {
		name    string
		config  LoaderConfig
		want    runtime.Config
		wantErr bool
	}{
		{
			name: "Loads config from file",
			config: LoaderConfig{
				Type:     "file",
				Interval: 1 * time.Second,
				File: FileLoaderConfig{
					Path: "test/data/config.yaml",
				},
			},
			want: runtime.Config{
				Health: &health.Config{
					Targets:  []string{"http://localhost:8080/health"},
					Interval: 1 * time.Second,
					Timeout:  1 * time.Second,
				},
			},
			wantErr: false,
		},
		{
			name: "Continuous loading disabled",
			config: LoaderConfig{
				Type:     "file",
				Interval: 0,
				File: FileLoaderConfig{
					Path: "test/data/config.yaml",
				},
			},
			want: runtime.Config{
				Health: &health.Config{
					Targets:  []string{"http://localhost:8080/health"},
					Interval: 1 * time.Second,
					Timeout:  1 * time.Second,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			result := make(chan runtime.Config, 1)
			f := NewFileLoader(&Config{
				Loader: tt.config,
			}, result)

			go func(wantErr bool) {
				defer close(result)
				err := f.Run(ctx)
				if (err != nil) != wantErr {
					t.Errorf("Run() error %v, want %v", err, tt.wantErr)
				}
			}(tt.wantErr)
			defer f.Shutdown(ctx)

			if !tt.wantErr {
				config := <-result
				if !reflect.DeepEqual(config, tt.want) {
					t.Errorf("Expected config to be %v, got %v", tt.want, config)
				}
			}
		})
	}
}

func TestFileLoader_getRuntimeConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  LoaderConfig
		mockFS  func(t *testing.T) fs.FS
		want    runtime.Config
		wantErr bool
	}{
		{
			name: "Invalid File Path",
			config: LoaderConfig{
				Type:     "file",
				Interval: 1 * time.Second,
				File: FileLoaderConfig{
					Path: "test/data/nonexistent.yaml",
				},
			},
			wantErr: true,
		},
		{
			name: "Malformed Config File",
			config: LoaderConfig{
				Type:     "file",
				Interval: 1 * time.Second,
				File: FileLoaderConfig{
					Path: "test/data/malformed.yaml",
				},
			},
			mockFS: func(_ *testing.T) fs.FS {
				return &test.MockFS{
					OpenFunc: func(name string) (fs.File, error) {
						content := []byte("this is not a valid yaml content")
						return &test.MockFile{Content: content}, nil
					},
				}
			},
			wantErr: true,
		},
		{
			name: "Failed to close file",
			config: LoaderConfig{
				Type:     "file",
				Interval: 1 * time.Second,
				File: FileLoaderConfig{
					Path: "test/data/valid.yaml",
				},
			},
			mockFS: func(t *testing.T) fs.FS {
				b, err := yaml.Marshal(LoaderConfig{
					Type:     "file",
					Interval: 1 * time.Second,
					File: FileLoaderConfig{
						Path: "test/data/valid.yaml",
					},
				})
				if err != nil {
					t.Fatalf("Failed marshaling response to bytes: %v", err)
				}

				return &test.MockFS{
					OpenFunc: func(name string) (fs.File, error) {
						return &test.MockFile{
							Content: b,
							CloseFunc: func() error {
								return fmt.Errorf("failed to close file")
							},
						}, nil
					},
				}
			},
			wantErr: true,
		},
		{
			name: "Malformed config file and failed to close file",
			config: LoaderConfig{
				Type:     "file",
				Interval: 1 * time.Second,
				File: FileLoaderConfig{
					Path: "test/data/malformed.yaml",
				},
			},
			mockFS: func(t *testing.T) fs.FS {
				return &test.MockFS{
					OpenFunc: func(name string) (fs.File, error) {
						return &test.MockFile{
							Content: []byte("this is not a valid yaml content"),
							CloseFunc: func() error {
								return fmt.Errorf("failed to close file")
							},
						}, nil
					},
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := make(chan runtime.Config, 1)
			defer close(res)
			f := NewFileLoader(&Config{
				Loader: tt.config,
			}, res)
			if tt.mockFS != nil {
				f.fsys = tt.mockFS(t)
			}

			cfg, err := f.getRuntimeConfig(t.Context())
			if (err != nil) != tt.wantErr {
				t.Errorf("getRuntimeConfig() error %v, want %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if !reflect.DeepEqual(cfg, tt.want) {
					t.Errorf("Expected config to be %v, got %v", tt.want, cfg)
				}
			}
		})
	}
}
