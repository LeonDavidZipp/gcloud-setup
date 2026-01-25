package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"gcsetup/embedded"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize project with workflow and .env.gcloud template",
	Long: `Creates the following files in your project:
  - .github/workflows/gcloud-deploy.yml  (CI/CD workflow)
  - .env.gcloud                   (environment variables template)`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

// runInit creates the workflow file and environment template in the current directory.
func runInit(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Create .github/workflows directory
	workflowDir := filepath.Join(cwd, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0755); err != nil {
		return fmt.Errorf("failed to create .github/workflows directory: %w", err)
	}

	// Write gcloud-deploy.yml
	workflowPath := filepath.Join(workflowDir, "gcloud-deploy.yml")
	if err := writeFileIfNotExists(workflowPath, embedded.DeployWorkflow); err != nil {
		return err
	}
	fmt.Println("✓ Created", workflowPath)

	// Write .env.gcloud
	envPath := filepath.Join(cwd, ".env.gcloud")
	if err := writeFileIfNotExists(envPath, embedded.EnvTemplate); err != nil {
		return err
	}
	fmt.Println("✓ Created", envPath)

	// Add .env.gcloud to .gitignore if exists
	gitignorePath := filepath.Join(cwd, ".gitignore")
	if err := appendToGitignore(gitignorePath); err != nil {
		fmt.Println("⚠ Could not update .gitignore:", err)
	} else {
		fmt.Println("✓ Updated .gitignore")
	}

	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Edit .env.gcloud with your project values")
	fmt.Println("  2. Run: gc setup")

	return nil
}

// writeFileIfNotExists creates a file with the given content only if it does not already exist.
func writeFileIfNotExists(path string, content []byte) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("file already exists: %s", path)
	}
	return os.WriteFile(path, content, 0644)
}

// appendToGitignore adds the .env.gcloud entry to .gitignore if not already present.
func appendToGitignore(path string) error {
	entry := "\n# GCloud setup\n.env.gcloud\n"

	// Check if file exists and already contains entry
	if data, err := os.ReadFile(path); err == nil {
		if contains(string(data), ".env.gcloud") {
			return nil
		}
		// Append to existing file
		f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()
		_, err = f.WriteString(entry)
		return err
	}

	return os.WriteFile(path, []byte(entry), 0644)
}

// contains checks if string s contains the substring substr.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

// containsSubstring performs a simple substring search.
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
