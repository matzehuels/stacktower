package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/matzehuels/stacktower/pkg/dag/perm"
)

func newPQTreeCmd() *cobra.Command {
	var output string
	var labels string

	cmd := &cobra.Command{
		Use:   "pqtree [constraints...]",
		Short: "Render a PQ-tree with optional constraints (debug tool)",
		Long: `Render a PQ-tree visualization showing valid permutations.

Constraints are comma-separated indices that must be adjacent.
Example: "0,1" means elements 0 and 1 must be adjacent.`,
		Example: `  # Universal tree with 4 elements
  stacktower pqtree --labels A,B,C,D -o tree.svg

  # With constraint: A,B must be adjacent  
  stacktower pqtree --labels A,B,C,D -o tree.svg 0,1

  # Multiple constraints
  stacktower pqtree --labels A,B,C,D -o tree.svg 0,1 2,3`,
		RunE: func(cmd *cobra.Command, args []string) error {
			labelList := strings.Split(labels, ",")
			if len(labelList) == 0 {
				return fmt.Errorf("at least one label required")
			}

			tree := perm.NewPQTree(len(labelList))

			for _, arg := range args {
				constraint, err := parseConstraint(arg)
				if err != nil {
					return fmt.Errorf("invalid constraint %q: %w", arg, err)
				}
				if !tree.Reduce(constraint) {
					return fmt.Errorf("constraint %q made tree unsatisfiable", arg)
				}
			}

			svg, err := tree.RenderSVG(labelList)
			if err != nil {
				return fmt.Errorf("render: %w", err)
			}

			out, err := openOutput(output)
			if err != nil {
				return err
			}
			defer out.Close()

			if _, err := out.Write(svg); err != nil {
				return err
			}

			fmt.Fprintf(os.Stderr, "Tree: %s\n", tree.StringWithLabels(labelList))
			fmt.Fprintf(os.Stderr, "Valid permutations: %d\n", tree.ValidCount())
			if output != "" {
				fmt.Fprintf(os.Stderr, "Output: %s\n", output)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "output file (stdout if empty)")
	cmd.Flags().StringVar(&labels, "labels", "A,B,C,D", "comma-separated node labels")

	return cmd
}

func parseConstraint(s string) ([]int, error) {
	parts := strings.Split(s, ",")
	if len(parts) < 2 {
		return nil, fmt.Errorf("need at least 2 indices")
	}
	result := make([]int, len(parts))
	for i, p := range parts {
		var n int
		if _, err := fmt.Sscanf(strings.TrimSpace(p), "%d", &n); err != nil {
			return nil, fmt.Errorf("invalid index %q", p)
		}
		result[i] = n
	}
	return result, nil
}
