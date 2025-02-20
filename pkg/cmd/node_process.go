package cmd

import (
	"fmt"

	"github.com/mohamed-rafraf/kubectl-foren/pkg/state"
	"github.com/mohamed-rafraf/kubectl-foren/pkg/tasks"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type nodeProcessOpts struct {
	globalOptions
}

func (opts *nodeProcessOpts) BuildState() (*state.State, error) {
	s, err := opts.globalOptions.BuildState()
	if err != nil {
		return nil, err
	}

	return s, nil
}

func nodeProcessCmd(rootFlags *pflag.FlagSet) *cobra.Command {
	opts := &nodeProcessOpts{}
	cmd := &cobra.Command{
		Use:           "node-process [node-name]",
		Short:         "List running processes on a node",
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, args []string) error {
			nodeName := args[0]

			gopts, err := persistentGlobalOptions(rootFlags)
			if err != nil {
				return err
			}

			opts.globalOptions = *gopts
			st, err := opts.BuildState()
			if err != nil {
				return err
			}

			return runNodeProcessCmd(st, opts, nodeName)
		},
	}
	return cmd
}

// runNodeProcessCmd orchestrates the deployment of a temporary pod to list running processes on a node.
func runNodeProcessCmd(st *state.State, _ *nodeProcessOpts, nodeName string) error {
	st.Logger.Info(fmt.Sprintf("Listing the running processes on %s", nodeName))
	// Define the task list for node process command
	taskList := tasks.Tasks{
		tasks.DeloyForenPod(st, nodeName),
		tasks.WaitForenPodRunning(st, nodeName),
		tasks.ExecuteInteractive(st, nodeName, "top"),
		tasks.DeleteForenPod(st, nodeName),
	}

	return taskList.Run(st)
}
