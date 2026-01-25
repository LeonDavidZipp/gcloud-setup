#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../gcloud/_common.sh"

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
CLOUD_RUN_SERVICE="${CLOUD_RUN_SERVICE:-$GITHUB_REPOSITORY}"
CLOUD_RUN_REGION="${CLOUD_RUN_REGION:-$ARTIFACT_REGISTRY_LOCATION}"

# =============================================================================
# Derived Variables
# =============================================================================
REPO="${GITHUB_ORGANIZATION}/${GITHUB_REPOSITORY}"
SERVICE_ACCOUNT_EMAIL="${SERVICE_ACCOUNT_NAME}@${GCP_PROJECT_ID}.iam.gserviceaccount.com"
WORKLOAD_IDENTITY_PROVIDER="projects/${GCP_PROJECT_NUMBER}/locations/global/workloadIdentityPools/github-pool/providers/github-provider"
ARTIFACT_REGISTRY_URL="${ARTIFACT_REGISTRY_LOCATION}-docker.pkg.dev/${GCP_PROJECT_ID}/${ARTIFACT_REGISTRY_NAME}"

# =============================================================================
# Check gh CLI is installed and authenticated
# =============================================================================
if ! command -v gh &> /dev/null; then
    echo "Error: GitHub CLI (gh) is not installed." >&2
    echo "Install it: https://cli.github.com/" >&2
    exit 1
fi

if ! gh auth status &> /dev/null; then
    echo "Error: GitHub CLI is not authenticated." >&2
    echo "Run: gh auth login" >&2
    exit 1
fi

# =============================================================================
# Verify repository access
# =============================================================================
echo "Verifying access to repository: $REPO"
if ! gh repo view "$REPO" &> /dev/null; then
    echo "Error: Cannot access repository $REPO" >&2
    echo "Check that the repository exists and you have admin access." >&2
    exit 1
fi

# =============================================================================
# Set GitHub Secrets
# =============================================================================
echo ""
echo "Setting GitHub Secrets for $REPO..."

echo "  GCP_SERVICE_ACCOUNT"
gh secret set GCP_SERVICE_ACCOUNT \
    --repo "$REPO" \
    --body "$SERVICE_ACCOUNT_EMAIL"

echo "  GCP_WORKLOAD_IDENTITY_PROVIDER"
gh secret set GCP_WORKLOAD_IDENTITY_PROVIDER \
    --repo "$REPO" \
    --body "$WORKLOAD_IDENTITY_PROVIDER"

# =============================================================================
# Set GitHub Variables
# =============================================================================
echo ""
echo "Setting GitHub Variables for $REPO..."

echo "  CLOUD_RUN_SERVICE"
gh variable set CLOUD_RUN_SERVICE \
    --repo "$REPO" \
    --body "$CLOUD_RUN_SERVICE"

echo "  CLOUD_RUN_REGION"
gh variable set CLOUD_RUN_REGION \
    --repo "$REPO" \
    --body "$CLOUD_RUN_REGION"

echo "  ARTIFACT_REGISTRY_URL"
gh variable set ARTIFACT_REGISTRY_URL \
    --repo "$REPO" \
    --body "$ARTIFACT_REGISTRY_URL"

# =============================================================================
# Summary
# =============================================================================
echo ""
echo "=============================================="
echo "GitHub Configuration Complete!"
echo "=============================================="
echo ""
echo "Repository: $REPO"
echo ""
echo "Secrets:"
echo "  GCP_SERVICE_ACCOUNT:            $SERVICE_ACCOUNT_EMAIL"
echo "  GCP_WORKLOAD_IDENTITY_PROVIDER: $WORKLOAD_IDENTITY_PROVIDER"
echo ""
echo "Variables:"
echo "  CLOUD_RUN_SERVICE:              $CLOUD_RUN_SERVICE"
echo "  CLOUD_RUN_REGION:               $CLOUD_RUN_REGION"
echo "  ARTIFACT_REGISTRY_URL:          $ARTIFACT_REGISTRY_URL"
echo ""
echo "View at: https://github.com/$REPO/settings/secrets/actions"
