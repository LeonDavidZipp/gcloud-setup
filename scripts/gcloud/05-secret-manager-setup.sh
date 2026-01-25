#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/_common.sh"

# =============================================================================
# Required Environment Variables
# =============================================================================
require_vars GCP_PROJECT_ID SERVICE_ACCOUNT_EMAIL

# =============================================================================
# Optional: Secrets to Create (space-separated list)
# Example: SECRETS_TO_CREATE="DATABASE_URL API_KEY JWT_SECRET"
# =============================================================================
SECRETS_TO_CREATE="${SECRETS_TO_CREATE:-}"

# =============================================================================
# Create Secrets (if specified)
# =============================================================================
if [[ -n "$SECRETS_TO_CREATE" ]]; then
    echo "Creating secrets..."
    
    for secret_name in $SECRETS_TO_CREATE; do
        echo "  Creating secret: $secret_name"
        
        # Create the secret
        gcloud secrets create "$secret_name" \
            --project="$GCP_PROJECT_ID" \
            --replication-policy="automatic"
        
        # Grant access to service account
        gcloud secrets add-iam-policy-binding "$secret_name" \
            --project="$GCP_PROJECT_ID" \
            --member="serviceAccount:$SERVICE_ACCOUNT_EMAIL" \
            --role="roles/secretmanager.secretAccessor"
        
        echo "    Created and granted access to $SERVICE_ACCOUNT_EMAIL"
    done
    
    echo ""
    echo "Secrets created. Add values with:"
    echo "  echo -n 'secret-value' | gcloud secrets versions add SECRET_NAME --data-file=-"
else
    echo "No secrets specified in SECRETS_TO_CREATE."
    echo ""
    echo "To create secrets, set SECRETS_TO_CREATE:"
    echo "  export SECRETS_TO_CREATE=\"DATABASE_URL API_KEY JWT_SECRET\""
    echo ""
    echo "Or create manually:"
    echo "  gcloud secrets create SECRET_NAME --project=\"$GCP_PROJECT_ID\" --replication-policy=\"automatic\""
fi
