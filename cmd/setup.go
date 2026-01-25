package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Set up GCloud project and GitHub repository",
	Long: `Runs the complete setup process:
  1. Enable required GCP APIs
  2. Create service account with necessary roles
  3. Set up Workload Identity Federation for GitHub
  4. Create Artifact Registry repository
  5. Configure GitHub repository secrets and variables`,
	RunE: runSetup,
}

var dryRun bool

func init() {
	rootCmd.AddCommand(setupCmd)
	setupCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print commands without executing")
}

func runSetup(cmd *cobra.Command, args []string) error {
	// Validate all required variables are set
	if err := ValidateConfig(); err != nil {
		return err
	}

	// Check gcloud is installed and authenticated
	if err := checkGcloud(); err != nil {
		return err
	}

	// Check gh is installed and authenticated
	if err := checkGH(); err != nil {
		return err
	}

	cfg := loadConfig()

	fmt.Println("==============================================")
	fmt.Println("GCloud Project Setup")
	fmt.Println("==============================================")
	fmt.Printf("Project ID:     %s\n", cfg.ProjectID)
	fmt.Printf("Project Number: %s\n", cfg.ProjectNumber)
	fmt.Printf("GitHub:         %s/%s\n", cfg.GitHubOrg, cfg.GitHubRepo)
	fmt.Println("==============================================")
	fmt.Println()

	steps := []struct {
		name string
		fn   func(Config) error
	}{
		{"Enabling APIs", enableAPIs},
		{"Creating Service Account", createServiceAccount},
		{"Setting up Workload Identity Federation", setupWorkloadIdentity},
		{"Creating Artifact Registry", createArtifactRegistry},
		{"Configuring GitHub Repository", configureGitHub},
	}

	for i, step := range steps {
		fmt.Printf("Step %d/%d: %s...\n", i+1, len(steps), step.name)
		fmt.Println("----------------------------------------------")
		if err := step.fn(cfg); err != nil {
			return fmt.Errorf("%s failed: %w", step.name, err)
		}
		fmt.Println()
	}

	fmt.Println("==============================================")
	fmt.Println("Setup Complete!")
	fmt.Println("==============================================")
	fmt.Println()
	fmt.Println("Your repository is fully configured.")
	fmt.Println("Push to main or create a PR to trigger a deployment.")

	return nil
}

type Config struct {
	ProjectID                string
	ProjectNumber            string
	GitHubOrg                string
	GitHubRepo               string
	ServiceAccountName       string
	ServiceAccountEmail      string
	ArtifactRegistryName     string
	ArtifactRegistryLocation string
	CloudRunService          string
	CloudRunRegion           string
	WorkloadIdentityProvider string
	ArtifactRegistryURL      string
}

func loadConfig() Config {
	projectID := viper.GetString("GCP_PROJECT_ID")
	projectNumber := viper.GetString("GCP_PROJECT_NUMBER")
	saName := viper.GetString("SERVICE_ACCOUNT_NAME")
	arLocation := viper.GetString("ARTIFACT_REGISTRY_LOCATION")
	arName := viper.GetString("ARTIFACT_REGISTRY_NAME")

	return Config{
		ProjectID:                projectID,
		ProjectNumber:            projectNumber,
		GitHubOrg:                viper.GetString("GITHUB_ORGANIZATION"),
		GitHubRepo:               viper.GetString("GITHUB_REPOSITORY"),
		ServiceAccountName:       saName,
		ServiceAccountEmail:      fmt.Sprintf("%s@%s.iam.gserviceaccount.com", saName, projectID),
		ArtifactRegistryName:     arName,
		ArtifactRegistryLocation: arLocation,
		CloudRunService:          viper.GetString("CLOUD_RUN_SERVICE"),
		CloudRunRegion:           viper.GetString("CLOUD_RUN_REGION"),
		WorkloadIdentityProvider: fmt.Sprintf("projects/%s/locations/global/workloadIdentityPools/github-pool/providers/github-provider", projectNumber),
		ArtifactRegistryURL:      fmt.Sprintf("%s-docker.pkg.dev/%s/%s", arLocation, projectID, arName),
	}
}

func checkGcloud() error {
	if _, err := exec.LookPath("gcloud"); err != nil {
		return fmt.Errorf("gcloud CLI not found. Install it: https://cloud.google.com/sdk/docs/install")
	}
	return nil
}

func checkGH() error {
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("GitHub CLI (gh) not found. Install it: https://cli.github.com/")
	}
	// Check authentication
	cmd := exec.Command("gh", "auth", "status")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("GitHub CLI not authenticated. Run: gh auth login")
	}
	return nil
}

func runCommand(name string, args ...string) error {
	if dryRun {
		fmt.Printf("  [dry-run] %s %s\n", name, strings.Join(args, " "))
		return nil
	}
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runCommandSilent(name string, args ...string) error {
	if dryRun {
		fmt.Printf("  [dry-run] %s %s\n", name, strings.Join(args, " "))
		return nil
	}
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

func enableAPIs(cfg Config) error {
	apis := []string{
		"cloudresourcemanager.googleapis.com",
		"iam.googleapis.com",
		"iamcredentials.googleapis.com",
		"artifactregistry.googleapis.com",
		"run.googleapis.com",
		"secretmanager.googleapis.com",
		"cloudbuild.googleapis.com",
	}

	for _, api := range apis {
		fmt.Printf("  Enabling %s\n", api)
		if err := runCommandSilent("gcloud", "services", "enable", api, "--project="+cfg.ProjectID); err != nil {
			return err
		}
	}
	return nil
}

func createServiceAccount(cfg Config) error {
	// Create service account
	fmt.Printf("  Creating service account: %s\n", cfg.ServiceAccountName)
	err := runCommandSilent("gcloud", "iam", "service-accounts", "create", cfg.ServiceAccountName,
		"--project="+cfg.ProjectID,
		"--display-name="+cfg.ServiceAccountName+" Service Account",
		"--description=Service account for GitHub Actions CI/CD",
	)
	if err != nil {
		// Might already exist, continue
		fmt.Println("  (service account may already exist, continuing...)")
	}

	// Grant roles
	roles := []string{
		"roles/run.developer",
		"roles/artifactregistry.writer",
		"roles/secretmanager.secretAccessor",
		"roles/iam.serviceAccountUser",
		"roles/cloudbuild.builds.builder",
		"roles/logging.logWriter",
	}

	for _, role := range roles {
		fmt.Printf("  Granting %s\n", role)
		if err := runCommandSilent("gcloud", "projects", "add-iam-policy-binding", cfg.ProjectID,
			"--member=serviceAccount:"+cfg.ServiceAccountEmail,
			"--role="+role,
			"--condition=None",
		); err != nil {
			return err
		}
	}

	return nil
}

func setupWorkloadIdentity(cfg Config) error {
	// Create Workload Identity Pool
	fmt.Println("  Creating Workload Identity Pool...")
	err := runCommandSilent("gcloud", "iam", "workload-identity-pools", "create", "github-pool",
		"--project="+cfg.ProjectID,
		"--location=global",
		"--display-name=GitHub Actions Pool",
	)
	if err != nil {
		fmt.Println("  (pool may already exist, continuing...)")
	}

	// Create Provider
	fmt.Println("  Creating OIDC Provider...")
	err = runCommandSilent("gcloud", "iam", "workload-identity-pools", "providers", "create-oidc", "github-provider",
		"--project="+cfg.ProjectID,
		"--location=global",
		"--workload-identity-pool=github-pool",
		"--display-name=GitHub Provider",
		"--attribute-mapping=google.subject=assertion.sub,attribute.actor=assertion.actor,attribute.repository=assertion.repository",
		"--issuer-uri=https://token.actions.githubusercontent.com",
	)
	if err != nil {
		fmt.Println("  (provider may already exist, continuing...)")
	}

	// Allow repo to impersonate service account
	fmt.Println("  Configuring repository access...")
	member := fmt.Sprintf("principalSet://iam.googleapis.com/projects/%s/locations/global/workloadIdentityPools/github-pool/attribute.repository/%s/%s",
		cfg.ProjectNumber, cfg.GitHubOrg, cfg.GitHubRepo)

	return runCommandSilent("gcloud", "iam", "service-accounts", "add-iam-policy-binding", cfg.ServiceAccountEmail,
		"--project="+cfg.ProjectID,
		"--role=roles/iam.workloadIdentityUser",
		"--member="+member,
	)
}

func createArtifactRegistry(cfg Config) error {
	fmt.Printf("  Creating repository: %s\n", cfg.ArtifactRegistryName)
	err := runCommandSilent("gcloud", "artifacts", "repositories", "create", cfg.ArtifactRegistryName,
		"--project="+cfg.ProjectID,
		"--location="+cfg.ArtifactRegistryLocation,
		"--repository-format=docker",
		"--description=Container registry for CI/CD",
	)
	if err != nil {
		fmt.Println("  (repository may already exist, continuing...)")
	}
	return nil
}

func configureGitHub(cfg Config) error {
	repo := fmt.Sprintf("%s/%s", cfg.GitHubOrg, cfg.GitHubRepo)

	// Set secrets
	fmt.Println("  Setting secrets...")
	secrets := map[string]string{
		"GCP_SERVICE_ACCOUNT":            cfg.ServiceAccountEmail,
		"GCP_WORKLOAD_IDENTITY_PROVIDER": cfg.WorkloadIdentityProvider,
	}

	for name, value := range secrets {
		fmt.Printf("    %s\n", name)
		if dryRun {
			fmt.Printf("    [dry-run] gh secret set %s --repo %s\n", name, repo)
			continue
		}
		cmd := exec.Command("gh", "secret", "set", name, "--repo", repo, "--body", value)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set secret %s: %w", name, err)
		}
	}

	// Set variables
	fmt.Println("  Setting variables...")
	variables := map[string]string{
		"CLOUD_RUN_SERVICE":     cfg.CloudRunService,
		"CLOUD_RUN_REGION":      cfg.CloudRunRegion,
		"ARTIFACT_REGISTRY_URL": cfg.ArtifactRegistryURL,
	}

	for name, value := range variables {
		fmt.Printf("    %s\n", name)
		if dryRun {
			fmt.Printf("    [dry-run] gh variable set %s --repo %s\n", name, repo)
			continue
		}
		cmd := exec.Command("gh", "variable", "set", name, "--repo", repo, "--body", value)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set variable %s: %w", name, err)
		}
	}

	return nil
}
