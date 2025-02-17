package cmd

import (
	"fmt"

	"github.com/mohamed-rafraf/kubectl-foren/pkg/state"
	"github.com/mohamed-rafraf/kubectl-foren/pkg/tasks"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type nodeNetworkOpts struct {
	globalOptions
}

func (opts *nodeNetworkOpts) BuildState() (*state.State, error) {
	s, err := opts.globalOptions.BuildState()
	if err != nil {
		return nil, err
	}

	return s, nil
}

func nodeNetworkCmd(rootFlags *pflag.FlagSet) *cobra.Command {
	opts := &nodeNetworkOpts{}
	cmd := &cobra.Command{
		Use:           "node-net [node-name]",
		Short:         "List network interfaces on a node",
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

			return runNodeNetworkCmd(st, opts, nodeName)
		},
	}
	return cmd
}

// runNodeNetworkCmd lists the network interfaces in node.
func runNodeNetworkCmd(st *state.State, _ *nodeNetworkOpts, nodeName string) error {
	st.Logger.Info(fmt.Sprintf("Listing the network interfaces on %s", nodeName))
	taskList := tasks.Tasks{
		tasks.DeloyForenPod(st, nodeName),
		tasks.WaitForenPodRunning(st, nodeName),
		tasks.ExecuteNoraml(st, nodeName, "ip addr"),
		tasks.DeleteForenPod(st, nodeName),
	}

	err := taskList.Run(st)
	if err != nil {
		tasks.DeleteForenPod(st, nodeName).Fn(st)
		return err
	}
	return nil
}
