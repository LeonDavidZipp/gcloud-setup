#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/_common.sh"

# =============================================================================
# Required Environment Variables
# =============================================================================
require_vars GCP_PROJECT_ID SERVICE_ACCOUNT_NAME

# =============================================================================
# Derived Variables
# =============================================================================
SERVICE_ACCOUNT_EMAIL="${SERVICE_ACCOUNT_NAME}@${GCP_PROJECT_ID}.iam.gserviceaccount.com"
SERVICE_ACCOUNT_DISPLAY_NAME="${SERVICE_ACCOUNT_NAME} Service Account"

# =============================================================================
# Roles to Grant
# =============================================================================
ROLES=(
    "roles/run.developer"
    "roles/artifactregistry.writer"
    "roles/secretmanager.secretAccessor"
    "roles/iam.serviceAccountUser"
    "roles/cloudbuild.builds.builder"
    "roles/logging.logWriter"
)

# =============================================================================
# 1. Create Service Account
# =============================================================================
echo "Creating service account: $SERVICE_ACCOUNT_NAME"

gcloud iam service-accounts create "$SERVICE_ACCOUNT_NAME" \
    --project="$GCP_PROJECT_ID" \
    --display-name="$SERVICE_ACCOUNT_DISPLAY_NAME" \
    --description="Service account for GitHub Actions CI/CD"

# =============================================================================
# 2. Grant Roles to Service Account
# =============================================================================
echo "Granting roles to service account..."

for role in "${ROLES[@]}"; do
    echo "  Granting $role..."
    gcloud projects add-iam-policy-binding "$GCP_PROJECT_ID" \
        --member="serviceAccount:$SERVICE_ACCOUNT_EMAIL" \
        --role="$role" \
        --condition=None
done

echo ""
echo "Service account created successfully."
echo "Email: $SERVICE_ACCOUNT_EMAIL"
echo ""
echo "Export this for subsequent scripts:"
echo "  export SERVICE_ACCOUNT_EMAIL=\"$SERVICE_ACCOUNT_EMAIL\""
