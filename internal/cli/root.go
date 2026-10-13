// Package cli wires the cobra command tree for the chuck binary.
package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/AsaphNoam/Chuck/internal/version"
)

// NewRootCmd builds the root cobra command with --version and subcommands.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "chuck",
		Short:         "Chuck — local dashboard for orchestrating coding agents",
		Version:       version.String(),
		SilenceUsage:  true,
		SilenceErrors: false,
	}
	// `chuck --version` prints "chuck version <version> (commit, date)".
	root.SetVersionTemplate("chuck version {{.Version}}\n")
	root.AddCommand(newDashboardCmd())
	root.AddCommand(newReindexCmd())
	root.AddCommand(newResumeCmd())
	root.AddCommand(newReleaseCmd())
	root.AddCommand(newUpdateCmd())
	root.AddCommand(newAuthCmd())
	root.AddCommand(newPipelineCmd())
	operatorCommands := append(newOperatorManagementCommands(), newOperatorWorkflowCommands()...)
	operatorCommands = append(operatorCommands, newOperatorAgentCmd(), newOperatorStatusCmd())
	for _, cmd := range operatorCommands {
		markOperatorCommands(cmd)
		root.AddCommand(cmd)
	}
	return root
}

func markOperatorCommands(cmd *cobra.Command) {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations["operator"] = "true"
	if cmd.HasSubCommands() && cmd.PersistentFlags().Lookup("json") == nil {
		cmd.PersistentFlags().Bool("json", false, "write machine-readable diagnostics")
	}
	for _, child := range cmd.Commands() {
		markOperatorCommands(child)
	}
}

// newResumeCmd returns the `chuck resume <agent_id>` cobra command.
func newResumeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resume <agent_id>",
		Short: "Resume an inactive persisted session by agent_id",
		Args:  cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			os.Exit(runResumeByID(args[0]))
		},
	}
}

// Execute is the entrypoint called by cmd/chuck/main.go. It intercepts the
// reserved `<role>@<project>` launch syntax before cobra dispatch, then runs the
// command tree. Returns the process exit code.
func Execute(args []string) int {
	// Reserved launch syntax: first positional arg of the form role@project.
	if len(args) > 0 && isLaunchArg(args[0]) {
		return runLaunch(args)
	}

	root := NewRootCmd()
	root.SetArgs(args)
	root.SilenceErrors = true
	cmd, err := root.ExecuteC()
	if err != nil {
		if cmd != nil && (cmd.Annotations["operator"] == "true" || operatorMode(cmd)) {
			var op *operatorError
			if !errors.As(err, &op) {
				err = operatorInvalid(err.Error())
			}
			operatorDiagnostic(cmd, err)
			return operatorExitCode(err)
		}
		fmt.Fprintln(os.Stderr, "Error:", err)
		if errors.Is(err, errProviderUnavailable) {
			return 3
		}
		return 1
	}
	return 0
}

func operatorMode(cmd *cobra.Command) bool {
	json, _ := cmd.Flags().GetBool("json")
	request, _ := cmd.Flags().GetString("request-file")
	return json || request != "" || cmd.Flags().Changed("timeout")
}
