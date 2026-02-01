package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/runoshun/git-crew/v2/internal/domain"
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
		Use:   "review [id] [agent] [-- message...]",
		Short: "Review task changes with AI",
		Long: `Review task changes using an AI reviewer agent.

The reviewer analyzes the diff and provides feedback on code quality,
correctness, and adherence to best practices.

The review runs synchronously and prints the result.

The review result is saved as a comment with author "reviewer".

Arguments:
  [id]      Task ID to review (optional, auto-detected from branch if omitted)
  [agent]   Reviewer agent name (optional, uses default if not specified)
  [-- message...]  Additional instructions for the reviewer (optional)

The message can also be provided via stdin.

Examples:
  # Run review (auto-detect task ID from current branch)
  crew review

  # Run review with explicit task ID
  crew review 1

  # Review with specific agent (auto-detect task ID)
  crew review claude-reviewer

  # Review with specific agent and explicit task ID
  crew review 1 claude-reviewer

  # Review with additional instructions
  crew review 1 -- "Focus on the last commit only"
  crew review 1 claude-reviewer -- "Check security aspects carefully"
  crew review -- "Focus on security"

  # Review with message from stdin
  echo "Focus on performance" | crew review 1

  # Review with verbose output (shows real-time reviewer output)
  crew review 1 --verbose`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse task ID, agent, and message from args
			// ArgsLenAtDash returns the count of positional args before "--" (after flag parsing),
			// or -1 if "--" was not present
			taskID, agent, message, err := parseReviewAllArgs(args, cmd.ArgsLenAtDash(), c.Git)
			if err != nil {
				return err
			}

			// Print start message early (before stdin read which can be slow)
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Reviewing task #%d...\n", taskID)

			// If no message from args, try to read from stdin (if not a terminal)
			if message == "" {
				stdinMessage, readErr := readStdinIfNotTerminal()
				if readErr != nil {
					return fmt.Errorf("read stdin: %w", readErr)
				}
				message = stdinMessage
			}

			// Execute use case
			uc := c.ReviewTaskUseCase(cmd.ErrOrStderr())
			out, err := uc.Execute(cmd.Context(), usecase.ReviewTaskInput{
				TaskID:  taskID,
				Agent:   agent,
				Model:   opts.model,
				Message: message,
				Verbose: opts.verbose,
			})
			if err != nil {
				return err
			}

			if out.Review != "" {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), out.Review)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.model, "model", "m", "", "Model to use (overrides agent default)")
	cmd.Flags().BoolVarP(&opts.verbose, "verbose", "v", false, "Show reviewer output in real-time")

	return cmd
}

// parseReviewAllArgs parses task ID, agent, and message from command line arguments.
// Format: [id] [agent] [-- message...]
// If the first argument is a valid task ID (numeric), it's used as the ID.
// Otherwise, it's treated as the agent name and the ID is resolved from the current branch.
//
// argsLenAtDash is the value from cmd.ArgsLenAtDash():
//   - -1 if "--" was not present
//   - N if "--" was present, where N is the number of args before "--"
//
// When "--" is present, Cobra sets argsLenAtDash to the count of args before the separator.
// The args slice contains all positional arguments (both before and after "--"), with the
// separator itself removed. We use argsLenAtDash to split args into positional and message parts.
//
// Returns task ID, agent name, message, and error.
func parseReviewAllArgs(args []string, argsLenAtDash int, git domain.Git) (taskID int, agent, message string, err error) {
	// Split args into positional args (before "--") and message args (after "--")
	var positionalArgs, messageArgs []string
	if argsLenAtDash == -1 {
		// No "--" present - all args are positional
		positionalArgs = args
	} else {
		// "--" was present - split at the boundary
		positionalArgs = args[:argsLenAtDash]
		messageArgs = args[argsLenAtDash:]
	}

	// Build message from args after "--"
	if len(messageArgs) > 0 {
		message = strings.Join(messageArgs, " ")
	}

	// Validate: at most 2 positional args are allowed ([id] [agent])
	// This catches typos like "crew review 1 claude-reviewer extra" or
	// "crew review 1 Focus on X" (missing "--" before message)
	if len(positionalArgs) > 2 {
		return 0, "", "", fmt.Errorf("too many arguments: expected at most 2 (id and agent), got %d; use \"--\" before message text", len(positionalArgs))
	}

	// No positional args - resolve task ID from branch
	if len(positionalArgs) == 0 {
		taskID, err = resolveTaskID(nil, git)
		return taskID, "", message, err
	}

	// Try to parse first positional arg as task ID
	first := positionalArgs[0]
	id, parseErr := parseTaskID(first)
	if parseErr == nil {
		// First arg is a valid task ID
		taskID = id
		if len(positionalArgs) > 1 {
			agent = positionalArgs[1]
		}
		return taskID, agent, message, nil
	}

	// Check if it looks like an ID but is invalid (e.g., "0", "#0", "-5")
	// This prevents accidental review of wrong task when user types invalid ID
	if looksLikeTaskID(first) {
		return 0, "", "", fmt.Errorf("invalid task ID: %s", first)
	}

	// First arg is not a valid task ID - treat it as agent name
	// and resolve task ID from branch
	agent = first
	taskID, err = resolveTaskID(nil, git)
	return taskID, agent, message, err
}

// looksLikeTaskID returns true if the string looks like a task ID attempt
// (starts with # or is purely numeric), even if it would be invalid.
func looksLikeTaskID(s string) bool {
	if strings.HasPrefix(s, "#") {
		return true
	}
	// Check if it's all digits (possibly with leading minus)
	trimmed := strings.TrimPrefix(s, "-")
	if len(trimmed) == 0 {
		return false
	}
	for _, c := range trimmed {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
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
