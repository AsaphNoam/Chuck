// Package cli wires the cobra command tree for the chuck binary.
package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/AsaphNoam/Chuck/internal/runtime/terminal"
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
	for _, cmd := range root.Commands() {
		rejectUnknownSubcommands(cmd)
	}
	return root
}

// noPositionalArgs replaces cobra.NoArgs, whose "unknown command" wording
// misreads a stray value as a subcommand name.
func noPositionalArgs(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return operatorInvalid(fmt.Sprintf("`%s` takes no positional arguments (got %q); name targets with flags, see `%s --help`", cmd.CommandPath(), args[0], cmd.CommandPath()))
	}
	return nil
}

// rejectUnknownSubcommands makes a command group refuse a mistyped
// subcommand instead of printing help and exiting 0; bare groups still print help.
func rejectUnknownSubcommands(cmd *cobra.Command) {
	if !cmd.HasSubCommands() {
		return
	}
	if !cmd.Runnable() {
		cmd.RunE = func(c *cobra.Command, args []string) error {
			if len(args) == 0 {
				return c.Help()
			}
			return operatorInvalid(fmt.Sprintf("unknown command %q for `%s`; see `%s --help`", args[0], c.CommandPath(), c.CommandPath()))
		}
	}
	for _, child := range cmd.Commands() {
		rejectUnknownSubcommands(child)
	}
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
	// Internal terminal bootstrap (TS-04.R102): exec'd by an owned tmux/iTerm2
	// view, never offered as a command. It returns only on failure.
	if len(args) == 2 && args[0] == terminal.BootstrapCommand {
		err := terminal.RunBootstrap(args[1])
		fmt.Fprintln(os.Stderr, "chuck: terminal launch failed:", err)
		return 1
	}
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
