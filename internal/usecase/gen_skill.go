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
	Claude   bool // Generate only for .claude/ directory
	OpenCode bool // Generate only for .opencode/ directory
	Codex    bool // Generate only for .codex/ directory
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
func (uc *GenSkill) Execute(_ context.Context, in GenSkillInput) (*GenSkillOutput, error) {
	// Define all possible target directories
	allTargetDirs := map[string]string{
		"claude":   ".claude/skills/crew-manager",
		"codex":    ".codex/skills/crew-manager",
		"opencode": ".opencode/skill/crew-manager",
	}

	// Determine which directories to generate based on input flags
	var targetDirs []string
	if !in.Claude && !in.OpenCode && !in.Codex {
		// No flags specified, generate for all
		targetDirs = []string{
			allTargetDirs["claude"],
			allTargetDirs["codex"],
			allTargetDirs["opencode"],
		}
	} else {
		// Generate only for specified flags
		if in.Claude {
			targetDirs = append(targetDirs, allTargetDirs["claude"])
		}
		if in.Codex {
			targetDirs = append(targetDirs, allTargetDirs["codex"])
		}
		if in.OpenCode {
			targetDirs = append(targetDirs, allTargetDirs["opencode"])
		}
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
