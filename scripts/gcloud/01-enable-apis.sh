#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/_common.sh"

# =============================================================================
# Required Environment Variables
# =============================================================================
require_vars GCP_PROJECT_ID

# =============================================================================
# APIs to Enable
# =============================================================================
APIS=(
    "cloudresourcemanager.googleapis.com"
    "iam.googleapis.com"
    "iamcredentials.googleapis.com"
    "artifactregistry.googleapis.com"
    "run.googleapis.com"
    "secretmanager.googleapis.com"
    "cloudbuild.googleapis.com"
)

# =============================================================================
# Enable APIs
# =============================================================================
echo "Enabling APIs for project: $GCP_PROJECT_ID"

for api in "${APIS[@]}"; do
    echo "  Enabling $api..."
    gcloud services enable "$api" --project="$GCP_PROJECT_ID"
done

echo "All APIs enabled successfully."
