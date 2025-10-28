#!/bin/bash

# Setup script to configure git hooks

echo "🔧 Setting up git hooks..."

# Configure git to use .githooks directory
git config core.hooksPath .githooks

echo "✅ Git hooks configured!"
echo ""
echo "Pre-commit hook will now run automatically before each commit."
echo "It will check:"
echo "  - Code formatting (gofmt)"
echo "  - Templ formatting"
echo "  - Tests pass"
echo "  - Linters pass (golangci-lint)"
echo "  - go.mod is tidy"
echo ""
