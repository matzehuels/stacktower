package main

import (
	"fmt"
	"os"

	"github.com/matzehuels/stacktower/internal/api"
)

func main() {
	if err := api.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
