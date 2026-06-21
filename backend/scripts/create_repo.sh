#!/bin/bash
# 仓库创建辅助脚本 - 可在此扩展更复杂的逻辑
set -e

REPO_NAME="$1"
DESCRIPTION="${2:-}"
PRIVATE="${3:-false}"
TEMPLATE="${4:-}"

echo "Creating repo: $REPO_NAME"

if [ -z "$REPO_NAME" ]; then
    echo "ERROR: Repo name is required"
    exit 1
fi

if ! command -v gh &> /dev/null; then
    echo "ERROR: GitHub CLI (gh) is not installed"
    exit 1
fi

if ! gh auth status &> /dev/null; then
    echo "ERROR: GitHub CLI is not authenticated. Run 'gh auth login' first."
    exit 1
fi

echo "GitHub CLI ready"
