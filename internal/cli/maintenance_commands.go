package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func newIndexCommand(stdout io.Writer) *cobra.Command {
	cmd := &cobra.Command{Use: "index", Short: "Manage local indexes"}
	cmd.AddCommand(&cobra.Command{
		Use:   "rebuild",
		Short: "Rebuild the SQLite revision index from immutable objects",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			project, err := openProject()
			if err != nil {
				return err
			}
			defer project.Store.Close()
			count, err := project.Store.RebuildIndex(cmd.Context(), project.Project.Config.Project.ID)
			if err != nil {
				return err
			}
			fmt.Fprintf(stdout, "Rebuilt index from %d revision objects.\n", count)
			return nil
		},
	})
	return cmd
}

func newGCCommand(stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "gc",
		Short: "Remove regenerable cache objects",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			project, err := openProject()
			if err != nil {
				return err
			}
			defer project.Store.Close()
			count, err := project.Store.GarbageCollect()
			if err != nil {
				return err
			}
			fmt.Fprintf(stdout, "Removed %d cache objects.\n", count)
			return nil
		},
	}
}
