package embedded

import (
	_ "embed"
)

//go:embed gcloud-deploy.yml
var DeployWorkflow []byte

//go:embed env.gcloud.template
var EnvTemplate []byte
