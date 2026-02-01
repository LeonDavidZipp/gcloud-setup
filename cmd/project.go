package cmd

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage GCP projects",
	Long:  `Commands for creating and managing GCP projects.`,
}

var projectCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new GCP project",
	Long: `Create a new GCP project with:
  1. Project creation in GCP
  2. Enabling required APIs
  3. Creating service account with necessary roles
  4. Setting up Workload Identity Federation for GitHub
  5. Creating Artifact Registry repository`,
	RunE: runProjectCreate,
}

var projectDryRun bool
var projectNonInteractive bool

func init() {
	rootCmd.AddCommand(projectCmd)
	projectCmd.AddCommand(projectCreateCmd)
	projectCreateCmd.Flags().BoolVar(&projectDryRun, "dry-run", false,
		"Print commands without executing")
	projectCreateCmd.Flags().BoolVarP(&projectNonInteractive, "yes", "y", false,
		"Non-interactive mode (accept all defaults)")
}

type ProjectConfig struct {
	ProjectID                string
	ProjectNumber            string
	ProjectName              string
	ServiceAccountName       string
	ServiceAccountEmail      string
	ArtifactRegistryName     string
	ArtifactRegistryLocation string
	WorkloadIdentityProvider string
	ArtifactRegistryURL      string
}

func runProjectCreate(cmd *cobra.Command, args []string) error {
	if err := checkGcloud(); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("==============================================")
	fmt.Println("  GCP Project Creation")
	fmt.Println("==============================================")
	fmt.Println()

	cfg := ProjectConfig{}

	if !projectNonInteractive {
		if err := interactiveProjectConfig(&cfg); err != nil {
			return err
		}
	} else {
		cfg.ProjectName = promptProject("Project Name", "my-project")
		cfg.ProjectID = promptProject("Project ID", "my-project")
		cfg.ServiceAccountName = promptProject("Service Account Name", "github-actions")
		cfg.ArtifactRegistryName = promptProject("Artifact Registry Name", "docker")
		cfg.ArtifactRegistryLocation = promptProject("Artifact Registry Location", "us-central1")
	}

	fmt.Println()
	fmt.Println("==============================================")
	fmt.Println("  Project Configuration Summary")
	fmt.Println("==============================================")
	fmt.Printf("  Project Name:         %s\n", cfg.ProjectName)
	fmt.Printf("  Project ID:           %s\n", cfg.ProjectID)
	fmt.Printf("  Service Account:      %s\n", cfg.ServiceAccountName)
	fmt.Printf("  Artifact Registry:    %s (%s)\n", cfg.ArtifactRegistryName, cfg.ArtifactRegistryLocation)
	fmt.Println("==============================================")
	fmt.Println()

	if !projectNonInteractive {
		if !promptConfirm("Proceed with project creation?") {
			fmt.Println("Project creation cancelled.")
			return nil
		}
		fmt.Println()
	}

	steps := []struct {
		name string
		fn   func(ProjectConfig) error
	}{
		{"Creating GCP Project", createGCPProject},
		{"Enabling APIs", enableProjectAPIs},
		{"Creating Service Account", createProjectServiceAccount},
		{"Setting up Workload Identity Federation", setupProjectWorkloadIdentity},
		{"Creating Artifact Registry", createProjectArtifactRegistry},
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
	fmt.Println("  Project Creation Complete!")
	fmt.Println("==============================================")
	fmt.Println()
	fmt.Printf("Project ID: %s\n", cfg.ProjectID)
	fmt.Println("Next steps:")
	fmt.Println("  1. Save the project ID for later use")
	fmt.Println("  2. Run: gcsetup service setup")
	fmt.Println("     (with your GitHub org/repo and cloud run service details)")

	return nil
}

func interactiveProjectConfig(cfg *ProjectConfig) error {
	cfg.ProjectName = promptProject("Project Name", "my-project")
	cfg.ProjectID = promptProject("Project ID (must be globally unique)", "my-project-"+randomSuffix())

	fmt.Println()
	cfg.ServiceAccountName = promptProject("Service Account Name", "github-actions")
	cfg.ArtifactRegistryName = promptProject("Artifact Registry Name", "docker")
	cfg.ArtifactRegistryLocation = promptProject("Artifact Registry Location (e.g., us-central1)", "us-central1")

	return nil
}

func createGCPProject(cfg ProjectConfig) error {
	if projectDryRun {
		fmt.Printf("  [dry-run] gcloud projects create %s --name=\"%s\"\n", cfg.ProjectID, cfg.ProjectName)
		return nil
	}

	fmt.Printf("  Creating project '%s'...\n", cfg.ProjectID)
	cmd := exec.Command("gcloud", "projects", "create", cfg.ProjectID, "--name="+cfg.ProjectName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}
	fmt.Printf("  ✓ Project '%s' created\n", cfg.ProjectID)
	return nil
}

func enableProjectAPIs(cfg ProjectConfig) error {
	apis := []string{
		"cloudresourcemanager.googleapis.com",
		"serviceusage.googleapis.com",
		"iam.googleapis.com",
		"artifactregistry.googleapis.com",
		"iamcredentials.googleapis.com",
		"cloudkms.googleapis.com",
	}

	if projectDryRun {
		for _, api := range apis {
			fmt.Printf("  [dry-run] gcloud services enable %s --project=%s\n", api, cfg.ProjectID)
		}
		return nil
	}

	fmt.Printf("  Enabling APIs for project '%s'...\n", cfg.ProjectID)
	for _, api := range apis {
		fmt.Printf("    Enabling %s\n", api)
		args := []string{"services", "enable", api, "--project=" + cfg.ProjectID}
		cmd := exec.Command("gcloud", args...)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to enable API %s: %w", api, err)
		}
	}
	fmt.Printf("  ✓ APIs enabled\n")
	return nil
}

func createProjectServiceAccount(cfg ProjectConfig) error {
	cfg.ServiceAccountEmail = fmt.Sprintf("%s@%s.iam.gserviceaccount.com", cfg.ServiceAccountName, cfg.ProjectID)

	if projectDryRun {
		fmt.Printf("  [dry-run] gcloud iam service-accounts create %s "+
			"--project=%s\n", cfg.ServiceAccountName, cfg.ProjectID)
		return nil
	}

	fmt.Printf("  Creating service account '%s'...\n", cfg.ServiceAccountName)
	cmd := exec.Command("gcloud", "iam", "service-accounts", "create",
		cfg.ServiceAccountName, "--project="+cfg.ProjectID)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create service account: %w", err)
	}
	fmt.Printf("  ✓ Service account created: %s\n", cfg.ServiceAccountEmail)
	return nil
}

func setupProjectWorkloadIdentity(cfg ProjectConfig) error {
	if projectDryRun {
		fmt.Println("  [dry-run] Setting up Workload Identity Federation")
		fmt.Printf("  [dry-run] gcloud iam workload-identity-pools create "+
			"github-pool --project=%s --location=global\n", cfg.ProjectID)
		fmt.Println("  [dry-run] gcloud iam workload-identity-pools providers create-oidc github-provider ...")
		return nil
	}

	fmt.Println("  Setting up Workload Identity Federation...")

	fmt.Println("    Creating workload identity pool 'github-pool'")
	cmd := exec.Command("gcloud", "iam", "workload-identity-pools", "create", "github-pool",
		"--project="+cfg.ProjectID, "--location=global", "--display-name=GitHub")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create workload identity pool: %w", err)
	}

	fmt.Println("    Creating OIDC provider 'github-provider'")
	cmd = exec.Command("gcloud", "iam", "workload-identity-pools", "providers", "create-oidc", "github-provider",
		"--project="+cfg.ProjectID,
		"--location=global",
		"--workload-identity-pool=github-pool",
		"--display-name=GitHub",
		"--attribute-mapping=google.subject=assertion.sub,assertion.aud=assertion.aud",
		"--issuer-uri=https://token.actions.githubusercontent.com")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create OIDC provider: %w", err)
	}

	fmt.Println("  ✓ Workload Identity Federation configured")
	return nil
}

func createProjectArtifactRegistry(cfg ProjectConfig) error {
	cfg.ArtifactRegistryURL = fmt.Sprintf("%s-docker.pkg.dev/%s/%s", cfg.ArtifactRegistryLocation,
		cfg.ProjectID, cfg.ArtifactRegistryName)

	if projectDryRun {
		fmt.Printf("  [dry-run] gcloud artifacts repositories create %s "+
			"--repository-format=docker --location=%s --project=%s\n",
			cfg.ArtifactRegistryName, cfg.ArtifactRegistryLocation, cfg.ProjectID)
		return nil
	}

	fmt.Printf("  Creating artifact registry '%s'...\n", cfg.ArtifactRegistryName)
	cmd := exec.Command("gcloud", "artifacts", "repositories", "create", cfg.ArtifactRegistryName,
		"--repository-format=docker",
		"--location="+cfg.ArtifactRegistryLocation,
		"--project="+cfg.ProjectID)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create artifact registry: %w", err)
	}
	fmt.Printf("  ✓ Artifact registry created: %s\n", cfg.ArtifactRegistryURL)
	return nil
}

func promptProject(label, defaultVal string) string {
	if projectNonInteractive {
		return defaultVal
	}

	fmt.Printf("%s (default: %s): ", label, defaultVal)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return defaultVal
	}

	return input
}

func randomSuffix() string {
	return fmt.Sprintf("%d", rand.Intn(100000))
}
