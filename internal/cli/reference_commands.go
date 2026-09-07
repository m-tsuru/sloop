package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/m-tsuru/sloop/internal/sloop"
	"github.com/spf13/cobra"
)

func newReferenceCommand(stdout io.Writer) *cobra.Command {
	cmd := &cobra.Command{Use: "ref", Short: "Manage specification references"}
	cmd.AddCommand(newReferenceAddCommand(stdout))
	cmd.AddCommand(newReferenceListCommand(stdout))
	cmd.AddCommand(newReferenceRemoveCommand(stdout))
	return cmd
}

var lineRangePattern = regexp.MustCompile(`^(.*):(\d+)-(\d+)$`)

func parseReferenceTarget(value string) (string, *int, *int, error) {
	path := value
	var start, end *int
	if match := lineRangePattern.FindStringSubmatch(value); match != nil {
		path = match[1]
		startValue, _ := strconv.Atoi(match[2])
		endValue, _ := strconv.Atoi(match[3])
		if startValue < 1 || endValue < startValue {
			return "", nil, nil, fmt.Errorf("invalid line range in reference %q", value)
		}
		start, end = &startValue, &endValue
	}
	path = filepath.ToSlash(filepath.Clean(path))
	if path == "." || filepath.IsAbs(path) || path == ".." || strings.HasPrefix(path, "../") {
		return "", nil, nil, fmt.Errorf("reference path must be repository-relative: %q", value)
	}
	return path, start, end, nil
}

func validateReference(root, path, at string) (string, error) {
	if at == "" {
		info, err := os.Stat(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			return "", fmt.Errorf("inspect reference target %q: %w", path, err)
		}
		if info.IsDir() {
			return "", fmt.Errorf("reference target is not a file: %s", path)
		}
		return "", nil
	}
	commit, err := exec.Command("git", "-C", root, "rev-parse", "--verify", at+"^{commit}").Output()
	if err != nil {
		return "", fmt.Errorf("resolve Git commit %q", at)
	}
	fullCommit := strings.TrimSpace(string(commit))
	if err := exec.Command("git", "-C", root, "cat-file", "-e", fullCommit+":"+path).Run(); err != nil {
		return "", fmt.Errorf("path %q does not exist at commit %s", path, at)
	}
	return fullCommit, nil
}

func newReferenceAddCommand(stdout io.Writer) *cobra.Command {
	var kind, at string
	var authorOptions authorFlags
	cmd := &cobra.Command{
		Use:   "add <specification> <path[:start-end]>",
		Short: "Add an explicit repository reference",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			kind = strings.ToLower(strings.TrimSpace(kind))
			if kind != "code" && kind != "test" && kind != "context" {
				return fmt.Errorf("invalid reference kind %q (expected code, test, or context)", kind)
			}
			path, start, end, err := parseReferenceTarget(args[1])
			if err != nil {
				return err
			}
			project, err := openProject()
			if err != nil {
				return err
			}
			defer project.Store.Close()
			commit, err := validateReference(project.Project.Root, path, strings.TrimSpace(at))
			if err != nil {
				return err
			}
			spec, err := resolveSpecification(cmd.Context(), project, args[0])
			if err != nil {
				return err
			}
			ref := sloop.Reference{
				ID: uuid.NewString(), Kind: kind, Path: path, StartLine: start, EndLine: end,
				GitCommit: commit, CreatedAt: time.Now().Truncate(time.Microsecond),
			}
			if err := project.Store.AddReference(cmd.Context(), &spec, ref, authorOptions.resolve(project.Project.Root)); err != nil {
				return err
			}
			fmt.Fprintf(stdout, "Reference %s added to %s.\n", ref.ID, spec.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&kind, "kind", "", "reference kind: code, test, or context")
	cmd.Flags().StringVar(&at, "at", "", "Git commit that anchors the reference")
	_ = cmd.MarkFlagRequired("kind")
	authorOptions.bind(cmd)
	return cmd
}

func newReferenceListCommand(stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "list <specification>",
		Short: "List explicit references",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			project, err := openProject()
			if err != nil {
				return err
			}
			defer project.Store.Close()
			spec, err := resolveSpecification(cmd.Context(), project, args[0])
			if err != nil {
				return err
			}
			fmt.Fprintln(stdout, "ID                                    Kind     Target")
			for _, ref := range spec.References {
				target := ref.Path
				if ref.StartLine != nil {
					target += fmt.Sprintf(":%d-%d", *ref.StartLine, *ref.EndLine)
				}
				if ref.GitCommit != "" {
					target += " @ " + shortHash(ref.GitCommit)
				}
				fmt.Fprintf(stdout, "%-37s %-8s %s\n", ref.ID, ref.Kind, target)
			}
			return nil
		},
	}
}

func newReferenceRemoveCommand(stdout io.Writer) *cobra.Command {
	var authorOptions authorFlags
	cmd := &cobra.Command{
		Use:   "remove <specification> <reference-id>",
		Short: "Remove an explicit reference",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			project, err := openProject()
			if err != nil {
				return err
			}
			defer project.Store.Close()
			spec, err := resolveSpecification(cmd.Context(), project, args[0])
			if err != nil {
				return err
			}
			ref, err := project.Store.RemoveReference(cmd.Context(), &spec, args[1], authorOptions.resolve(project.Project.Root))
			if err != nil {
				return err
			}
			fmt.Fprintf(stdout, "Reference %s removed from %s.\n", ref.ID, spec.ID)
			return nil
		},
	}
	authorOptions.bind(cmd)
	return cmd
}
