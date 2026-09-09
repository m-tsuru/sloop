package cli

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/m-tsuru/sloop/internal/sloop"
	"github.com/spf13/cobra"
)

func newStatusCommand(stdout io.Writer) *cobra.Command {
	var authorOptions authorFlags
	var reason string
	cmd := &cobra.Command{
		Use:   "status <specification[,specification...]> <status>",
		Short: "Change specification status",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			status, err := sloop.ParseStatus(args[1])
			if err != nil {
				return err
			}
			return changeStatuses(cmd, stdout, args[0], status, reason, authorOptions)
		},
	}
	cmd.Flags().StringVar(&reason, "reason", "", "reason for the transition")
	authorOptions.bind(cmd)
	return cmd
}

func newStatusAliasCommand(name string, status sloop.Status, stdout io.Writer) *cobra.Command {
	var authorOptions authorFlags
	var reason string
	cmd := &cobra.Command{
		Use:   name + " <specification[,specification...]>",
		Short: "Set specification status to " + string(status),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return changeStatuses(cmd, stdout, args[0], status, reason, authorOptions)
		},
	}
	cmd.Flags().StringVar(&reason, "reason", "", "reason for the transition")
	authorOptions.bind(cmd)
	return cmd
}

func changeStatuses(cmd *cobra.Command, stdout io.Writer, selectors string, status sloop.Status, reason string, authorOptions authorFlags) error {
	project, err := openProject()
	if err != nil {
		return err
	}
	defer project.Store.Close()
	author := authorOptions.resolve(project.Project.Root)
	reason = strings.TrimSpace(reason)
	if author.Agent {
		if reason == "" {
			return fmt.Errorf("agent status transitions require a reason.")
		}
		if status != sloop.StatusImplemented && status != sloop.StatusVerified {
			return fmt.Errorf("an agent cannot set specification status to %s.", status)
		}
		if trivialReason(reason) {
			return fmt.Errorf("agent status transition reason must explain the basis for the transition")
		}
	}
	seen := make(map[string]bool)
	for _, selector := range strings.Split(selectors, ",") {
		selector = strings.TrimSpace(selector)
		if selector == "" {
			return fmt.Errorf("empty specification selector")
		}
		spec, err := resolveSpecification(cmd.Context(), project, selector)
		if err != nil {
			return err
		}
		if seen[spec.UUID] {
			continue
		}
		seen[spec.UUID] = true
		if spec.Status == status {
			continue
		}
		if _, err := validateSpecificationBindings(project.Project.Root, spec,
			fmt.Sprintf("Remove or update the bindings before setting status %s.", status)); err != nil {
			return err
		}
		if author.Agent && (spec.HeadHash == "" || spec.Dirty) {
			return fmt.Errorf("agent status transitions require a current recorded revision for %s", spec.ID)
		}
		previous := spec
		basedOn := spec.HeadHash
		spec.Status = status
		spec.Author = author
		spec.UpdatedAt = time.Now().Truncate(time.Microsecond)
		spec.Dirty = true
		if err := project.Store.SaveSpecification(cmd.Context(), spec); err != nil {
			return err
		}
		revision, _, err := project.Store.RecordRevision(cmd.Context(), project.Project.Config.Project.ID, project.Project.Root, &spec, author)
		if err != nil {
			if rollbackErr := project.Store.SaveSpecification(cmd.Context(), previous); rollbackErr != nil {
				return fmt.Errorf("%w (also failed to restore previous working state: %v)", err, rollbackErr)
			}
			return err
		}
		if err := project.Store.RecordStatusTransition(cmd.Context(), sloop.StatusTransition{
			SpecUUID: spec.UUID, BasedOnRevision: basedOn, ResultingRevision: revision.Hash,
			Status: status, Reason: reason, Author: author, CreatedAt: revision.CreatedAt,
		}); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Specification %s status changed to %s.\n", spec.ID, status)
	}
	return nil
}

func trivialReason(reason string) bool {
	value := strings.ToLower(strings.TrimSpace(strings.TrimSuffix(reason, ".")))
	return value == "done" || value == "completed" || value == "success"
}

func newReviewCommand(result string, stdout io.Writer) *cobra.Command {
	var authorOptions authorFlags
	var reason string
	name := "reject"
	if result == "ACCEPTED" {
		name = "accept"
	}
	cmd := &cobra.Command{
		Use:   name + " <specification>",
		Short: "Record an " + result + " review",
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
			reason = strings.TrimSpace(reason)
			if result == "REJECTED" && reason == "" {
				return fmt.Errorf("rejected reviews require a reason")
			}
			if result == "REJECTED" && spec.Status == sloop.StatusForceReady {
				return fmt.Errorf("specification %s is FORCEREADY and cannot be rejected for specification ambiguity.", spec.ID)
			}
			if spec.HeadHash == "" || spec.Dirty {
				return fmt.Errorf("%s has no current recorded revision to review", spec.ID)
			}
			author := authorOptions.resolve(project.Project.Root)
			if err := project.Store.AddReview(cmd.Context(), sloop.Review{
				SpecUUID: spec.UUID, RevisionHash: spec.HeadHash, Result: result, Reason: reason,
				Author: author, CreatedAt: time.Now().Truncate(time.Microsecond),
			}); err != nil {
				return err
			}
			fmt.Fprintf(stdout, "%s review recorded for %s at %s.\n", result, spec.ID, shortHash(spec.HeadHash))
			return nil
		},
	}
	cmd.Flags().StringVar(&reason, "reason", "", "review reason")
	authorOptions.bind(cmd)
	return cmd
}

func newAgentRunCommand(stdout io.Writer) *cobra.Command {
	var authorOptions authorFlags
	var reason string
	cmd := &cobra.Command{
		Use:     "agent-run <specification> <implemented|failed>",
		Aliases: []string{"run-record"},
		Short:   "Record a coding agent run",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			result := strings.ToUpper(strings.TrimSpace(args[1]))
			if result != "IMPLEMENTED" && result != "FAILED" {
				return fmt.Errorf("invalid agent run result %q", args[1])
			}
			project, err := openProject()
			if err != nil {
				return err
			}
			defer project.Store.Close()
			spec, err := resolveSpecification(cmd.Context(), project, args[0])
			if err != nil {
				return err
			}
			if spec.HeadHash == "" || spec.Dirty {
				return fmt.Errorf("%s has no current recorded revision for an agent run", spec.ID)
			}
			author := authorOptions.resolve(project.Project.Root)
			author.Agent = true
			if err := project.Store.AddAgentRun(cmd.Context(), sloop.AgentRun{
				SpecUUID: spec.UUID, RevisionHash: spec.HeadHash, Result: result,
				Reason: strings.TrimSpace(reason), Author: author, CreatedAt: time.Now().Truncate(time.Microsecond),
			}); err != nil {
				return err
			}
			fmt.Fprintf(stdout, "Agent run %s recorded for %s at %s.\n", result, spec.ID, shortHash(spec.HeadHash))
			return nil
		},
	}
	cmd.Flags().StringVar(&reason, "reason", "", "agent run details")
	authorOptions.bind(cmd)
	return cmd
}
