#!/bin/bash

# gitws install script
# This script provides installation guidance for gitws

set -e

echo "🚀 gitws - Git Workspace Manager"
echo "================================="
echo ""
echo "gitws is distributed via Homebrew for easy installation."
echo ""
echo "To install gitws:"
echo ""
echo "  1. Add the tap:"
echo "     brew tap gitworkspaces/homebrew-tap"
echo ""
echo "  2. Install gitws:"
echo "     brew install gitws"
echo ""
echo "  3. Verify installation:"
echo "     gitws --help"
echo ""
echo "For more information, visit: https://gitws.dev"
echo ""
echo "If you prefer to build from source:"
echo "  git clone https://github.com/gitworkspaces/gitws.git"
echo "  cd gitws"
echo "  go build ./cmd/gws"
echo "  ./gws --help"
echo ""
