#!/bin/bash
# E2E Test Runner

set -e

cd "$(dirname "$0")/.."

claude \
	--allowed-tools="Bash(crew:*),Bash(bash scripts/e2e-setup.sh),Bash(sleep:*),Bash(rm -rf .e2e-test),Bash(echo:*),Bash(which:*),Bash(${HOME}/.claude/skills/terminal/scripts/terminal.sh:*),TodoWrite,Skill(terminal),Read,Grep" \
	"@docs/e2e-test-guide.md Read this document and execute e2e-test. First execute \`bash scripts/e2e-setup.sh\`"
