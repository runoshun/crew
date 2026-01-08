package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/runoshun/git-crew/v2/internal/usecase"
	"github.com/spf13/cobra"
)

// newReviewCommand creates the review command for reviewing task changes with AI.
func newReviewCommand(c *app.Container) *cobra.Command {
	var opts struct {
		model   string
		verbose bool
	}

	cmd := &cobra.Command{
		Use:   "review <id> [agent] [-- message...]",
		Short: "Review task changes with AI",
		Long: `Review task changes using an AI reviewer agent.

The reviewer analyzes the diff and provides feedback on code quality,
correctness, and adherence to best practices.

Arguments:
  <id>      Task ID to review
  [agent]   Reviewer agent name (optional, uses default if not specified)
  [-- message...]  Additional instructions for the reviewer (optional)

The message can also be provided via stdin.

Examples:
  # Basic review
  crew review 1

  # Review with specific agent
  crew review 1 claude-reviewer

  # Review with additional instructions
  crew review 1 -- "Focus on the last commit only"
  crew review 1 claude-reviewer -- "Check security aspects carefully"

  # Review with message from stdin
  echo "Focus on performance" | crew review 1`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse task ID
			taskID, err := parseTaskID(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			// Parse agent and message from remaining args
			agent, message := parseReviewArgs(args[1:])

			// If no message from args, try to read from stdin (if not a terminal)
			if message == "" {
				stdinMessage, readErr := readStdinIfNotTerminal()
				if readErr != nil {
					return fmt.Errorf("read stdin: %w", readErr)
				}
				message = stdinMessage
			}

			// Execute use case
			uc := c.ReviewTaskUseCase(cmd.OutOrStdout(), cmd.ErrOrStderr())
			_, err = uc.Execute(cmd.Context(), usecase.ReviewTaskInput{
				TaskID:  taskID,
				Agent:   agent,
				Model:   opts.model,
				Message: message,
				Verbose: opts.verbose,
			})
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.model, "model", "m", "", "Model to use (overrides agent default)")
	cmd.Flags().BoolVarP(&opts.verbose, "verbose", "v", false, "Show full output including intermediate steps")

	return cmd
}

// parseReviewArgs parses the agent and message from command line arguments.
// Format: [agent] [-- message...]
// Returns agent name (empty if not specified) and message (empty if not specified).
func parseReviewArgs(args []string) (agent, message string) {
	if len(args) == 0 {
		return "", ""
	}

	// Find "--" separator
	dashIdx := -1
	for i, arg := range args {
		if arg == "--" {
			dashIdx = i
			break
		}
	}

	if dashIdx == -1 {
		// No "--" found - first arg could be agent or nothing
		firstArg := args[0] //nolint:gosec // len(args) > 0 already checked above
		if !strings.HasPrefix(firstArg, "-") {
			return firstArg, ""
		}
		return "", ""
	}

	// Has "--" separator
	if dashIdx == 0 {
		// "--" is first - no agent, rest is message
		if len(args) > 1 {
			return "", strings.Join(args[1:], " ")
		}
		return "", ""
	}

	// Agent is before "--", message is after
	// dashIdx > 0 here, so args[0] is safe
	agent = args[0] //nolint:gosec // dashIdx > 0 ensures args[0] exists
	if len(args) > dashIdx+1 {
		message = strings.Join(args[dashIdx+1:], " ")
	}
	return agent, message
}

// readStdinIfNotTerminal reads from stdin if it's not a terminal (i.e., piped input).
func readStdinIfNotTerminal() (string, error) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return "", err
	}

	// Check if stdin is a terminal (character device)
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		// It's a terminal, don't read
		return "", nil
	}

	// It's piped input, read it
	var sb strings.Builder
	reader := bufio.NewReader(os.Stdin)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				sb.WriteString(line)
				break
			}
			return "", err
		}
		sb.WriteString(line)
	}

	return strings.TrimSpace(sb.String()), nil
}
