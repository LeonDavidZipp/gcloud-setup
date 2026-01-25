#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/gcloud/_common.sh"

# =============================================================================
# Required Environment Variables
# =============================================================================
require_vars GCP_PROJECT_ID GCP_PROJECT_NUMBER GITHUB_ORGANIZATION GITHUB_REPOSITORY

# =============================================================================
# Optional Environment Variables (with defaults)
# =============================================================================
SERVICE_ACCOUNT_NAME="${SERVICE_ACCOUNT_NAME:-github-actions}"
ARTIFACT_REGISTRY_NAME="${ARTIFACT_REGISTRY_NAME:-docker-registry}"
ARTIFACT_REGISTRY_LOCATION="${ARTIFACT_REGISTRY_LOCATION:-europe-west1}"

# =============================================================================
# Derived Variables
# =============================================================================
SERVICE_ACCOUNT_EMAIL="${SERVICE_ACCOUNT_NAME}@${GCP_PROJECT_ID}.iam.gserviceaccount.com"

# Export for child scripts
export GCP_PROJECT_ID
export GCP_PROJECT_NUMBER
export GITHUB_ORGANIZATION
export GITHUB_REPOSITORY
export SERVICE_ACCOUNT_NAME
export SERVICE_ACCOUNT_EMAIL
export ARTIFACT_REGISTRY_NAME
export ARTIFACT_REGISTRY_LOCATION

# =============================================================================
# Run Setup Scripts
# =============================================================================
echo "=============================================="
echo "GCloud Project Setup"
echo "=============================================="
echo "Project ID:     $GCP_PROJECT_ID"
echo "Project Number: $GCP_PROJECT_NUMBER"
echo "GitHub:         $GITHUB_ORGANIZATION/$GITHUB_REPOSITORY"
echo "=============================================="
echo ""

echo "Step 1/5: Enabling APIs..."
echo "----------------------------------------------"
"$SCRIPT_DIR/gcloud/01-enable-apis.sh"
echo ""

echo "Step 2/5: Creating Service Account..."
echo "----------------------------------------------"
"$SCRIPT_DIR/gcloud/02-create-service-account.sh"
echo ""

echo "Step 3/5: Setting up Workload Identity Federation..."
echo "----------------------------------------------"
"$SCRIPT_DIR/gcloud/03-iam-setup.sh"
echo ""

echo "Step 4/5: Creating Artifact Registry..."
echo "----------------------------------------------"
"$SCRIPT_DIR/gcloud/04-artifact-registry-setup.sh"
echo ""

echo "Step 5/5: Configuring GitHub Repository..."
echo "----------------------------------------------"
"$SCRIPT_DIR/github/06-github-setup.sh"
echo ""

# =============================================================================
# Summary
# =============================================================================
REGISTRY_URL="${ARTIFACT_REGISTRY_LOCATION}-docker.pkg.dev/${GCP_PROJECT_ID}/${ARTIFACT_REGISTRY_NAME}"

echo "=============================================="
echo "Setup Complete!"
echo "=============================================="
echo ""
echo "Your repository is fully configured."
echo "Push to main or create a PR to trigger a deployment."
echo ""
echo "Optional: Run 05-secret-manager-setup.sh to create secrets"
echo "  export SECRETS_TO_CREATE=\"DATABASE_URL API_KEY\""
echo "  ./05-secret-manager-setup.sh"
