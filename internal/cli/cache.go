package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newCacheCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage the HTTP response cache",
	}

	cmd.AddCommand(newCacheClearCmd())
	cmd.AddCommand(newCachePathCmd())

	return cmd
}

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

func cacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache", "stacktower"), nil
}
