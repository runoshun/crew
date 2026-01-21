package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/usecase"
	"github.com/spf13/cobra"
)

// newNewCommand creates the new command for creating tasks.
func newNewCommand(c *app.Container) *cobra.Command {
	var opts struct {
		Title       string
		Description string
		Base        string
		From        string
		Labels      []string
		ParentID    int
		Issue       int
		SkipReview  bool
		DryRun      bool
	}

	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new task",
		Long: `Create a new task for git-crew to manage.

The task is created with status 'todo'. The worktree and branch
are not created until the task is started with 'crew start <id> [agent]'.

Examples:
  # Create a root task
  crew new --title "Auth refactoring"

  # Create a sub-task under task #1
  crew new --parent 1 --title "OAuth2.0 implementation"

  # Create a task linked to a GitHub issue
  crew new --title "Fix login bug" --issue 42

  # Create a task with labels
  crew new --title "Add feature" --label feature --label urgent

  # Create a task with body using HEREDOC (recommended for complex descriptions)
  crew new --title "Complex task" --body "$(cat <<'EOF'
## Summary
- Step 1
- Step 2
EOF
)"

  # Create a task that skips review on completion
  crew new --title "Quick fix" --skip-review

  # Create tasks from a file (multiple tasks supported)
  crew new --from tasks.md

  # Preview tasks from a file without creating
  crew new --from tasks.md --dry-run

File format for --from:
  ---
  title: Task 1
  labels: [backend]
  ---
  Description here.

  ---
  title: Task 2
  parent: 1          # Relative: refers to Task 1 in this file
  ---

  ---
  title: Task 3
  parent: #123       # Absolute: refers to existing task #123
  ---`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Check if --from is specified
			if opts.From != "" {
				return createTasksFromFile(cmd, c, opts.From, opts.Base, opts.DryRun)
			}

			// Require --title when not using --from
			if opts.Title == "" {
				return fmt.Errorf("required flag(s) \"title\" not set")
			}

			// Build input
			// BaseBranch: if --base is provided, use it; otherwise empty (let UseCase decide)
			input := usecase.NewTaskInput{
				Title:       opts.Title,
				Description: opts.Description,
				Issue:       opts.Issue,
				Labels:      opts.Labels,
				BaseBranch:  opts.Base,
			}

			// Set parent ID if specified
			if opts.ParentID > 0 {
				input.ParentID = &opts.ParentID
			}

			// Set skip_review only if flag was explicitly provided
			if cmd.Flags().Changed("skip-review") {
				input.SkipReview = &opts.SkipReview
			}

			// Execute use case
			uc := c.NewTaskUseCase()
			out, err := uc.Execute(cmd.Context(), input)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created task #%d\n", out.TaskID)
			return nil
		},
	}

	// Flags (--title is conditionally required based on --from)
	cmd.Flags().StringVar(&opts.Title, "title", "", "Task title (required unless --from is used)")
	cmd.Flags().StringVar(&opts.Description, "body", "", "Task description")
	cmd.Flags().IntVar(&opts.ParentID, "parent", 0, "Parent task ID (creates a sub-task)")
	cmd.Flags().IntVar(&opts.Issue, "issue", 0, "Linked GitHub issue number")
	cmd.Flags().StringArrayVar(&opts.Labels, "label", nil, "Labels (can specify multiple)")
	cmd.Flags().StringVar(&opts.Base, "base", "", "Base branch for worktree (default: current branch)")
	cmd.Flags().BoolVar(&opts.SkipReview, "skip-review", false, "Skip review on task completion (go directly to reviewed)")
	cmd.Flags().StringVar(&opts.From, "from", "", "Create tasks from a Markdown file")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Preview tasks without creating (requires --from)")

	return cmd
}

// createTasksFromFile creates tasks from a Markdown file.
func createTasksFromFile(cmd *cobra.Command, c *app.Container, filePath, baseBranch string, dryRun bool) error {
	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	// Execute use case
	uc := c.CreateTasksFromFileUseCase()
	out, err := uc.Execute(cmd.Context(), usecase.CreateTasksFromFileInput{
		Content:    string(content),
		BaseBranch: baseBranch,
		DryRun:     dryRun,
	})
	if err != nil {
		return err
	}

	// Print results
	w := cmd.OutOrStdout()
	if dryRun {
		_, _ = fmt.Fprintln(w, "Dry run - tasks that would be created:")
		_, _ = fmt.Fprintln(w, "")
	}

	for i, task := range out.Tasks {
		if dryRun {
			_, _ = fmt.Fprintf(w, "Task %d:\n", i+1)
		} else {
			_, _ = fmt.Fprintf(w, "Created task #%d:\n", task.ID)
		}
		_, _ = fmt.Fprintf(w, "  Title: %s\n", task.Title)
		if task.ParentID != nil {
			if dryRun {
				// In dry-run, show if it's a relative reference
				if *task.ParentID <= len(out.Tasks) {
					_, _ = fmt.Fprintf(w, "  Parent: task %d (in this file)\n", *task.ParentID)
				} else {
					_, _ = fmt.Fprintf(w, "  Parent: #%d\n", *task.ParentID)
				}
			} else {
				_, _ = fmt.Fprintf(w, "  Parent: #%d\n", *task.ParentID)
			}
		}
		if len(task.Labels) > 0 {
			_, _ = fmt.Fprintf(w, "  Labels: [%s]\n", strings.Join(task.Labels, ", "))
		}
		if task.Description != "" {
			// Show first line of description
			lines := strings.Split(task.Description, "\n")
			preview := lines[0]
			if len(preview) > 50 {
				preview = preview[:50] + "..."
			}
			if len(lines) > 1 {
				preview += " ..."
			}
			_, _ = fmt.Fprintf(w, "  Description: %s\n", preview)
		}
		if i < len(out.Tasks)-1 {
			_, _ = fmt.Fprintln(w, "")
		}
	}

	if !dryRun {
		_, _ = fmt.Fprintf(w, "\nCreated %d task(s)\n", len(out.Tasks))
	}

	return nil
}

// newListCommand creates the list command for listing tasks.
func newListCommand(c *app.Container) *cobra.Command {
	var opts struct {
		Labels    []string
		ParentID  int
		All       bool
		Sessions  bool
		Processes bool
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks",
		Long: `Display a list of tasks.

By default, tasks with terminal status (closed) are hidden.
Use --all to show all tasks including closed tasks.

Output format is tab-separated with columns:
  ID, PARENT, STATUS, AGENT, LABELS, [ELAPSED], TITLE

ELAPSED is only shown for tasks with status 'in_progress'.

With --sessions (-s), SESSION column is added showing the session name.
With --processes (-p), process details are shown instead of the task list.

Examples:
  # List active tasks (default: exclude closed)
  crew list

  # List all tasks including closed
  crew list --all
  crew list -a

  # List with session information
  crew list -s
  crew list --sessions

  # List with process information
  crew list -p
  crew list --processes

  # List only sub-tasks of task #1
  crew list --parent 1

  # List tasks with specific labels
  crew list --label bug --label urgent`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Build input
			input := usecase.ListTasksInput{
				Labels:           opts.Labels,
				IncludeTerminal:  opts.All,
				IncludeSessions:  opts.Sessions,
				IncludeProcesses: opts.Processes,
			}

			// Set parent ID if specified
			if opts.ParentID > 0 {
				input.ParentID = &opts.ParentID
			}

			// Execute use case
			uc := c.ListTasksUseCase()
			out, err := uc.Execute(cmd.Context(), input)
			if err != nil {
				return err
			}

			// Print output
			if opts.Processes {
				printProcessList(cmd.OutOrStdout(), out.TasksWithInfo)
			} else if opts.Sessions {
				printTaskListWithSessions(cmd.OutOrStdout(), out.TasksWithInfo, c.Clock)
			} else {
				printTaskList(cmd.OutOrStdout(), out.Tasks, c.Clock)
			}
			return nil
		},
	}

	// Optional flags
	cmd.Flags().IntVar(&opts.ParentID, "parent", 0, "Show only children of this task")
	cmd.Flags().StringArrayVar(&opts.Labels, "label", nil, "Filter by labels (AND condition)")
	cmd.Flags().BoolVarP(&opts.All, "all", "a", false, "Show all tasks including closed")
	cmd.Flags().BoolVarP(&opts.Sessions, "sessions", "s", false, "Include session information")
	cmd.Flags().BoolVarP(&opts.Processes, "processes", "p", false, "Show process details")

	return cmd
}

// printTaskList prints tasks in TSV format.
func printTaskList(w io.Writer, tasks []*domain.Task, clock domain.Clock) {
	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	defer func() { _ = tw.Flush() }()

	// Header
	_, _ = fmt.Fprintln(tw, "ID\tPARENT\tSTATUS\tAGENT\tLABELS\tTITLE")

	// Rows
	for _, task := range tasks {
		parentStr := "-"
		if task.ParentID != nil {
			parentStr = fmt.Sprintf("%d", *task.ParentID)
		}

		agentStr := "-"
		if task.Agent != "" {
			agentStr = task.Agent
		}

		labelsStr := "-"
		if len(task.Labels) > 0 {
			labelsStr = "[" + strings.Join(task.Labels, ",") + "]"
		}

		// Format status with optional elapsed time for in_progress
		statusStr := string(task.Status)
		if task.Status == domain.StatusInProgress && !task.Started.IsZero() {
			elapsed := clock.Now().Sub(task.Started)
			statusStr = fmt.Sprintf("%s (%s)", task.Status, formatDuration(elapsed))
		}

		_, _ = fmt.Fprintf(tw, "%d\t%s\t%s\t%s\t%s\t%s\n",
			task.ID,
			parentStr,
			statusStr,
			agentStr,
			labelsStr,
			task.Title,
		)
	}
}

// printTaskListWithSessions prints tasks with session information in TSV format.
func printTaskListWithSessions(w io.Writer, tasksWithInfo []usecase.TaskWithSession, clock domain.Clock) {
	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	defer func() { _ = tw.Flush() }()

	// Header
	_, _ = fmt.Fprintln(tw, "ID\tPARENT\tSTATUS\tAGENT\tSESSION\tLABELS\tTITLE")

	// Rows
	for _, info := range tasksWithInfo {
		task := info.Task
		parentStr := "-"
		if task.ParentID != nil {
			parentStr = fmt.Sprintf("%d", *task.ParentID)
		}

		agentStr := "-"
		if task.Agent != "" {
			agentStr = task.Agent
		}

		sessionStr := "-"
		if info.IsRunning {
			sessionStr = info.SessionName
		}

		labelsStr := "-"
		if len(task.Labels) > 0 {
			labelsStr = "[" + strings.Join(task.Labels, ",") + "]"
		}

		// Format status with optional elapsed time for in_progress
		statusStr := string(task.Status)
		if task.Status == domain.StatusInProgress && !task.Started.IsZero() {
			elapsed := clock.Now().Sub(task.Started)
			statusStr = fmt.Sprintf("%s (%s)", task.Status, formatDuration(elapsed))
		}

		_, _ = fmt.Fprintf(tw, "%d\t%s\t%s\t%s\t%s\t%s\t%s\n",
			task.ID,
			parentStr,
			statusStr,
			agentStr,
			sessionStr,
			labelsStr,
			task.Title,
		)
	}
}

// printProcessList prints process details in TSV format.
func printProcessList(w io.Writer, tasksWithInfo []usecase.TaskWithSession) {
	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	defer func() { _ = tw.Flush() }()

	// Header
	_, _ = fmt.Fprintln(tw, "ID\tSESSION\tPID\tSTATE\tPROCESS")

	// Rows
	for _, info := range tasksWithInfo {
		task := info.Task

		if !info.IsRunning || len(info.Processes) == 0 {
			// No running session or no processes
			continue
		}

		// Print first process
		first := info.Processes[0]
		_, _ = fmt.Fprintf(tw, "%d\t%s\t%d\t%s\t%s\n",
			task.ID,
			info.SessionName,
			first.PID,
			first.State,
			first.Command,
		)

		// Print child processes with indentation
		for i := 1; i < len(info.Processes); i++ {
			proc := info.Processes[i]
			_, _ = fmt.Fprintf(tw, "\t\t%d\t%s\t└─ %s\n",
				proc.PID,
				proc.State,
				proc.Command,
			)
		}
	}
}

// formatDuration formats a duration in a human-readable way.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

// newShowCommand creates the show command for displaying task details.
func newShowCommand(c *app.Container) *cobra.Command {
	var opts struct {
		CommentsBy string
		JSON       bool
		LastReview bool
	}

	cmd := &cobra.Command{
		Use:   "show [id]",
		Short: "Display task details",
		Long: `Display detailed information about a task.

If no ID is provided, the task ID is auto-detected from the current branch name.
The branch must follow the naming convention: crew-<id> or crew-<id>-gh-<issue>

Output includes:
  - Task ID and title
  - Description
  - Status, parent, branch, labels
  - GitHub issue and PR numbers (if linked)
  - Running agent and session info
  - Sub-tasks (if any)
  - Comments (if any)

Examples:
  # Show task by ID
  crew show 1

  # Auto-detect task from current branch
  crew show

  # Output in JSON format
  crew show 1 --json

  # Show only the latest reviewer comment
  crew show 1 --last-review

  # Show comments by a specific author
  crew show 1 --comments-by reviewer`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Resolve task ID
			taskID, err := resolveTaskID(args, c.Git)
			if err != nil {
				return err
			}

			// Execute use case
			uc := c.ShowTaskUseCase()
			out, err := uc.Execute(cmd.Context(), usecase.ShowTaskInput{
				TaskID:     taskID,
				CommentsBy: opts.CommentsBy,
				LastReview: opts.LastReview,
			})
			if err != nil {
				return err
			}

			// Print output
			if opts.JSON {
				type jsonComment struct {
					Time   time.Time `json:"time"`
					Text   string    `json:"text"`
					Author string    `json:"author,omitempty"`
				}
				type jsonTask struct {
					Created     time.Time     `json:"created"`
					Started     *time.Time    `json:"started,omitempty"`
					ParentID    *int          `json:"parent_id"`
					Description string        `json:"description"`
					Agent       string        `json:"agent"`
					Branch      string        `json:"branch"`
					Status      domain.Status `json:"status"`
					Title       string        `json:"title"`
					Labels      []string      `json:"labels"`
					Comments    []jsonComment `json:"comments"`
					ID          int           `json:"id"`
					Issue       int           `json:"issue"`
				}

				jt := jsonTask{
					Created:     out.Task.Created,
					ParentID:    out.Task.ParentID,
					Description: out.Task.Description,
					Agent:       out.Task.Agent,
					Branch:      domain.BranchName(out.Task.ID, out.Task.Issue),
					Status:      out.Task.Status,
					Title:       out.Task.Title,
					Labels:      out.Task.Labels,
					ID:          out.Task.ID,
					Issue:       out.Task.Issue,
					Comments:    make([]jsonComment, len(out.Comments)),
				}
				if !out.Task.Started.IsZero() {
					jt.Started = &out.Task.Started
				}
				if jt.Labels == nil {
					jt.Labels = []string{}
				}
				for i, c := range out.Comments {
					jt.Comments[i] = jsonComment{
						Time:   c.Time,
						Text:   c.Text,
						Author: c.Author,
					}
				}

				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(jt)
			}

			printTaskDetails(cmd.OutOrStdout(), out)
			return nil
		},
	}

	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output in JSON format")
	cmd.Flags().StringVar(&opts.CommentsBy, "comments-by", "", "Filter comments by author")
	cmd.Flags().BoolVar(&opts.LastReview, "last-review", false, "Show only the latest review comment")

	return cmd
}

// resolveTaskID resolves the task ID from arguments or current branch.
func resolveTaskID(args []string, git domain.Git) (int, error) {
	if len(args) > 0 {
		// Parse from argument
		id, err := parseTaskID(args[0])
		if err != nil {
			return 0, fmt.Errorf("invalid task ID: %w", err)
		}
		return id, nil
	}

	// Auto-detect from branch
	if git == nil {
		return 0, fmt.Errorf("task ID is required (not on a crew branch)")
	}

	branch, err := git.CurrentBranch()
	if err != nil {
		return 0, fmt.Errorf("failed to detect current branch: %w", err)
	}

	id, ok := domain.ParseBranchTaskID(branch)
	if !ok {
		return 0, fmt.Errorf("task ID is required (current branch '%s' is not a crew branch)", branch)
	}

	return id, nil
}

// parseTaskID parses a task ID string to int.
func parseTaskID(s string) (int, error) {
	// Remove leading # if present
	s = strings.TrimPrefix(s, "#")
	var id int
	_, err := fmt.Sscanf(s, "%d", &id)
	if err != nil {
		return 0, err
	}
	if id <= 0 {
		return 0, fmt.Errorf("task ID must be positive")
	}
	return id, nil
}

// newEditCommand creates the edit command for editing task information.
func newEditCommand(c *app.Container) *cobra.Command {
	var opts struct {
		Title        string
		Description  string
		Status       string
		Labels       string
		From         string
		AddLabels    []string
		RemoveLabels []string
		IfStatus     []string
		ParentID     int
		SkipReview   bool
		NoSkipReview bool
		NoParent     bool
	}

	cmd := &cobra.Command{
		Use:   "edit <id>",
		Short: "Edit task information",
		Long: `Edit an existing task's title, description, status, or labels.

If no flags are provided, the task is opened in the user's $EDITOR for editing
title and description using a Markdown format with frontmatter.

If any flags are provided (--title, --body, --status, --labels, --add-label, --rm-label),
the task is updated directly without opening an editor.

Examples:
  # Open task in editor
  crew edit 1

  # Change task title
  crew edit 1 --title "New task title"

  # Update description
  crew edit 1 --body "Updated description text"

  # Change task status
  crew edit 1 --status for_review

  # Conditional status change (only if current status matches)
  crew edit 1 --status needs_input --if-status in_progress

  # Multiple conditions (status change only if current status is one of these)
  crew edit 1 --status needs_input --if-status in_progress --if-status needs_input

  # Replace all labels (comma-separated)
  crew edit 1 --labels bug,urgent

  # Clear all labels
  crew edit 1 --labels ""

  # Add labels
  crew edit 1 --add-label bug --add-label urgent

  # Remove labels
  crew edit 1 --rm-label old-label

  # Multiple changes at once
  crew edit 1 --title "New title" --add-label feature --rm-label draft

  # Enable skip_review for a task
  crew edit 1 --skip-review

  # Disable skip_review for a task
  crew edit 1 --no-skip-review

  # Set parent task
  crew edit 1 --parent 5

  # Remove parent (make it a root task)
  crew edit 1 --no-parent
  crew edit 1 --parent 0

  # Edit task from a file (updates title, body, and labels)
  crew edit 1 --from task.md

File format for --from:
  ---
  title: New Task Title
  labels: [backend, feature]
  ---
  New task description here.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse task ID
			taskID, err := parseTaskID(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			// Check if --from is specified
			if opts.From != "" {
				return editTaskFromFile(cmd, c, taskID, opts.From)
			}

			// Check if any flags are provided
			hasFlags := cmd.Flags().Changed("title") ||
				cmd.Flags().Changed("body") ||
				cmd.Flags().Changed("status") ||
				cmd.Flags().Changed("labels") ||
				len(opts.AddLabels) > 0 ||
				len(opts.RemoveLabels) > 0 ||
				len(opts.IfStatus) > 0 ||
				cmd.Flags().Changed("skip-review") ||
				cmd.Flags().Changed("no-skip-review") ||
				cmd.Flags().Changed("parent") ||
				cmd.Flags().Changed("no-parent")

			if !hasFlags {
				// Editor mode: open task in editor
				return editTaskWithEditor(cmd, c, taskID)
			}

			// Flag mode: update task directly
			// Build input
			input := usecase.EditTaskInput{
				TaskID:       taskID,
				AddLabels:    opts.AddLabels,
				RemoveLabels: opts.RemoveLabels,
			}

			// Set optional fields only if provided
			if cmd.Flags().Changed("title") {
				input.Title = &opts.Title
			}
			if cmd.Flags().Changed("body") {
				input.Description = &opts.Description
			}
			if cmd.Flags().Changed("status") {
				status := domain.Status(opts.Status)
				input.Status = &status
			}
			if cmd.Flags().Changed("labels") {
				input.LabelsSet = true
				if opts.Labels != "" {
					input.Labels = strings.Split(opts.Labels, ",")
				}
			}
			if len(opts.IfStatus) > 0 {
				statuses := make([]domain.Status, len(opts.IfStatus))
				for i, s := range opts.IfStatus {
					statuses[i] = domain.Status(s)
				}
				input.IfStatus = statuses
			}
			if cmd.Flags().Changed("skip-review") {
				v := true
				input.SkipReview = &v
			}
			if cmd.Flags().Changed("no-skip-review") {
				v := false
				input.SkipReview = &v
			}
			if cmd.Flags().Changed("no-parent") {
				input.RemoveParent = true
			} else if cmd.Flags().Changed("parent") {
				input.ParentID = &opts.ParentID
			}

			// Execute use case
			uc := c.EditTaskUseCase()
			_, err = uc.Execute(cmd.Context(), input)
			if err != nil {
				return err
			}

			return nil
		},
	}

	// Optional flags
	cmd.Flags().StringVar(&opts.Title, "title", "", "New task title")
	cmd.Flags().StringVar(&opts.Description, "body", "", "New task description")
	cmd.Flags().StringVar(&opts.Status, "status", "", "New task status (todo, in_progress, needs_input, for_review, reviewing, reviewed, stopped, error, closed)")
	cmd.Flags().StringArrayVar(&opts.IfStatus, "if-status", nil, "Only update status if current status matches (can specify multiple)")
	cmd.Flags().StringVar(&opts.Labels, "labels", "", "Replace all labels (comma-separated, empty string clears all)")
	cmd.Flags().StringArrayVar(&opts.AddLabels, "add-label", nil, "Labels to add (can specify multiple)")
	cmd.Flags().StringArrayVar(&opts.RemoveLabels, "rm-label", nil, "Labels to remove (can specify multiple)")
	cmd.Flags().BoolVar(&opts.SkipReview, "skip-review", false, "Enable skip_review for this task (skip review on completion)")
	cmd.Flags().BoolVar(&opts.NoSkipReview, "no-skip-review", false, "Disable skip_review for this task (require review on completion)")
	cmd.MarkFlagsMutuallyExclusive("skip-review", "no-skip-review")
	cmd.Flags().IntVar(&opts.ParentID, "parent", 0, "Set parent task ID (0 to remove parent)")
	cmd.Flags().BoolVar(&opts.NoParent, "no-parent", false, "Remove parent task (make this a root task)")
	cmd.MarkFlagsMutuallyExclusive("parent", "no-parent")
	cmd.Flags().StringVar(&opts.From, "from", "", "Edit task from a Markdown file (updates title, body, and labels)")

	return cmd
}

// editTaskWithEditor opens the task in an editor for editing.
func editTaskWithEditor(cmd *cobra.Command, c *app.Container, taskID int) error {
	// Get current task with comments
	showUC := c.ShowTaskUseCase()
	showOut, err := showUC.Execute(cmd.Context(), usecase.ShowTaskInput{
		TaskID: taskID,
	})
	if err != nil {
		return err
	}

	task := showOut.Task

	// Create temporary file with task content
	tmpFile, err := os.CreateTemp("", fmt.Sprintf("crew-task-%d-*.md", taskID))
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	// Write task as markdown with comments
	markdown := task.ToMarkdownWithComments(showOut.Comments)
	if _, writeErr := tmpFile.WriteString(markdown); writeErr != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to write temp file: %w", writeErr)
	}
	if closeErr := tmpFile.Close(); closeErr != nil {
		return fmt.Errorf("failed to close temp file: %w", closeErr)
	}

	// Open editor
	if editorErr := openEditor(tmpPath, c.Executor); editorErr != nil {
		return editorErr
	}

	// Read edited content
	editedContent, err := os.ReadFile(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to read edited file: %w", err)
	}

	// Check if content changed
	if string(editedContent) == markdown {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No changes made")
		return nil
	}

	// Update task with edited content
	input := usecase.EditTaskInput{
		TaskID:     taskID,
		EditorEdit: true,
		EditorText: string(editedContent),
	}

	uc := c.EditTaskUseCase()
	_, err = uc.Execute(cmd.Context(), input)
	if err != nil {
		return err
	}

	return nil
}

// editTaskFromFile updates a task from a Markdown file.
func editTaskFromFile(cmd *cobra.Command, c *app.Container, taskID int, filePath string) error {
	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	// Parse single task from file
	draft, err := domain.ParseSingleTaskDraft(string(content))
	if err != nil {
		return err
	}

	// Build input
	input := usecase.EditTaskInput{
		TaskID:      taskID,
		Title:       &draft.Title,
		Description: &draft.Description,
	}

	// Set labels if present in the file
	if draft.Labels != nil {
		input.LabelsSet = true
		input.Labels = draft.Labels
	}

	// Execute use case
	uc := c.EditTaskUseCase()
	_, err = uc.Execute(cmd.Context(), input)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Updated task #%d from %s\n", taskID, filePath)
	return nil
}

// newCpCommand creates the cp command for copying tasks.
func newCpCommand(c *app.Container) *cobra.Command {
	var opts struct {
		Title string
	}

	cmd := &cobra.Command{
		Use:   "cp <id>",
		Short: "Copy a task",
		Long: `Copy a task to create a new task.

The new task copies the title (with " (copy)" suffix by default),
description, labels, and parent reference.

The new task does NOT copy: issue, PR, comments.
The base branch is set to the source task's branch name.

Examples:
  # Copy task with default title
  crew cp 1

  # Copy task with custom title
  crew cp 1 --title "New feature based on #1"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse task ID
			taskID, err := parseTaskID(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			// Build input
			input := usecase.CopyTaskInput{
				SourceID: taskID,
			}

			// Set title if provided
			if cmd.Flags().Changed("title") {
				input.Title = &opts.Title
			}

			// Execute use case
			uc := c.CopyTaskUseCase()
			out, err := uc.Execute(cmd.Context(), input)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Copied task #%d to #%d\n", taskID, out.TaskID)
			return nil
		},
	}

	// Optional flags
	cmd.Flags().StringVar(&opts.Title, "title", "", "Custom title for the new task")

	return cmd
}

// newRmCommand creates the rm command for deleting tasks.
func newRmCommand(c *app.Container) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rm <id>",
		Short: "Delete a task",
		Long: `Delete a task from git-crew.

This removes the task from the store. In Phase 2, this does not
clean up worktrees or sessions - that will be added in later phases.

Examples:
  # Delete task by ID
  crew rm 1

  # Delete task using # prefix
  crew rm "#1"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse task ID
			taskID, err := parseTaskID(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			// Execute use case
			uc := c.DeleteTaskUseCase()
			_, err = uc.Execute(cmd.Context(), usecase.DeleteTaskInput{
				TaskID: taskID,
			})
			if err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}

// newCommentCommand creates the comment command for adding comments to tasks.
func newCommentCommand(c *app.Container) *cobra.Command {
	var opts struct {
		Author         string
		Edit           int
		RequestChanges bool
	}

	cmd := &cobra.Command{
		Use:   "comment <id> <message>",
		Short: "Add a comment to a task",
		Long: `Add a comment to a task.

Comments are timestamped and displayed in the task details output.
They can be used to track progress, notes, or any relevant information.

Examples:
  # Add a comment to task #1
  crew comment 1 "Started working on authentication"

  # Request changes (comment + status change + notification)
  crew comment 1 "修正してください" --request-changes
  crew comment 1 "修正してください" -R

  # Edit an existing comment (index starts from 0)
  crew comment 1 --edit 0 "Updated message"

  # Use with task ID prefix
  crew comment "#1" "Completed initial implementation"`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse task ID
			taskID, err := parseTaskID(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			if cmd.Flags().Changed("edit") {
				// Execute edit comment use case
				uc := c.EditCommentUseCase()
				err = uc.Execute(cmd.Context(), usecase.EditCommentInput{
					TaskID:  taskID,
					Index:   opts.Edit,
					Message: args[1],
				})
				if err != nil {
					return err
				}

				return nil
			}
			// Execute add comment use case
			uc := c.AddCommentUseCase()
			out, err := uc.Execute(cmd.Context(), usecase.AddCommentInput{
				TaskID:         taskID,
				Message:        args[1],
				Author:         opts.Author,
				RequestChanges: opts.RequestChanges,
			})
			if err != nil {
				return err
			}

			if opts.RequestChanges {
				if out.SessionStarted {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Added comment and started session for task #%d\n", taskID)
				} else {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Added comment and requested changes for task #%d\n", taskID)
				}
			} else {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Added comment to task #%d at %s\n",
					taskID, out.Comment.Time.Format(time.RFC3339))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.Author, "author", "", "Author name (manager, worker, etc.)")
	cmd.Flags().IntVar(&opts.Edit, "edit", -1, "Edit an existing comment by index")
	cmd.Flags().BoolVarP(&opts.RequestChanges, "request-changes", "R", false, "Request changes and update status to in_progress")

	return cmd
}

// newCloseCommand creates the close command for closing tasks.
func newCloseCommand(c *app.Container) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "close <id>",
		Short: "Close task without merging",
		Long: `Close a task without merging it.

This command will:
1. Stop any running session for the task
2. Delete the task's worktree if it exists
3. Transition the task status to 'closed'

The task will remain in the task list but will not be merged.

Examples:
  # Close task by ID
  crew close 1

  # Close task using # prefix
  crew close "#1"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse task ID
			taskID, err := parseTaskID(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			// Execute use case
			uc := c.CloseTaskUseCase()
			_, err = uc.Execute(cmd.Context(), usecase.CloseTaskInput{
				TaskID: taskID,
			})
			if err != nil {
				return err
			}

			return nil
		},
	}
	return cmd
}

// printTaskDetails prints task details in a formatted output.
func printTaskDetails(w io.Writer, out *usecase.ShowTaskOutput) {
	task := out.Task

	// Header
	_, _ = fmt.Fprintf(w, "# Task %d: %s\n\n", task.ID, task.Title)

	// Description
	if task.Description != "" {
		_, _ = fmt.Fprintf(w, "%s\n\n", task.Description)
	}

	// Fields
	_, _ = fmt.Fprintf(w, "Status: %s\n", task.Status)

	if task.ParentID != nil {
		_, _ = fmt.Fprintf(w, "Parent: #%d\n", *task.ParentID)
	} else {
		_, _ = fmt.Fprintln(w, "Parent: none")
	}

	_, _ = fmt.Fprintf(w, "Branch: %s\n", domain.BranchName(task.ID, task.Issue))

	if len(task.Labels) > 0 {
		_, _ = fmt.Fprintf(w, "Labels: [%s]\n", strings.Join(task.Labels, ", "))
	} else {
		_, _ = fmt.Fprintln(w, "Labels: none")
	}

	_, _ = fmt.Fprintf(w, "Created: %s\n", task.Created.Format(time.RFC3339))

	if task.Issue > 0 {
		_, _ = fmt.Fprintf(w, "Issue: #%d\n", task.Issue)
	}

	if task.PR > 0 {
		_, _ = fmt.Fprintf(w, "PR: #%d\n", task.PR)
	}

	if task.Agent != "" {
		_, _ = fmt.Fprintf(w, "Agent: %s (session: %s)\n", task.Agent, task.Session)
	}

	// Sub-tasks
	if len(out.Children) > 0 {
		_, _ = fmt.Fprintln(w, "\nSub-tasks:")
		for _, child := range out.Children {
			_, _ = fmt.Fprintf(w, "  #%d [%s] %s\n", child.ID, child.Status, child.Title)
		}
	}

	// Comments
	if len(out.Comments) > 0 {
		_, _ = fmt.Fprintln(w, "\nComments:")
		separator := "  ─────────────────"
		for _, comment := range out.Comments {
			_, _ = fmt.Fprintln(w, separator)
			authorPart := ""
			if comment.Author != "" {
				authorPart = " " + comment.Author
			}
			_, _ = fmt.Fprintf(w, "  [%s]%s\n", comment.Time.Format(time.RFC3339), authorPart)
			// Indent message
			lines := strings.Split(strings.TrimSpace(comment.Text), "\n")
			for _, line := range lines {
				_, _ = fmt.Fprintf(w, "  %s\n", line)
			}
		}
	}
}
