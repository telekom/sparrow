// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package main

//go:generate go run gen-docs.go gen-docs --path ../../docs

import (
	"fmt"
	"os"

	sparrowcmd "github.com/caas-team/sparrow/cmd"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func main() {
	execute()
}

func execute() {
	rootCmd := &cobra.Command{
		Use:   "gen-docs",
		Short: "Generates docs for sparrow",
	}
	rootCmd.AddCommand(NewCmdGenDocs())

	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// NewCmdGenDocs creates a new gen-docs command
func NewCmdGenDocs() *cobra.Command {
	var docPath string

	cmd := &cobra.Command{
		Use:   "gen-docs",
		Short: "Generate markdown documentation",
		Long:  `Generate the markdown documentation of available CLI flags`,
		RunE:  runGenDocs(&docPath),
	}

	cmd.PersistentFlags().StringVar(&docPath, "path", "docs", "directory path where the markdown files will be created")

	return cmd
}

// runGenDocs generates the markdown files for the flag documentation
func runGenDocs(path *string) func(cmd *cobra.Command, args []string) error {
	c := sparrowcmd.BuildCmd("")
	c.DisableAutoGenTag = false
	return func(_ *cobra.Command, _ []string) error {
		if err := doc.GenMarkdownTree(c, *path); err != nil {
			return fmt.Errorf("failed to generate docs: %w", err)
		}
		return nil
	}
}
