package cli

import (
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/m-tsuru/sloop/internal/sloop"
	"github.com/spf13/cobra"
)

func newBindingCommand(stdout io.Writer) *cobra.Command {
	cmd := &cobra.Command{Use: "bind", Short: "Manage feature bindings"}
	cmd.AddCommand(newBindingAddCommand(stdout))
	cmd.AddCommand(newBindingRemoveCommand(stdout))
	cmd.AddCommand(newBindingListCommand(stdout))
	return cmd
}

type bindingFlags struct {
	impls []string
	tests []string
}

func (f *bindingFlags) bind(cmd *cobra.Command) {
	cmd.Flags().StringArrayVar(&f.impls, "impl", nil, "implementation symbol locator")
	cmd.Flags().StringArrayVar(&f.tests, "test", nil, "test symbol locator")
}

func (f bindingFlags) normalized() (bindingFlags, error) {
	if len(f.impls) == 0 && len(f.tests) == 0 {
		return bindingFlags{}, fmt.Errorf("at least one --impl or --test is required")
	}
	impls, err := normalizeCLIFlags(f.impls)
	if err != nil {
		return bindingFlags{}, err
	}
	tests, err := normalizeCLIFlags(f.tests)
	if err != nil {
		return bindingFlags{}, err
	}
	return bindingFlags{impls: impls, tests: tests}, nil
}

func normalizeCLIFlags(values []string) ([]string, error) {
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		locator, err := sloop.ParseSymbolLocator(value)
		if err != nil {
			return nil, err
		}
		normalized := locator.String()
		if !seen[normalized] {
			seen[normalized] = true
			result = append(result, normalized)
		}
	}
	sort.Strings(result)
	return result, nil
}

func newBindingAddCommand(stdout io.Writer) *cobra.Command {
	var flags bindingFlags
	var authorOptions authorFlags
	cmd := &cobra.Command{
		Use:   "add <specification> <feature>",
		Short: "Add resolved implementation or test symbols to a feature",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := sloop.ValidateFeatureID(args[1]); err != nil {
				return err
			}
			normalized, err := flags.normalized()
			if err != nil {
				return err
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

			for _, value := range append(append([]string(nil), normalized.impls...), normalized.tests...) {
				resolution, err := sloop.ResolveSymbol(project.Project.Root, value)
				if err != nil {
					return err
				}
				if resolution.Status != sloop.BindingResolved {
					return bindingAddResolutionError(resolution)
				}
			}

			binding := spec.Features[args[1]]
			binding.Impls = addUniqueStrings(binding.Impls, normalized.impls)
			binding.Tests = addUniqueStrings(binding.Tests, normalized.tests)
			features := sloop.CloneFeatureBindings(spec.Features)
			features[args[1]] = binding
			if sloop.EqualFeatureBindings(spec.Features, features) {
				fmt.Fprintf(stdout, "Feature bindings for %s in %s are unchanged.\n", args[1], spec.ID)
				return nil
			}
			if err := saveBindingChange(cmd, project, &spec, features, authorOptions.resolve(project.Project.Root)); err != nil {
				return err
			}
			fmt.Fprintf(stdout, "Feature bindings added to %s in %s.\n", args[1], spec.ID)
			return nil
		},
	}
	flags.bind(cmd)
	authorOptions.bind(cmd)
	return cmd
}

func bindingAddResolutionError(resolution sloop.BindingResolution) error {
	switch resolution.Status {
	case sloop.BindingMissing:
		symbol := resolution.Symbol
		locator, err := sloop.ParseSymbolLocator(resolution.Locator)
		if err == nil && locator.Owner != "" {
			symbol = locator.Owner + ":" + locator.Symbol
		}
		return fmt.Errorf("symbol '%s' could not be resolved in %s.", symbol, resolution.Path)
	case sloop.BindingAmbiguous:
		return fmt.Errorf("symbol locator '%s' is ambiguous.", resolution.Locator)
	case sloop.BindingUnsupported:
		return fmt.Errorf("symbol locator '%s' uses an unsupported language.", resolution.Locator)
	default:
		return fmt.Errorf("symbol locator '%s' could not be resolved.", resolution.Locator)
	}
}

func newBindingRemoveCommand(stdout io.Writer) *cobra.Command {
	var flags bindingFlags
	var authorOptions authorFlags
	cmd := &cobra.Command{
		Use:   "remove <specification> <feature>",
		Short: "Remove implementation or test symbols from a feature",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := sloop.ValidateFeatureID(args[1]); err != nil {
				return err
			}
			normalized, err := flags.normalized()
			if err != nil {
				return err
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
			binding, exists := spec.Features[args[1]]
			if !exists {
				fmt.Fprintf(stdout, "Feature bindings for %s in %s are unchanged.\n", args[1], spec.ID)
				return nil
			}
			binding.Impls = removeStrings(binding.Impls, normalized.impls)
			binding.Tests = removeStrings(binding.Tests, normalized.tests)
			features := sloop.CloneFeatureBindings(spec.Features)
			features[args[1]] = binding
			if sloop.EqualFeatureBindings(spec.Features, features) {
				fmt.Fprintf(stdout, "Feature bindings for %s in %s are unchanged.\n", args[1], spec.ID)
				return nil
			}
			if err := saveBindingChange(cmd, project, &spec, features, authorOptions.resolve(project.Project.Root)); err != nil {
				return err
			}
			fmt.Fprintf(stdout, "Feature bindings removed from %s in %s.\n", args[1], spec.ID)
			return nil
		},
	}
	flags.bind(cmd)
	authorOptions.bind(cmd)
	return cmd
}

func addUniqueStrings(existing, additions []string) []string {
	values := append(append([]string(nil), existing...), additions...)
	seen := make(map[string]bool, len(values))
	result := values[:0]
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}

func removeStrings(existing, removals []string) []string {
	remove := make(map[string]bool, len(removals))
	for _, value := range removals {
		remove[value] = true
	}
	result := make([]string, 0, len(existing))
	for _, value := range existing {
		if !remove[value] {
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}

func saveBindingChange(cmd *cobra.Command, project *commandContext, spec *sloop.Specification, features sloop.FeatureBindings, author sloop.Author) error {
	oldStatus := spec.Status
	authorBoundary := !spec.Author.Equal(author)
	if authorBoundary && spec.Dirty {
		boundaryAuthor := spec.Author
		if boundaryAuthor.Name == "" {
			boundaryAuthor = author
		}
		if _, _, err := project.Store.RecordRevision(cmd.Context(), project.Project.Config.Project.ID, project.Project.Root, spec, boundaryAuthor); err != nil {
			return err
		}
	}
	spec.Features = features
	spec.Status = sloop.StatusDraft
	spec.Author = author
	spec.UpdatedAt = time.Now().Truncate(time.Microsecond)
	spec.Dirty = true
	if err := project.Store.SaveSpecification(cmd.Context(), *spec); err != nil {
		return err
	}
	resolutions, err := sloop.ResolveFeatureBindings(project.Project.Root, spec.Features)
	if err != nil {
		return err
	}
	if !sloop.FeatureBindingsResolved(resolutions) {
		cmd.PrintErrln(sloop.FormatUnresolvedFeatureBindings(resolutions,
			"Warning: unresolved feature binding.", ""))
		return nil
	}
	if oldStatus != sloop.StatusDraft || authorBoundary || author.Agent {
		_, _, err = project.Store.RecordRevision(cmd.Context(), project.Project.Config.Project.ID, project.Project.Root, spec, author)
	}
	return err
}

type bindingListFeature struct {
	Feature string                    `json:"feature"`
	Impls   []sloop.BindingResolution `json:"impls"`
	Tests   []sloop.BindingResolution `json:"tests"`
}

func newBindingListCommand(stdout io.Writer) *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "list <specification> [feature]",
		Short: "List feature bindings and current resolution status",
		Args:  cobra.RangeArgs(1, 2),
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
			resolutions, err := sloop.ResolveFeatureBindings(project.Project.Root, spec.Features)
			if err != nil {
				return err
			}
			records := make([]bindingListFeature, 0, len(resolutions))
			for _, resolution := range resolutions {
				if len(args) == 2 && resolution.ID != args[1] {
					continue
				}
				records = append(records, bindingListFeature{
					Feature: resolution.ID, Impls: resolution.Impls, Tests: resolution.Tests,
				})
			}
			if len(args) == 2 {
				if err := sloop.ValidateFeatureID(args[1]); err != nil {
					return err
				}
				if len(records) == 0 {
					return fmt.Errorf("feature %q not found in %s", args[1], spec.ID)
				}
				if jsonOutput {
					return writeJSON(stdout, records[0])
				}
				writeBindingFeature(stdout, records[0])
				return nil
			}
			if jsonOutput {
				return writeJSON(stdout, records)
			}
			for index, record := range records {
				if index > 0 {
					fmt.Fprintln(stdout)
				}
				writeBindingFeature(stdout, record)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output JSON")
	return cmd
}

func writeBindingFeature(stdout io.Writer, feature bindingListFeature) {
	fmt.Fprintf(stdout, "Feature: %s\n", feature.Feature)
	writeBindingKind(stdout, "impls", feature.Impls)
	writeBindingKind(stdout, "tests", feature.Tests)
}

func writeBindingKind(stdout io.Writer, kind string, bindings []sloop.BindingResolution) {
	fmt.Fprintf(stdout, "  %s:\n", kind)
	for _, binding := range bindings {
		fmt.Fprintf(stdout, "    %s [%s]\n", binding.Locator, binding.Status)
	}
}

func validateSpecificationBindings(root string, spec sloop.Specification, action string) ([]sloop.FeatureResolution, error) {
	resolutions, err := sloop.ResolveFeatureBindings(root, spec.Features)
	if err != nil {
		return nil, err
	}
	if sloop.FeatureBindingsResolved(resolutions) {
		return resolutions, nil
	}
	return resolutions, fmt.Errorf("%s", sloop.FormatUnresolvedFeatureBindings(resolutions,
		fmt.Sprintf("%s contains unresolved feature bindings.", spec.ID), action))
}

func warnUnresolvedBindings(cmd *cobra.Command, root string, features sloop.FeatureBindings) error {
	resolutions, err := sloop.ResolveFeatureBindings(root, features)
	if err != nil {
		return err
	}
	if !sloop.FeatureBindingsResolved(resolutions) {
		cmd.PrintErrln(sloop.FormatUnresolvedFeatureBindings(resolutions,
			"Warning: unresolved feature binding.", ""))
	}
	return nil
}

func featureBindingsForContext(root string, features sloop.FeatureBindings) ([]sloop.FeatureResolution, error) {
	return sloop.ResolveFeatureBindings(root, features)
}
