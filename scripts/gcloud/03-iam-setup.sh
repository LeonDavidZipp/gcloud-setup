#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/_common.sh"

# =============================================================================
# Required Environment Variables
# =============================================================================
require_vars GCP_PROJECT_ID GCP_PROJECT_NUMBER GITHUB_ORGANIZATION GITHUB_REPOSITORY SERVICE_ACCOUNT_EMAIL

# =============================================================================
# 1. Create a Workload Identity Pool
# =============================================================================
gcloud iam workload-identity-pools create "github-pool" \
    --project="$GCP_PROJECT_ID" \
    --location="global" \
    --display-name="GitHub Actions Pool"

# =============================================================================
# 2. Create a Provider for GitHub
# =============================================================================
gcloud iam workload-identity-pools providers create-oidc "github-provider" \
    --project="$GCP_PROJECT_ID" \
    --location="global" \
    --workload-identity-pool="github-pool" \
    --display-name="GitHub Provider" \
    --attribute-mapping="google.subject=assertion.sub,attribute.actor=assertion.actor,attribute.repository=assertion.repository" \
    --issuer-uri="https://token.actions.githubusercontent.com"

# =============================================================================
# 3. Allow your repo to impersonate the service account
# =============================================================================
gcloud iam service-accounts add-iam-policy-binding "$SERVICE_ACCOUNT_EMAIL" \
    --project="$GCP_PROJECT_ID" \
    --role="roles/iam.workloadIdentityUser" \
    --member="principalSet://iam.googleapis.com/projects/$GCP_PROJECT_NUMBER/locations/global/workloadIdentityPools/github-pool/attribute.repository/$GITHUB_ORGANIZATION/$GITHUB_REPOSITORY"