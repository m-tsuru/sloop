package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/m-tsuru/sloop/internal/sloop"
	"github.com/spf13/cobra"
)

func Execute(args []string, stdout, stderr io.Writer) error {
	root := newRootCommand(stdout, stderr)
	root.SetArgs(args)
	return root.Execute()
}

func newRootCommand(stdout, stderr io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "sloop",
		Short:         "Local-first specification management",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.AddCommand(newInitCommand(stdout))
	return cmd
}

func newInitCommand(stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "init <project-slug> <repository-path>",
		Short: "Initialize a Sloop project",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := strings.TrimSpace(args[0])
			if slug == "" || strings.ContainsAny(slug, `/\\`) {
				return fmt.Errorf("invalid project slug %q", args[0])
			}
			root, err := filepath.Abs(args[1])
			if err != nil {
				return fmt.Errorf("resolve repository path: %w", err)
			}
			info, err := os.Stat(root)
			if err != nil {
				return fmt.Errorf("inspect repository path: %w", err)
			}
			if !info.IsDir() {
				return fmt.Errorf("repository path is not a directory: %s", root)
			}

			configDir := filepath.Join(root, ".sloop")
			configPath := filepath.Join(configDir, "config.yaml")
			if _, err := os.Stat(configPath); err == nil {
				return fmt.Errorf("Sloop project is already initialized at %s", root)
			} else if !os.IsNotExist(err) {
				return fmt.Errorf("inspect existing project: %w", err)
			}

			projectID := slug + "-" + uuid.NewString()
			store, err := sloop.OpenStore(projectID)
			if err != nil {
				return err
			}
			if err := store.Close(); err != nil {
				return fmt.Errorf("close project database: %w", err)
			}

			if err := os.MkdirAll(filepath.Join(configDir, "templates"), 0o755); err != nil {
				return fmt.Errorf("create project directory: %w", err)
			}
			cfg := sloop.Config{
				Project:    sloop.ProjectConfig{ID: projectID, Slug: slug, SpecPrefix: slug},
				Repository: sloop.RepositoryConfig{Path: "."},
				Git:        sloop.GitConfig{NotesRef: "refs/notes/sloop"},
			}
			if err := sloop.WriteConfig(configPath, cfg); err != nil {
				return err
			}
			const template = "## 目的 {#goal}\n\n## 仕様 {#specification}\n\n## 完了条件 {#acceptance-criteria}\n"
			if err := os.WriteFile(filepath.Join(configDir, "templates", "default.md"), []byte(template), 0o644); err != nil {
				return fmt.Errorf("write default template: %w", err)
			}
			fmt.Fprintf(stdout, "Project '%s' is created.\nProject ID: %s\n", slug, projectID)
			return nil
		},
	}
}
