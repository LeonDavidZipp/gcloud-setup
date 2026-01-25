#!/usr/bin/env bash
# =============================================================================
# Common utilities for all setup scripts
# =============================================================================

# Determine script directory (works even when sourced)
SCRIPTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPTS_DIR")"

# =============================================================================
# Load .env file if it exists
# =============================================================================
load_env() {
    local env_file="${1:-$PROJECT_ROOT/.env}"
    
    if [[ -f "$env_file" ]]; then
        echo "Loading environment from: $env_file"
        set -a  # automatically export all variables
        source "$env_file"
        set +a
    fi
}

# =============================================================================
# Validate required variables
# =============================================================================
require_vars() {
    local missing_vars=()
    
    for var in "$@"; do
        if [[ -z "${!var:-}" ]]; then
            missing_vars+=("$var")
        fi
    done
    
    if [[ ${#missing_vars[@]} -gt 0 ]]; then
        echo "Error: The following required environment variables are not set:" >&2
        for var in "${missing_vars[@]}"; do
            echo "  - $var" >&2
        done
        echo "" >&2
        echo "Set them in .env file or export them before running." >&2
        exit 1
    fi
}

# Auto-load .env when this file is sourced
load_env
