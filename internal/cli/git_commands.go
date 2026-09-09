package cli

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/m-tsuru/sloop/internal/sloop"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type gitNote struct {
	Sloop []gitNoteRelation `yaml:"sloop"`
}

type gitNoteRelation struct {
	ID           string `yaml:"id"`
	RevisionHash string `yaml:"revision-hash"`
	Relation     string `yaml:"relation"`
}

func newCommitRecordCommand(stdout io.Writer) *cobra.Command {
	var authorOptions authorFlags
	cmd := &cobra.Command{
		Use:     "commit-record <specification[,specification...]> <commit[,commit...]>",
		Aliases: []string{"cr"},
		Short:   "Relate recorded specification revisions to Git commits",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			project, err := openProject()
			if err != nil {
				return err
			}
			defer project.Store.Close()
			commits, err := resolveGitCommits(project.Project.Root, args[1])
			if err != nil {
				return err
			}
			selectors, err := commaValues(args[0], "specification")
			if err != nil {
				return err
			}
			type recordedSpec struct {
				spec     sloop.Specification
				revision sloop.Revision
			}
			var recorded []recordedSpec
			seen := make(map[string]bool)
			for _, selector := range selectors {
				spec, err := resolveSpecification(cmd.Context(), project, selector)
				if err != nil {
					return err
				}
				if seen[spec.UUID] {
					continue
				}
				seen[spec.UUID] = true
				if spec.Dirty || spec.HeadHash == "" {
					author := spec.Author
					if authorOptions.name != "" || authorOptions.email != "" || authorOptions.agent {
						author = authorOptions.resolve(project.Project.Root)
					}
					if _, _, err := project.Store.RecordRevision(cmd.Context(), project.Project.Config.Project.ID, project.Project.Root, &spec, author); err != nil {
						return err
					}
				}
				revision, err := project.Store.RevisionByHash(cmd.Context(), spec.HeadHash)
				if err != nil {
					return err
				}
				recorded = append(recorded, recordedSpec{spec, revision})
			}

			for _, commit := range commits {
				note, err := readSloopNote(project.Project.Root, project.Project.Config.Git.NotesRef, commit)
				if err != nil {
					return err
				}
				for _, item := range recorded {
					note.Sloop = appendNoteRelation(note.Sloop, gitNoteRelation{
						ID: item.spec.ID, RevisionHash: item.revision.Hash, Relation: "implementation",
					})
				}
				if err := writeSloopNote(project.Project.Root, project.Project.Config.Git.NotesRef, commit, note); err != nil {
					return err
				}
				for _, item := range recorded {
					relation := sloop.GitRelation{SpecificationID: item.spec.ID, RevisionHash: item.revision.Hash,
						Commit: commit, Relation: "implementation", CreatedAt: time.Now().Truncate(time.Microsecond)}
					if err := project.Store.AddGitRelation(cmd.Context(), item.spec.UUID, relation); err != nil {
						return err
					}
					fmt.Fprintf(stdout, "Recorded %s@%s -> %s.\n", item.spec.ID, shortHash(item.revision.Hash), shortHash(commit))
				}
			}
			return nil
		},
	}
	authorOptions.bind(cmd)
	return cmd
}

func commaValues(value, kind string) ([]string, error) {
	var values []string
	seen := make(map[string]bool)
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			return nil, fmt.Errorf("empty %s selector", kind)
		}
		if !seen[item] {
			seen[item] = true
			values = append(values, item)
		}
	}
	return values, nil
}

func resolveGitCommits(root, selectors string) ([]string, error) {
	values, err := commaValues(selectors, "commit")
	if err != nil {
		return nil, err
	}
	var commits []string
	seen := make(map[string]bool)
	for _, value := range values {
		output, err := exec.Command("git", "-C", root, "rev-parse", "--verify", value+"^{commit}").Output()
		if err != nil {
			return nil, fmt.Errorf("resolve Git commit %q", value)
		}
		commit := strings.TrimSpace(string(output))
		if !seen[commit] {
			seen[commit] = true
			commits = append(commits, commit)
		}
	}
	return commits, nil
}

func readSloopNote(root, notesRef, commit string) (gitNote, error) {
	command := exec.Command("git", "-C", root, "notes", "--ref="+notesRef, "show", commit)
	output, err := command.Output()
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok && (strings.Contains(string(exit.Stderr), "no note found") || exit.ExitCode() == 1) {
			return gitNote{Sloop: []gitNoteRelation{}}, nil
		}
		return gitNote{}, fmt.Errorf("read Sloop Git note for %s: %w", shortHash(commit), err)
	}
	var note gitNote
	if err := yaml.Unmarshal(output, &note); err != nil {
		return gitNote{}, fmt.Errorf("decode Sloop Git note for %s: %w", shortHash(commit), err)
	}
	if note.Sloop == nil {
		return gitNote{}, fmt.Errorf("Git note for %s is not a Sloop note", shortHash(commit))
	}
	return note, nil
}

func appendNoteRelation(relations []gitNoteRelation, added gitNoteRelation) []gitNoteRelation {
	for _, relation := range relations {
		if relation == added {
			return relations
		}
	}
	relations = append(relations, added)
	sort.Slice(relations, func(i, j int) bool {
		if relations[i].ID != relations[j].ID {
			return relations[i].ID < relations[j].ID
		}
		if relations[i].RevisionHash != relations[j].RevisionHash {
			return relations[i].RevisionHash < relations[j].RevisionHash
		}
		return relations[i].Relation < relations[j].Relation
	})
	return relations
}

func writeSloopNote(root, notesRef, commit string, note gitNote) error {
	data, err := yaml.Marshal(note)
	if err != nil {
		return fmt.Errorf("encode Sloop Git note: %w", err)
	}
	command := exec.Command("git", "-C", root, "notes", "--ref="+notesRef, "add", "-f", "-F", "-", commit)
	command.Stdin = bytes.NewReader(data)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("write Sloop Git note for %s: %s", shortHash(commit), strings.TrimSpace(stderr.String()))
	}
	return nil
}
