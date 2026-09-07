package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/m-tsuru/sloop/internal/sloop"
	"github.com/spf13/cobra"
)

type commandContext struct {
	Project sloop.Project
	Store   *sloop.Store
}

func openProject() (*commandContext, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get current directory: %w", err)
	}
	project, err := sloop.FindProject(cwd)
	if err != nil {
		return nil, err
	}
	store, err := sloop.OpenStore(project.Config.Project.ID)
	if err != nil {
		return nil, err
	}
	return &commandContext{Project: project, Store: store}, nil
}

type authorFlags struct {
	name  string
	email string
	agent bool
}

func (a *authorFlags) bind(cmd *cobra.Command) {
	cmd.Flags().StringVar(&a.name, "author.name", "", "author name")
	cmd.Flags().StringVar(&a.email, "author.email", "", "author email")
	cmd.Flags().BoolVar(&a.agent, "author.agent", false, "author is a coding agent")
}

func (a authorFlags) resolve(repoRoot string) sloop.Author {
	name := a.name
	email := a.email
	if name == "" {
		name = gitConfig(repoRoot, "user.name")
	}
	if email == "" {
		email = gitConfig(repoRoot, "user.email")
	}
	if name == "" {
		name = os.Getenv("USER")
	}
	if name == "" {
		name = "unknown"
	}
	return sloop.Author{Name: name, Email: email, Agent: a.agent}
}

func gitConfig(root, key string) string {
	output, err := exec.Command("git", "-C", root, "config", "--get", key).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func resolveSpecification(ctx context.Context, project *commandContext, selector string) (sloop.Specification, error) {
	id := selector
	if number, ok := sloop.ParseSpecificationNumber(selector); ok {
		id = project.Project.Config.Project.SpecPrefix + "-" + strconv.Itoa(number)
	}
	return project.Store.SpecificationByID(ctx, id)
}

func editorName() (string, error) {
	editor := strings.TrimSpace(os.Getenv("SLOOP_EDITOR"))
	if editor == "" {
		editor = strings.TrimSpace(os.Getenv("EDITOR"))
	}
	if editor == "" {
		return "", fmt.Errorf("neither SLOOP_EDITOR nor EDITOR is set")
	}
	return editor, nil
}

func runEditor(content string) ([]byte, error) {
	editor, err := editorName()
	if err != nil {
		return nil, err
	}
	file, err := os.CreateTemp("", "sloop-*.md")
	if err != nil {
		return nil, fmt.Errorf("create temporary Markdown file: %w", err)
	}
	path := file.Name()
	defer os.Remove(path)
	if _, err := io.WriteString(file, content); err != nil {
		file.Close()
		return nil, fmt.Errorf("write temporary Markdown file: %w", err)
	}
	if err := file.Close(); err != nil {
		return nil, fmt.Errorf("close temporary Markdown file: %w", err)
	}
	command := exec.Command("sh", "-c", "exec "+editor+` "$1"`, "sloop-editor", path)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return nil, fmt.Errorf("editor failed: %w", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read edited Markdown: %w", err)
	}
	return data, nil
}

func shortHash(hash string) string {
	if len(hash) <= 8 {
		return hash
	}
	return hash[:8]
}
