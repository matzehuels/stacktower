package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// cacheCommand creates the cache management command.
func (c *CLI) cacheCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage the HTTP response cache",
	}

	cmd.AddCommand(c.cacheClearCommand())
	cmd.AddCommand(c.cachePathCommand())

	return cmd
}

// cacheClearCommand creates the "cache clear" subcommand.
func (c *CLI) cacheClearCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Clear all cached HTTP responses",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := cacheDir()
			if err != nil {
				return fmt.Errorf("get cache dir: %w", err)
			}

			if _, err := os.Stat(dir); os.IsNotExist(err) {
				printInfo("Cache is empty")
				return nil
			}

			count := 0
			err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil // Skip errors, continue walking
				}
				if path == dir {
					return nil
				}
				if !info.IsDir() {
					if err := os.Remove(path); err == nil {
						count++
					}
				}
				return nil
			})
			if err != nil {
				return err
			}

			// Clean up empty subdirectories
			_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
				if err != nil || path == dir {
					return nil
				}
				if info.IsDir() {
					os.Remove(path)
				}
				return nil
			})

			printSuccess("Cleared %d cached entries", count)
			printDetail("Directory: %s", dir)
			return nil
		},
	}
}

// cachePathCommand creates the "cache path" subcommand.
func (c *CLI) cachePathCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the cache directory path",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := cacheDir()
			if err != nil {
				return fmt.Errorf("get cache dir: %w", err)
			}
			fmt.Println(dir)
			return nil
		},
	}
}
