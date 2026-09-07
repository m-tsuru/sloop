package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/m-tsuru/sloop/internal/sloop"
	"github.com/spf13/cobra"
)

func newNewCommand(stdout io.Writer) *cobra.Command {
	var templateName string
	var authorOptions authorFlags
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create a specification",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if templateName == "" || filepath.Base(templateName) != templateName || templateName == "." || templateName == ".." {
				return fmt.Errorf("invalid template name %q", templateName)
			}
			project, err := openProject()
			if err != nil {
				return err
			}
			defer project.Store.Close()
			ctx := cmd.Context()
			number, err := project.Store.NextSpecificationNumber(ctx)
			if err != nil {
				return err
			}
			templatePath := filepath.Join(project.Project.Root, ".sloop", "templates", templateName+".md")
			template, err := os.ReadFile(templatePath)
			if err != nil {
				return fmt.Errorf("read template %q: %w", templateName, err)
			}
			if strings.HasPrefix(strings.TrimSpace(string(template)), "---") {
				return fmt.Errorf("template %q must not contain YAML front matter", templateName)
			}
			author := authorOptions.resolve(project.Project.Root)
			spec := sloop.NewSpecification(project.Project.Config.Project.SpecPrefix, number, author, string(template))
			edited, err := runEditor(sloop.RenderEditable(spec))
			if err != nil {
				return err
			}
			doc, err := sloop.ParseEditable(edited)
			if err != nil {
				return err
			}
			if doc.ID != spec.ID {
				return fmt.Errorf("specification ID cannot be changed (expected %s)", spec.ID)
			}
			if author.Agent && sloop.Status(doc.Status) != sloop.StatusDraft {
				return fmt.Errorf("agents cannot change specification status through Markdown editing.\nUse `sloop status` with a reason instead.")
			}
			spec.Title, spec.Status, spec.Parents, spec.Body = doc.Title, sloop.Status(doc.Status), doc.Parents, doc.Body
			if err := project.Store.CreateSpecification(ctx, spec); err != nil {
				return err
			}
			if spec.Status != sloop.StatusDraft {
				revision, _, err := project.Store.RecordRevision(ctx, project.Project.Config.Project.ID, &spec, author)
				if err != nil {
					return err
				}
				if err := project.Store.RecordStatusTransition(ctx, sloop.StatusTransition{
					SpecUUID: spec.UUID, ResultingRevision: revision.Hash, Status: spec.Status,
					Author: author, CreatedAt: revision.CreatedAt,
				}); err != nil {
					return err
				}
			}
			fmt.Fprintf(stdout, "Specification %s is created.\n", spec.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&templateName, "template", "default", "template name")
	authorOptions.bind(cmd)
	return cmd
}

type editTarget struct {
	spec     sloop.Specification
	revision *sloop.Revision
}

func resolveEditTarget(ctx context.Context, project *commandContext, selector string) (editTarget, error) {
	if _, numeric := sloop.ParseSpecificationNumber(selector); numeric {
		spec, err := resolveSpecification(ctx, project, selector)
		if err == nil {
			return editTarget{spec: spec}, nil
		}
		// A digit-only short hash is also valid. Prefer a specification number
		// when it exists, then fall back to revision-prefix resolution.
		if len(selector) <= 64 {
			revision, revisionErr := project.Store.ResolveRevisionPrefix(ctx, selector)
			if revisionErr == nil {
				spec, revisionErr = project.Store.SpecificationByUUID(ctx, revision.SpecificationUUID)
				return editTarget{spec: spec, revision: &revision}, revisionErr
			}
		}
		return editTarget{}, err
	}
	if before, after, ok := strings.Cut(selector, "#"); ok {
		spec, err := resolveSpecification(ctx, project, before)
		if err != nil {
			return editTarget{}, err
		}
		number, err := strconv.Atoi(after)
		if err != nil || number < 1 {
			return editTarget{}, fmt.Errorf("invalid revision selector %q", selector)
		}
		revision, err := project.Store.RevisionByNumber(ctx, spec.UUID, number)
		return editTarget{spec: spec, revision: &revision}, err
	}
	if hash := strings.ToLower(selector); isHex(hash) {
		revision, err := project.Store.ResolveRevisionPrefix(ctx, hash)
		if err != nil {
			return editTarget{}, err
		}
		spec, err := project.Store.SpecificationByUUID(ctx, revision.SpecificationUUID)
		return editTarget{spec: spec, revision: &revision}, err
	}
	spec, err := resolveSpecification(ctx, project, selector)
	return editTarget{spec: spec}, err
}

func isHex(value string) bool {
	for _, char := range value {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')) {
			return false
		}
	}
	return value != ""
}

func newEditCommand(stdout io.Writer) *cobra.Command {
	var record bool
	var authorOptions authorFlags
	cmd := &cobra.Command{
		Use:   "edit <specification-or-revision>",
		Short: "Edit a specification working state",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			project, err := openProject()
			if err != nil {
				return err
			}
			defer project.Store.Close()
			ctx := cmd.Context()
			target, err := resolveEditTarget(ctx, project, args[0])
			if err != nil {
				return err
			}
			spec := target.spec
			author := authorOptions.resolve(project.Project.Root)

			if target.revision != nil && target.revision.Hash != spec.HeadHash {
				latestNumber := 0
				if spec.HeadHash != "" {
					if latest, err := project.Store.RevisionByHash(ctx, spec.HeadHash); err == nil {
						latestNumber = latest.RevisionNumber
					}
				}
				fmt.Fprintf(stdout, "Revision %d is currently the latest revision.\nRestore the content of revision %d (%s) as a new revision and open it for editing? [y/N] ",
					latestNumber, target.revision.RevisionNumber, shortHash(target.revision.Hash))
				answer, _ := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
				answer = strings.ToLower(strings.TrimSpace(answer))
				if answer != "y" && answer != "yes" {
					return nil
				}
				if spec.Dirty {
					boundaryAuthor := spec.Author
					if boundaryAuthor.Name == "" {
						boundaryAuthor = author
					}
					if _, _, err := project.Store.RecordRevision(ctx, project.Project.Config.Project.ID, &spec, boundaryAuthor); err != nil {
						return err
					}
				}
				if _, err := project.Store.RestoreRevision(ctx, project.Project.Config.Project.ID, &spec, *target.revision, author); err != nil {
					return err
				}
			}

			authorBoundary := !spec.Author.Equal(author)
			if record || authorBoundary {
				boundaryAuthor := spec.Author
				if boundaryAuthor.Name == "" {
					boundaryAuthor = author
				}
				if _, _, err := project.Store.RecordRevision(ctx, project.Project.Config.Project.ID, &spec, boundaryAuthor); err != nil {
					return err
				}
			}
			edited, err := runEditor(sloop.RenderEditable(spec))
			if err != nil {
				return err
			}
			doc, err := sloop.ParseEditable(edited)
			if err != nil {
				return err
			}
			if doc.ID != spec.ID {
				return fmt.Errorf("specification ID cannot be changed (expected %s)", spec.ID)
			}
			requestedStatus := sloop.Status(doc.Status)
			if author.Agent && requestedStatus != spec.Status {
				return fmt.Errorf("agents cannot change specification status through Markdown editing.\nUse `sloop status` with a reason instead.")
			}
			meaningful := !sloop.EqualMeaning(spec, doc)
			statusChanged := requestedStatus != spec.Status
			if !meaningful && !statusChanged {
				return nil
			}
			oldStatus := spec.Status
			spec.Title, spec.Parents, spec.Body = doc.Title, doc.Parents, doc.Body
			if meaningful && oldStatus != sloop.StatusDraft {
				spec.Status = sloop.StatusDraft
			} else {
				spec.Status = requestedStatus
			}
			spec.Author = author
			spec.UpdatedAt = time.Now().Truncate(time.Microsecond)
			spec.Dirty = true
			if err := project.Store.SaveSpecification(ctx, spec); err != nil {
				return err
			}
			if record || authorBoundary || spec.Status != oldStatus {
				if _, _, err := project.Store.RecordRevision(ctx, project.Project.Config.Project.ID, &spec, author); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&record, "record", false, "record revisions before and after editing")
	authorOptions.bind(cmd)
	return cmd
}

func newLogCommand(stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "log <specification>",
		Short: "Show recorded revision history",
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
			revisions, err := project.Store.Revisions(cmd.Context(), spec.UUID)
			if err != nil {
				return err
			}
			fmt.Fprintf(stdout, "Specification: %s\n\nRev  Hash      Status       Author                  Date\n", spec.ID)
			for _, revision := range revisions {
				fmt.Fprintf(stdout, "%-4d %-9s %-12s %-23s %s\n", revision.RevisionNumber, shortHash(revision.Hash),
					revision.Status, revision.Author.Name, revision.CreatedAt.Local().Format("2006-01-02 15:04"))
			}
			return nil
		},
	}
}
