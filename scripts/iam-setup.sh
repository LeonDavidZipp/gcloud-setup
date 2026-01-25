#!/usr/bin/env bash
set -euo pipefail

# =============================================================================
# Required Environment Variables
# =============================================================================
GCLOUD_PROJECT_ID="${GCLOUD_PROJECT_ID:-}"
GCLOUD_PROJECT_NUMBER="${GCLOUD_PROJECT_NUMBER:-}"
GITHUB_ORGANIZATION="${GITHUB_ORGANIZATION:-}"
GITHUB_REPOSITORY="${GITHUB_REPOSITORY:-}"
SERVICE_ACCOUNT_EMAIL="${SERVICE_ACCOUNT_EMAIL:-}"

# =============================================================================
# Validate Required Variables
# =============================================================================
missing_vars=()

[[ -z "$GCLOUD_PROJECT_ID" ]] && missing_vars+=("GCLOUD_PROJECT_ID")
[[ -z "$GCLOUD_PROJECT_NUMBER" ]] && missing_vars+=("GCLOUD_PROJECT_NUMBER")
[[ -z "$GITHUB_ORGANIZATION" ]] && missing_vars+=("GITHUB_ORGANIZATION")
[[ -z "$GITHUB_REPOSITORY" ]] && missing_vars+=("GITHUB_REPOSITORY")
[[ -z "$SERVICE_ACCOUNT_EMAIL" ]] && missing_vars+=("SERVICE_ACCOUNT_EMAIL")

if [[ ${#missing_vars[@]} -gt 0 ]]; then
    echo "Error: The following required environment variables are not set:" >&2
    for var in "${missing_vars[@]}"; do
        echo "  - $var" >&2
    done
    exit 1
fi

# =============================================================================
# 1. Create a Workload Identity Pool
# =============================================================================
gcloud iam workload-identity-pools create "github-pool" \
    --project="$GCLOUD_PROJECT_ID" \
    --location="global" \
    --display-name="GitHub Actions Pool"

# =============================================================================
# 2. Create a Provider for GitHub
# =============================================================================
gcloud iam workload-identity-pools providers create-oidc "github-provider" \
    --project="$GCLOUD_PROJECT_ID" \
    --location="global" \
    --workload-identity-pool="github-pool" \
    --display-name="GitHub Provider" \
    --attribute-mapping="google.subject=assertion.sub,attribute.actor=assertion.actor,attribute.repository=assertion.repository" \
    --issuer-uri="https://token.actions.githubusercontent.com"

# =============================================================================
# 3. Allow your repo to impersonate the service account
# =============================================================================
gcloud iam service-accounts add-iam-policy-binding "$SERVICE_ACCOUNT_EMAIL" \
    --project="$GCLOUD_PROJECT_ID" \
    --role="roles/iam.workloadIdentityUser" \
    --member="principalSet://iam.googleapis.com/projects/$GCLOUD_PROJECT_NUMBER/locations/global/workloadIdentityPools/github-pool/attribute.repository/$GITHUB_ORGANIZATION/$GITHUB_REPOSITORY"