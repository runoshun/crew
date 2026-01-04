package usecase

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenSkill_Execute(t *testing.T) {
	// Create temporary directory for testing
	tmpDir := t.TempDir()

	// Create use case
	uc := NewGenSkill(tmpDir)

	// Execute
	out, err := uc.Execute(context.Background(), GenSkillInput{})

	// Assert no error
	require.NoError(t, err)

	// Assert output contains 3 paths
	require.Len(t, out.CreatedPaths, 3)

	// Check that files were created
	expectedDirs := []string{
		".claude/skills/crew-manager",
		".codex/skills/crew-manager",
		".opencode/skill/crew-manager",
	}

	for i, dir := range expectedDirs {
		skillPath := filepath.Join(tmpDir, dir, "SKILL.md")

		// Check path is in output
		assert.Equal(t, skillPath, out.CreatedPaths[i])

		// Check file exists
		_, err := os.Stat(skillPath)
		require.NoError(t, err, "skill file should exist at %s", skillPath)

		// Check file content
		content, err := os.ReadFile(skillPath)
		require.NoError(t, err)

		// Verify content contains key sections
		assert.Contains(t, string(content), "# Crew Manager Skill")
		assert.Contains(t, string(content), "crew --help-manager")
		assert.Contains(t, string(content), "## Constraints")
	}
}

func TestGenSkill_Execute_CreatesDirectories(t *testing.T) {
	// Create temporary directory for testing
	tmpDir := t.TempDir()

	// Create use case
	uc := NewGenSkill(tmpDir)

	// Execute
	_, err := uc.Execute(context.Background(), GenSkillInput{})

	// Assert no error
	require.NoError(t, err)

	// Check that directories were created
	dirs := []string{
		".claude/skills/crew-manager",
		".codex/skills/crew-manager",
		".opencode/skill/crew-manager",
	}

	for _, dir := range dirs {
		dirPath := filepath.Join(tmpDir, dir)
		info, err := os.Stat(dirPath)
		require.NoError(t, err, "directory should exist at %s", dirPath)
		assert.True(t, info.IsDir(), "%s should be a directory", dirPath)
	}
}

func TestGenSkill_Execute_OverwritesExistingFile(t *testing.T) {
	// Create temporary directory for testing
	tmpDir := t.TempDir()

	// Pre-create one of the skill files with different content
	skillDir := filepath.Join(tmpDir, ".claude/skills/crew-manager")
	err := os.MkdirAll(skillDir, 0755)
	require.NoError(t, err)

	oldContent := "OLD CONTENT"
	skillPath := filepath.Join(skillDir, "SKILL.md")
	err = os.WriteFile(skillPath, []byte(oldContent), 0644)
	require.NoError(t, err)

	// Create use case
	uc := NewGenSkill(tmpDir)

	// Execute
	_, err = uc.Execute(context.Background(), GenSkillInput{})
	require.NoError(t, err)

	// Check that file was overwritten
	content, err := os.ReadFile(skillPath)
	require.NoError(t, err)
	assert.NotEqual(t, oldContent, string(content))
	assert.Contains(t, string(content), "# Crew Manager Skill")
}

func TestGenSkill_Execute_ClaudeOnly(t *testing.T) {
	tmpDir := t.TempDir()
	uc := NewGenSkill(tmpDir)

	out, err := uc.Execute(context.Background(), GenSkillInput{
		Claude: true,
	})

	require.NoError(t, err)
	require.Len(t, out.CreatedPaths, 1)

	// Only .claude should be created
	claudePath := filepath.Join(tmpDir, ".claude/skills/crew-manager/SKILL.md")
	assert.Equal(t, claudePath, out.CreatedPaths[0])

	_, err = os.Stat(claudePath)
	require.NoError(t, err)

	// Other directories should not exist
	_, err = os.Stat(filepath.Join(tmpDir, ".codex/skills/crew-manager/SKILL.md"))
	assert.True(t, os.IsNotExist(err))

	_, err = os.Stat(filepath.Join(tmpDir, ".opencode/skill/crew-manager/SKILL.md"))
	assert.True(t, os.IsNotExist(err))
}

func TestGenSkill_Execute_OpenCodeOnly(t *testing.T) {
	tmpDir := t.TempDir()
	uc := NewGenSkill(tmpDir)

	out, err := uc.Execute(context.Background(), GenSkillInput{
		OpenCode: true,
	})

	require.NoError(t, err)
	require.Len(t, out.CreatedPaths, 1)

	// Only .opencode should be created
	opencodePath := filepath.Join(tmpDir, ".opencode/skill/crew-manager/SKILL.md")
	assert.Equal(t, opencodePath, out.CreatedPaths[0])

	_, err = os.Stat(opencodePath)
	require.NoError(t, err)

	// Other directories should not exist
	_, err = os.Stat(filepath.Join(tmpDir, ".claude/skills/crew-manager/SKILL.md"))
	assert.True(t, os.IsNotExist(err))

	_, err = os.Stat(filepath.Join(tmpDir, ".codex/skills/crew-manager/SKILL.md"))
	assert.True(t, os.IsNotExist(err))
}

func TestGenSkill_Execute_CodexOnly(t *testing.T) {
	tmpDir := t.TempDir()
	uc := NewGenSkill(tmpDir)

	out, err := uc.Execute(context.Background(), GenSkillInput{
		Codex: true,
	})

	require.NoError(t, err)
	require.Len(t, out.CreatedPaths, 1)

	// Only .codex should be created
	codexPath := filepath.Join(tmpDir, ".codex/skills/crew-manager/SKILL.md")
	assert.Equal(t, codexPath, out.CreatedPaths[0])

	_, err = os.Stat(codexPath)
	require.NoError(t, err)

	// Other directories should not exist
	_, err = os.Stat(filepath.Join(tmpDir, ".claude/skills/crew-manager/SKILL.md"))
	assert.True(t, os.IsNotExist(err))

	_, err = os.Stat(filepath.Join(tmpDir, ".opencode/skill/crew-manager/SKILL.md"))
	assert.True(t, os.IsNotExist(err))
}

func TestGenSkill_Execute_MultipleFlags(t *testing.T) {
	tmpDir := t.TempDir()
	uc := NewGenSkill(tmpDir)

	out, err := uc.Execute(context.Background(), GenSkillInput{
		Claude: true,
		Codex:  true,
	})

	require.NoError(t, err)
	require.Len(t, out.CreatedPaths, 2)

	// Both .claude and .codex should be created
	claudePath := filepath.Join(tmpDir, ".claude/skills/crew-manager/SKILL.md")
	codexPath := filepath.Join(tmpDir, ".codex/skills/crew-manager/SKILL.md")

	assert.Contains(t, out.CreatedPaths, claudePath)
	assert.Contains(t, out.CreatedPaths, codexPath)

	_, err = os.Stat(claudePath)
	require.NoError(t, err)

	_, err = os.Stat(codexPath)
	require.NoError(t, err)

	// .opencode should not exist
	_, err = os.Stat(filepath.Join(tmpDir, ".opencode/skill/crew-manager/SKILL.md"))
	assert.True(t, os.IsNotExist(err))
}
