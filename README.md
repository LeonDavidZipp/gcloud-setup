# gcsetup

A CLI tool to set up GCloud projects with GitHub Actions CI/CD in a single command.

## What it does

`gcsetup` automates the entire setup process for deploying applications to Google Cloud Run via GitHub Actions:

**GCP Setup:**
- Enables required APIs (Cloud Run, Artifact Registry, IAM, etc.)
- Creates a service account with minimal required permissions
- Configures Workload Identity Federation (no service account keys!)
- Creates an Artifact Registry repository for Docker images

**GitHub Setup:**
- Configures repository secrets for GCP authentication
- Sets repository variables for deployment configuration
- Creates deployment environments (development, staging, production, preview)

**Project Setup:**
- Adds a production-ready GitHub Actions workflow
- Creates a `.env.gcloud` configuration template

## Prerequisites

- [Google Cloud SDK](https://cloud.google.com/sdk/docs/install) (`gcloud`)
- [GitHub CLI](https://cli.github.com/) (`gh`)
- [Go 1.21+](https://golang.org/dl/) (for building from source)

```bash
# Authenticate both CLIs
gcloud auth login
gh auth login
```

## Installation

### From source

```bash
git clone https://github.com/LeonDavidZipp/gcloud-setup.git
cd gcloud-setup
go build -o gcsetup .
sudo mv gcsetup /usr/local/bin/
```

### Verify installation

```bash
gcsetup --help
```

## Usage

### 1. Initialize a project

Navigate to your project repository and run:

```bash
gcsetup init
```

This creates:
- `.github/workflows/gcloud-deploy.yml` — CI/CD workflow
- `.env.gcloud` — Configuration template (added to `.gitignore`)

### 2. Configure variables

Edit `.env.gcloud` with your values:

```bash
# GCP Project
GCP_PROJECT_ID=my-project-id
GCP_PROJECT_NUMBER=123456789012

# GitHub Repository
GITHUB_ORGANIZATION=my-org
GITHUB_REPOSITORY=my-repo

# Service Account
SERVICE_ACCOUNT_NAME=github-actions

# Artifact Registry
ARTIFACT_REGISTRY_NAME=docker-registry
ARTIFACT_REGISTRY_LOCATION=europe-west1

# Cloud Run
CLOUD_RUN_SERVICE=my-api
CLOUD_RUN_REGION=europe-west1
```

> **Tip:** Find your project number with:
> ```bash
> gcloud projects describe YOUR_PROJECT_ID --format="value(projectNumber)"
> ```

### 3. Run setup

```bash
gcsetup setup
```

This executes all setup steps:
1. Enables required GCP APIs
2. Creates service account with CI/CD roles
3. Configures Workload Identity Federation
4. Creates Artifact Registry repository
5. Sets GitHub secrets, variables, and environments

### Alternative: Use flags

You can also pass configuration via flags:

```bash
gcsetup setup \
  --gcp-project-id=my-project \
  --gcp-project-number=123456789012 \
  --github-org=my-org \
  --github-repo=my-repo \
  --service-account-name=github-actions \
  --artifact-registry-name=docker-registry \
  --artifact-registry-location=europe-west1 \
  --cloud-run-service=my-api \
  --cloud-run-region=europe-west1
```

### Dry run

Preview commands without executing:

```bash
gcsetup setup --dry-run
```

## Configuration Reference

| Variable | Description | Example |
|----------|-------------|---------|
| `GCP_PROJECT_ID` | GCP project ID | `my-project` |
| `GCP_PROJECT_NUMBER` | GCP project number (numeric) | `123456789012` |
| `GITHUB_ORGANIZATION` | GitHub org or username | `my-org` |
| `GITHUB_REPOSITORY` | GitHub repository name | `my-repo` |
| `SERVICE_ACCOUNT_NAME` | Name for the service account | `github-actions` |
| `ARTIFACT_REGISTRY_NAME` | Docker registry name | `docker-registry` |
| `ARTIFACT_REGISTRY_LOCATION` | GCP region for registry | `europe-west1` |
| `CLOUD_RUN_SERVICE` | Cloud Run service name | `my-api` |
| `CLOUD_RUN_REGION` | GCP region for Cloud Run | `europe-west1` |

### Configuration priority

1. CLI flags (highest priority)
2. Environment variables
3. `.env.gcloud` file (lowest priority)

## What gets created

### In GCP

| Resource | Description |
|----------|-------------|
| **Service Account** | `{name}@{project}.iam.gserviceaccount.com` |
| **Workload Identity Pool** | `github-pool` |
| **OIDC Provider** | `github-provider` |
| **Artifact Registry** | Docker repository for container images |

### Service Account Roles

| Role | Purpose |
|------|---------|
| `roles/run.developer` | Deploy to Cloud Run |
| `roles/artifactregistry.writer` | Push container images |
| `roles/secretmanager.secretAccessor` | Access secrets (optional) |
| `roles/iam.serviceAccountUser` | Act as service account |
| `roles/cloudbuild.builds.builder` | Build with Cloud Build |
| `roles/logging.logWriter` | Write build logs |

### In GitHub

| Type | Name | Value |
|------|------|-------|
| Secret | `GCP_SERVICE_ACCOUNT` | Service account email |
| Secret | `GCP_WORKLOAD_IDENTITY_PROVIDER` | Workload Identity provider path |
| Variable | `CLOUD_RUN_SERVICE` | Cloud Run service name |
| Variable | `CLOUD_RUN_REGION` | Deployment region |
| Variable | `ARTIFACT_REGISTRY_URL` | Full registry URL |

### Environments

The following deployment environments are created:

| Environment | Purpose |
|-------------|----------|
| `development` | Development deployments |
| `staging` | Pre-production testing |
| `production` | Production deployments |
| `preview` | Pull request preview deployments |

> **💡 Tip:** Add protection rules in GitHub → Settings → Environments. For `production`, consider requiring reviewers before deployment.

## The Workflow

The generated `.github/workflows/gcloud-deploy.yml` handles:

| Trigger | Action |
|---------|--------|
| Push to `main` | Build → Test → Deploy to production |
| Tag `v*.*.*` | Build → Test → Deploy to production (tagged) |
| PR opened/updated | Build → Test → Deploy preview environment |
| PR closed | Cleanup preview environment |

### Features

- **Workload Identity Federation** — No service account keys
- **Cloud Build** — Builds run in GCP, not GitHub runners
- **Preview environments** — Each PR gets its own Cloud Run service
- **Auto-cleanup** — Preview services deleted when PR closes
- **Concurrency control** — Cancels outdated deployments

## Project structure

```
your-repo/
├── .github/
│   └── workflows/
│       └── gcloud-deploy.yml      # Created by `gcsetup init`
├── .env.gcloud             # Created by `gcsetup init` (gitignored)
├── Dockerfile              # You provide this
└── ...
```

## Requirements for your project

Your project needs a `Dockerfile` in the root directory. The workflow will:

1. Build the image using Cloud Build
2. Push to Artifact Registry
3. Deploy to Cloud Run

### Example Dockerfile

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o main .

FROM alpine:latest
COPY --from=builder /app/main /main
EXPOSE 8080
CMD ["/main"]
```

## Troubleshooting

### "Permission denied" errors

Ensure your `gcloud` account has Owner or Editor role on the project:

```bash
gcloud projects get-iam-policy YOUR_PROJECT_ID \
  --flatten="bindings[].members" \
  --filter="bindings.members:YOUR_EMAIL"
```

### "Repository not found" in GitHub

Ensure you have admin access to the repository:

```bash
gh repo view OWNER/REPO
```

### Workload Identity not working

Verify the pool and provider exist:

```bash
gcloud iam workload-identity-pools list --location=global
gcloud iam workload-identity-pools providers list \
  --workload-identity-pool=github-pool \
  --location=global
```

## License

MIT
