// Package usecase contains the application use cases.
package usecase

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed skill_crew_manager.md
var crewManagerSkillContent string

// GenSkillInput contains the input for the GenSkill use case.
type GenSkillInput struct {
	// Currently no input needed
}

// GenSkillOutput contains the output of the GenSkill use case.
type GenSkillOutput struct {
	CreatedPaths []string // Paths to the created skill files
}

// GenSkill generates crew-manager skill files.
type GenSkill struct {
	repoRoot string
}

// NewGenSkill creates a new GenSkill use case.
func NewGenSkill(repoRoot string) *GenSkill {
	return &GenSkill{
		repoRoot: repoRoot,
	}
}

// Execute creates crew-manager skill files in .claude, .codex, and .opencode directories.
func (uc *GenSkill) Execute(_ context.Context, _ GenSkillInput) (*GenSkillOutput, error) {
	// Define target directories
	targetDirs := []string{
		".claude/skills/crew-manager",
		".codex/skills/crew-manager",
		".opencode/skills/crew-manager",
	}

	createdPaths := make([]string, 0, len(targetDirs))

	for _, dir := range targetDirs {
		fullDir := filepath.Join(uc.repoRoot, dir)
		skillPath := filepath.Join(fullDir, "SKILL.md")

		// Create directory if it doesn't exist
		if err := os.MkdirAll(fullDir, 0750); err != nil {
			return nil, fmt.Errorf("create directory %s: %w", fullDir, err)
		}

		// Write skill file
		if err := os.WriteFile(skillPath, []byte(crewManagerSkillContent), 0600); err != nil {
			return nil, fmt.Errorf("write skill file %s: %w", skillPath, err)
		}

		createdPaths = append(createdPaths, skillPath)
	}

	return &GenSkillOutput{
		CreatedPaths: createdPaths,
	}, nil
}
