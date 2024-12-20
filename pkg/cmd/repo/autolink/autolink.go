package autolink

import (
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdAutolink(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "autolink <command>",
		Short: "Manage autolink references",
		Long:  "Work with GitHub autolink references.",
	}
	cmdutil.EnableRepoOverride(cmd, f)

	cmd.AddCommand(newCmdList(f, nil))

	return cmd
}
