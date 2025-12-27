// Package term provides terminal output components for the Stacktower CLI.
//
// This package includes:
//   - Styled output functions (PrintSuccess, PrintError, PrintInfo, etc.)
//   - Progress spinners for long-running operations
//   - Summary boxes for completion messages
//   - Interactive TUI models for GitHub integration
//   - Consistent color palette and styling
//
// # Output Functions
//
// Simple styled output for common patterns:
//
//	term.PrintSuccess("Operation completed")
//	term.PrintError("Something went wrong")
//	term.PrintInfo("Processing %d items", count)
//	term.PrintKeyValue("Status", "Active")
//
// # Progress Spinner
//
//	spinner := term.NewSpinner("Loading...")
//	spinner.Start()
//	// ... do work ...
//	spinner.StopWithSuccess("Done!")
//
// # Summary Boxes
//
//	summary := term.NewSuccessSummary("Build complete").
//	    AddKeyValue("Files", "42").
//	    AddKeyValue("Time", "2.3s")
//	summary.Print()
package term
