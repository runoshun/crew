package cli

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/usecase"
	"github.com/spf13/cobra"
)

// newCommentsCommand creates the comments command for listing task comments.
func newCommentsCommand(c *app.Container) *cobra.Command {
	var opts struct {
		Type    string
		TagsCSV string
		Tags    []string
	}

	cmd := &cobra.Command{
		Use:   "comments",
		Short: "List comments across tasks",
		Long: `List comments across tasks.

Examples:
  # List all friction comments
  crew comments --type friction

  # List suggestion comments with a specific tag
  crew comments --type suggestion --tag architecture

  # List comments with multiple tags (comma-separated)
  crew comments --tags testing,refactoring`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			commentType := domain.CommentType(strings.TrimSpace(strings.ToLower(opts.Type)))
			if !commentType.IsValid() {
				return fmt.Errorf("invalid comment type: %q: %w", opts.Type, domain.ErrInvalidCommentType)
			}
			tags := parseCommentTags(opts.Tags, opts.TagsCSV)

			uc := c.ListCommentsUseCase()
			out, err := uc.Execute(cmd.Context(), usecase.ListCommentsInput{
				Type:          commentType,
				Tags:          tags,
				AllNamespaces: true,
			})
			if err != nil {
				return err
			}

			return printCommentList(cmd.OutOrStdout(), out.Comments)
		},
	}

	cmd.Flags().StringVar(&opts.Type, "type", "", "Comment type (report, message, suggestion, friction)")
	cmd.Flags().StringArrayVar(&opts.Tags, "tag", nil, "Filter by tag (can specify multiple)")
	cmd.Flags().StringVar(&opts.TagsCSV, "tags", "", "Filter by tags (comma-separated)")

	return cmd
}

func printCommentList(w io.Writer, comments []usecase.CommentWithTask) error {
	for i, entry := range comments {
		if i > 0 {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(w, "---"); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
		ref, title := formatCommentTask(entry.Task)
		if _, err := fmt.Fprintf(w, "# Task: %s", ref); err != nil {
			return err
		}
		if title != "" {
			if _, err := fmt.Fprintf(w, " %s", title); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "# Comment: %d\n", entry.Index); err != nil {
			return err
		}
		if entry.Comment.Author != "" {
			if _, err := fmt.Fprintf(w, "# Author: %s\n", entry.Comment.Author); err != nil {
				return err
			}
		}
		if entry.Comment.Type != domain.CommentTypeGeneral {
			if _, err := fmt.Fprintf(w, "# Type: %s\n", entry.Comment.Type); err != nil {
				return err
			}
		}
		if len(entry.Comment.Tags) > 0 {
			if _, err := fmt.Fprintf(w, "# Tags: %s\n", strings.Join(entry.Comment.Tags, ", ")); err != nil {
				return err
			}
		}
		if len(entry.Comment.Metadata) > 0 {
			metadata := formatCommentMetadata(entry.Comment.Metadata)
			if metadata != "" {
				if _, err := fmt.Fprintf(w, "# Metadata: %s\n", metadata); err != nil {
					return err
				}
			}
		}
		if _, err := fmt.Fprintf(w, "# Time: %s\n\n", entry.Comment.Time.Format(time.RFC3339)); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, entry.Comment.Text); err != nil {
			return err
		}
	}
	return nil
}

func formatCommentTask(task *domain.Task) (string, string) {
	if task == nil {
		return "-", ""
	}
	namespace := task.Namespace
	if namespace == "" {
		namespace = "-"
	}
	return fmt.Sprintf("%s#%d", namespace, task.ID), task.Title
}

func formatCommentMetadata(metadata map[string]string) string {
	if len(metadata) == 0 {
		return ""
	}
	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+metadata[key])
	}
	return strings.Join(parts, ", ")
}
