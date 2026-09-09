package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/m-tsuru/sloop/internal/sloop"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type specFilters map[string][]string

func parseFilters(values []string) (specFilters, error) {
	filters := make(specFilters)
	for _, value := range values {
		key, wanted, ok := strings.Cut(value, ":")
		key = strings.ToLower(strings.TrimSpace(key))
		wanted = strings.TrimSpace(wanted)
		if !ok || wanted == "" || (key != "status" && key != "title") {
			return nil, fmt.Errorf("invalid filter %q (expected status:<status> or title:<text>)", value)
		}
		if key == "status" {
			status, err := sloop.ParseStatus(wanted)
			if err != nil {
				return nil, err
			}
			wanted = string(status)
		}
		filters[key] = append(filters[key], wanted)
	}
	return filters, nil
}

func matchesFilters(spec sloop.Specification, filters specFilters) bool {
	for key, values := range filters {
		matched := false
		for _, value := range values {
			switch key {
			case "status":
				matched = matched || strings.EqualFold(string(spec.Status), value)
			case "title":
				matched = matched || strings.Contains(spec.Title, value)
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

type listRecord struct {
	ID         string       `json:"id"`
	Title      string       `json:"title"`
	Status     sloop.Status `json:"status"`
	LastUpdate time.Time    `json:"last_updated"`
	LastAuthor sloop.Author `json:"last_author"`
}

func newListCommand(stdout io.Writer) *cobra.Command {
	var jsonOutput, csvOutput bool
	var filterValues []string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List specifications",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if jsonOutput && csvOutput {
				return fmt.Errorf("--json and --csv cannot be used together")
			}
			project, err := openProject()
			if err != nil {
				return err
			}
			defer project.Store.Close()
			filters, err := parseFilters(filterValues)
			if err != nil {
				return err
			}
			specs, err := project.Store.Specifications(cmd.Context())
			if err != nil {
				return err
			}
			records := make([]listRecord, 0)
			for _, spec := range specs {
				if matchesFilters(spec, filters) {
					records = append(records, listRecord{spec.ID, spec.Title, spec.Status, spec.UpdatedAt, spec.Author})
				}
			}
			if jsonOutput {
				return writeJSON(stdout, records)
			}
			if csvOutput {
				writer := csv.NewWriter(stdout)
				if err := writer.Write([]string{"id", "title", "status", "last_updated", "last_author_name", "last_author_email", "last_author_agent"}); err != nil {
					return err
				}
				for _, record := range records {
					if err := writer.Write([]string{record.ID, record.Title, string(record.Status), record.LastUpdate.Format(time.RFC3339), record.LastAuthor.Name, record.LastAuthor.Email, strconv.FormatBool(record.LastAuthor.Agent)}); err != nil {
						return err
					}
				}
				writer.Flush()
				return writer.Error()
			}
			fmt.Fprintf(stdout, "Project: %s <%s>\n\n", project.Project.Config.Project.Slug, project.Project.Config.Project.ID)
			writer := tabwriter.NewWriter(stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(writer, "ID\tTitle\tStatus\tLast Updated\tLast Author")
			for _, record := range records {
				fmt.Fprintf(writer, "[%s]\t%s\t%s\t%s\t%s\n", record.ID, record.Title, record.Status, relativeTime(record.LastUpdate), record.LastAuthor.Name)
			}
			return writer.Flush()
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output JSON")
	cmd.Flags().BoolVar(&csvOutput, "csv", false, "output CSV")
	cmd.Flags().StringArrayVar(&filterValues, "filter", nil, "filter by status or title")
	return cmd
}

func relativeTime(value time.Time) string {
	duration := time.Since(value)
	if duration < 0 {
		duration = 0
	}
	if duration < time.Minute {
		seconds := int(duration.Seconds())
		if seconds == 1 {
			return "1 second ago"
		}
		return fmt.Sprintf("%d seconds ago", seconds)
	}
	if duration < time.Hour {
		return fmt.Sprintf("%d minutes ago", int(duration.Minutes()))
	}
	if duration < 24*time.Hour {
		return fmt.Sprintf("%d hours ago", int(duration.Hours()))
	}
	return fmt.Sprintf("%d days ago", int(duration.Hours()/24))
}

type viewAuthor struct {
	Name  string `yaml:"name"`
	Email string `yaml:"email,omitempty"`
	Agent bool   `yaml:"agent"`
}

type viewLog struct {
	Author       viewAuthor   `yaml:"author"`
	RevisionHash string       `yaml:"revision-hash"`
	Status       sloop.Status `yaml:"status"`
	Date         time.Time    `yaml:"date"`
}

type viewFrontMatter struct {
	Project      string                `yaml:"project"`
	ID           string                `yaml:"id"`
	Title        string                `yaml:"title"`
	Status       sloop.Status          `yaml:"status"`
	Revision     int                   `yaml:"revision,omitempty"`
	RevisionHash string                `yaml:"revision-hash,omitempty"`
	Parents      []string              `yaml:"parents"`
	Features     sloop.FeatureBindings `yaml:"func,omitempty"`
	References   []sloop.Reference     `yaml:"references,omitempty"`
	Log          []viewLog             `yaml:"log,omitempty"`
}

func newViewCommand(stdout io.Writer) *cobra.Command {
	var section string
	cmd := &cobra.Command{
		Use:   "view <specification-or-revision>",
		Short: "Print a specification as Markdown",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			project, err := openProject()
			if err != nil {
				return err
			}
			defer project.Store.Close()
			target, err := resolveEditTarget(cmd.Context(), project, args[0])
			if err != nil {
				return err
			}
			spec := target.spec
			body := spec.Body
			status, title, parents, features, refs := spec.Status, spec.Title, spec.Parents, spec.Features, spec.References
			var revisionNumber int
			var revisionHash string
			if target.revision != nil {
				body, status, title = target.revision.Content, target.revision.Status, target.revision.Title
				parents, features, refs = target.revision.Parents, target.revision.Features, target.revision.References
				revisionNumber, revisionHash = target.revision.RevisionNumber, target.revision.Hash
			} else if spec.HeadHash != "" && !spec.Dirty {
				revision, err := project.Store.RevisionByHash(cmd.Context(), spec.HeadHash)
				if err != nil {
					return err
				}
				revisionNumber, revisionHash = revision.RevisionNumber, revision.Hash
			}
			if section != "" {
				content, ok := sloop.Sections(body)[section]
				if !ok {
					return fmt.Errorf("section %q not found in %s", section, spec.ID)
				}
				fmt.Fprintln(stdout, content)
				return nil
			}
			revisions, err := project.Store.Revisions(cmd.Context(), spec.UUID)
			if err != nil {
				return err
			}
			front := viewFrontMatter{Project: project.Project.Config.Project.ID, ID: spec.ID, Title: title,
				Status: status, Revision: revisionNumber, RevisionHash: shortHash(revisionHash), Parents: parents,
				Features: features, References: refs}
			for _, revision := range revisions {
				if revisionNumber > 0 && revision.RevisionNumber > revisionNumber {
					break
				}
				front.Log = append(front.Log, viewLog{viewAuthor{revision.Author.Name, revision.Author.Email, revision.Author.Agent},
					shortHash(revision.Hash), revision.Status, revision.CreatedAt})
			}
			data, err := yaml.Marshal(front)
			if err != nil {
				return err
			}
			fmt.Fprintf(stdout, "---\n%s---\n\n%s", data, strings.TrimPrefix(body, "\n"))
			return nil
		},
	}
	cmd.Flags().StringVar(&section, "section", "", "print only a section ID")
	return cmd
}

func newQueryCommand(stdout io.Writer) *cobra.Command {
	var section string
	var filterValues []string
	cmd := &cobra.Command{
		Use:   "query",
		Short: "Retrieve a section from multiple specifications",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(section) == "" {
				return fmt.Errorf("--section is required")
			}
			project, err := openProject()
			if err != nil {
				return err
			}
			defer project.Store.Close()
			filters, err := parseFilters(filterValues)
			if err != nil {
				return err
			}
			specs, err := project.Store.Specifications(cmd.Context())
			if err != nil {
				return err
			}
			first := true
			for _, spec := range specs {
				content, ok := sloop.Sections(spec.Body)[section]
				if !matchesFilters(spec, filters) || !ok {
					continue
				}
				if !first {
					fmt.Fprintln(stdout)
				}
				first = false
				fmt.Fprintf(stdout, "## [%s] %s\n\n%s\n", spec.ID, spec.Title, content)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&section, "section", "", "section ID")
	cmd.Flags().StringArrayVar(&filterValues, "filter", nil, "filter by status or title")
	return cmd
}

type contextDocument struct {
	ProjectID          string                    `json:"project_id"`
	SpecificationID    string                    `json:"specification_id"`
	SpecificationUUID  string                    `json:"specification_uuid"`
	RevisionHash       string                    `json:"revision_hash,omitempty"`
	Recorded           bool                      `json:"recorded"`
	Status             sloop.Status              `json:"status"`
	Title              string                    `json:"title"`
	Goal               string                    `json:"goal"`
	Specification      string                    `json:"specification"`
	AcceptanceCriteria string                    `json:"acceptance_criteria"`
	Sections           map[string]string         `json:"sections"`
	Parents            []string                  `json:"parents"`
	References         []sloop.Reference         `json:"references"`
	Features           []sloop.FeatureResolution `json:"features"`
	GitRelations       []sloop.GitRelation       `json:"git_relations"`
	Reviews            []sloop.Review            `json:"reviews,omitempty"`
	ReviewPolicy       string                    `json:"review_policy"`
}

func newContextCommand(stdout io.Writer) *cobra.Command {
	var jsonOutput, working bool
	cmd := &cobra.Command{
		Use:   "context <specification>",
		Short: "Generate local context for a coding agent",
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
			if !working && (spec.Dirty || spec.HeadHash == "") {
				return fmt.Errorf("%s has unrecorded working changes.\nRecord or mark the specification READY before generating an agent context.", spec.ID)
			}
			document := contextDocument{
				ProjectID: project.Project.Config.Project.ID, SpecificationID: spec.ID,
				SpecificationUUID: spec.UUID,
				Recorded:          !working, Status: spec.Status, Title: spec.Title, Parents: spec.Parents,
				References: spec.References, ReviewPolicy: reviewPolicy(spec.Status),
			}
			body := spec.Body
			features := spec.Features
			if !working {
				revision, err := project.Store.RevisionByHash(cmd.Context(), spec.HeadHash)
				if err != nil {
					return err
				}
				document.RevisionHash, document.Status, document.Title = revision.Hash, revision.Status, revision.Title
				document.Parents, document.References, features, body = revision.Parents, revision.References, revision.Features, revision.Content
			}
			document.Features, err = featureBindingsForContext(project.Project.Root, features)
			if err != nil {
				return err
			}
			if (document.Status == sloop.StatusReady || document.Status == sloop.StatusForceReady) &&
				!sloop.FeatureBindingsResolved(document.Features) {
				return fmt.Errorf("%s", sloop.FormatUnresolvedFeatureBindings(document.Features,
					fmt.Sprintf("%s contains unresolved feature bindings.", spec.ID),
					"Remove or update the bindings before generating an agent context."))
			}
			document.Sections = sloop.Sections(body)
			document.Goal = document.Sections["goal"]
			document.Specification = document.Sections["specification"]
			document.AcceptanceCriteria = document.Sections["acceptance-criteria"]
			document.GitRelations, err = project.Store.GitRelations(cmd.Context(), spec.UUID)
			if err != nil {
				return err
			}
			if working {
				document.GitRelations = []sloop.GitRelation{}
			} else {
				relations := document.GitRelations[:0]
				for _, relation := range document.GitRelations {
					if relation.RevisionHash == document.RevisionHash {
						relations = append(relations, relation)
					}
				}
				document.GitRelations = relations
			}
			reviews, err := project.Store.Reviews(cmd.Context(), spec.UUID)
			if err != nil {
				return err
			}
			for _, review := range reviews {
				if !working && review.RevisionHash == document.RevisionHash {
					document.Reviews = append(document.Reviews, review)
				}
			}
			if jsonOutput {
				return writeJSON(stdout, document)
			}
			writeTextContext(stdout, document)
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output JSON")
	cmd.Flags().BoolVar(&working, "working", false, "use the mutable working state")
	return cmd
}

func reviewPolicy(status sloop.Status) string {
	switch status {
	case sloop.StatusForceReady:
		return "Continue implementation despite specification ambiguity; do not record a REJECTED review for ambiguity."
	case sloop.StatusReady:
		return "Reject the revision without implementing when externally observable behavior cannot be determined from the specification."
	default:
		return "This status is not a user confirmation that the specification is ready for implementation."
	}
}

func writeTextContext(output io.Writer, document contextDocument) {
	fmt.Fprintln(output, "# Sloop Agent Context")
	fmt.Fprintf(output, "\nSpecification ID: %s\nSpecification UUID: %s\nTitle: %s\nRevision Hash: %s\nStatus: %s\nRecorded: %t\n", document.SpecificationID, document.SpecificationUUID, document.Title, document.RevisionHash, document.Status, document.Recorded)
	if len(document.Parents) > 0 {
		fmt.Fprintf(output, "Parents: %s\n", strings.Join(document.Parents, ", "))
	}
	fmt.Fprintf(output, "\n## Goal\n\n%s\n\n## Specification\n\n%s\n\n## Acceptance Criteria\n\n%s\n", document.Goal, document.Specification, document.AcceptanceCriteria)
	fmt.Fprintln(output, "\n## Explicit References")
	for _, ref := range document.References {
		target := ref.Path
		if ref.StartLine != nil {
			target += fmt.Sprintf(":%d-%d", *ref.StartLine, *ref.EndLine)
		}
		if ref.GitCommit != "" {
			target += " @ " + ref.GitCommit
		}
		fmt.Fprintf(output, "- %s: %s\n", ref.Kind, target)
	}
	fmt.Fprintln(output, "\n## Feature Bindings")
	for _, feature := range document.Features {
		fmt.Fprintf(output, "- func:%s\n", feature.ID)
		for _, binding := range feature.Impls {
			fmt.Fprintf(output, "  - impl: %s [%s]\n", binding.Locator, binding.Status)
		}
		for _, binding := range feature.Tests {
			fmt.Fprintf(output, "  - test: %s [%s]\n", binding.Locator, binding.Status)
		}
	}
	fmt.Fprintln(output, "\n## Git Relations")
	for _, relation := range document.GitRelations {
		fmt.Fprintf(output, "- %s: %s (%s)\n", relation.Commit, relation.RevisionHash, relation.Relation)
	}
	fmt.Fprintf(output, "\n## Review Policy\n\n%s\n", document.ReviewPolicy)
}

func writeJSON(output io.Writer, value any) error {
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
