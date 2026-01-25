#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/_common.sh"

# =============================================================================
# Required Environment Variables
# =============================================================================
require_vars GCP_PROJECT_ID ARTIFACT_REGISTRY_NAME ARTIFACT_REGISTRY_LOCATION

# =============================================================================
# Create Artifact Registry Repository
# =============================================================================
echo "Creating Artifact Registry repository: $ARTIFACT_REGISTRY_NAME"

gcloud artifacts repositories create "$ARTIFACT_REGISTRY_NAME" \
    --project="$GCP_PROJECT_ID" \
    --location="$ARTIFACT_REGISTRY_LOCATION" \
    --repository-format="docker" \
    --description="Container registry for CI/CD"

# =============================================================================
# Output
# =============================================================================
REGISTRY_URL="${ARTIFACT_REGISTRY_LOCATION}-docker.pkg.dev/${GCP_PROJECT_ID}/${ARTIFACT_REGISTRY_NAME}"

echo ""
echo "Artifact Registry created successfully."
echo "Registry URL: $REGISTRY_URL"
echo ""
echo "To push images:"
echo "  docker tag <image> ${REGISTRY_URL}/<image-name>:<tag>"
echo "  docker push ${REGISTRY_URL}/<image-name>:<tag>"
