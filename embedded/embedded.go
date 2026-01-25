// Package embedded provides embedded template files for gcsetup initialization.
package embedded

import (
	_ "embed"
)

// DeployWorkflow contains the GitHub Actions CI/CD workflow template.
//
//go:embed gcloud-deploy.yml
var DeployWorkflow []byte

// EnvTemplate contains the .env.gcloud configuration template.
//
//go:embed env.gcloud.template
var EnvTemplate []byte
