package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// newCacheCmd creates the cache management command with subcommands for clearing and inspecting the cache.
// The cache stores HTTP responses from package registries to reduce network calls and improve performance.
func newCacheCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage the HTTP response cache",
	}

	cmd.AddCommand(newCacheClearCmd())
	cmd.AddCommand(newCachePathCmd())

	return cmd
}

// newCacheClearCmd creates the "cache clear" subcommand.
// It removes all non-directory files from the cache directory.
// If the cache directory does not exist, the command prints "Cache is empty" and succeeds.
// Failed removals are silently skipped; only successful deletions are counted.
func newCacheClearCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Clear all cached HTTP responses",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := cacheDir()
			if err != nil {
				return err
			}

			entries, err := os.ReadDir(dir)
			if os.IsNotExist(err) {
				fmt.Println("Cache is empty")
				return nil
			}
			if err != nil {
				return err
			}

			count := 0
			for _, entry := range entries {
				if !entry.IsDir() {
					if err := os.Remove(filepath.Join(dir, entry.Name())); err == nil {
						count++
					}
				}
			}

			fmt.Printf("Cleared %d cached entries from %s\n", count, dir)
			return nil
		},
	}
}

// newCachePathCmd creates the "cache path" subcommand.
// It prints the absolute path to the cache directory.
// The directory may not exist if no cached responses have been stored yet.
func newCachePathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the cache directory path",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := cacheDir()
			if err != nil {
				return err
			}
			fmt.Println(dir)
			return nil
		},
	}
}

// cacheDir returns the absolute path to the stacktower cache directory.
// The directory is located at $HOME/.cache/stacktower and follows XDG conventions.
// The directory is created on-demand by the deps package; this function only computes the path.
//
// It returns an error if the user's home directory cannot be determined.
func cacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache", "stacktower"), nil
}
