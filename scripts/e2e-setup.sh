#!/bin/bash
# E2E Test Environment Setup Script
#
# Usage:
#   ./scripts/e2e-setup.sh [test-dir]
#
# Creates a temporary git repository for E2E testing.
# Default location is .e2e-test/ in the repository root (gitignored).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TEST_DIR="${1:-$REPO_ROOT/.e2e-test}"
CREW_BIN="${CREW_BIN:-$(which crew)}"

echo "=== E2E Test Environment Setup ==="
echo "Test directory: $TEST_DIR"
echo "Crew binary: $CREW_BIN"
echo ""

# Cleanup if exists
if [ -d "$TEST_DIR" ]; then
    echo "Removing existing test directory..."
    rm -rf "$TEST_DIR"
fi

# Create test repository
echo "Creating test repository..."
mkdir -p "$TEST_DIR"
cd "$TEST_DIR"

git init
git config user.email "test@example.com"
git config user.name "E2E Test"

# Create initial files
cat > README.md << 'EOF'
# E2E Test Repository

This is a temporary repository for crew E2E testing.
EOF

cat > main.go << 'EOF'
package main

import "fmt"

func main() {
    fmt.Println("Hello, E2E Test!")
}
EOF

git add .
git commit -m "Initial commit"

# Initialize crew
echo "Initializing crew..."
"$CREW_BIN" init

# Create .claude/settings.json for permission pre-approval
echo "Creating .claude/settings.json..."
mkdir -p .claude
cat > .claude/settings.json << 'EOF'
{
  "permissions": {
    "allow": [
      "Bash(mise run:*)",
      "Bash(crew diff:*)",
      "Bash(crew comment:*)",
      "Bash(cd:*)"
    ]
  }
}
EOF

echo ""
echo "=== Setup Complete ==="
echo ""
echo "Test directory: $TEST_DIR"
echo ""
echo "Next steps:"
echo "  cd $TEST_DIR"
echo "  crew new --title 'Test task'"
echo "  crew start <id> cc-small"
echo ""
echo "Cleanup:"
echo "  rm -rf $TEST_DIR"
echo ""
echo "Note: .e2e-test/ is gitignored in the main repository."
