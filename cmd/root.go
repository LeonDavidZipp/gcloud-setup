package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "gcsetup",
	Short: "GCloud project setup CLI",
	Long: `A CLI tool to set up GCloud projects with GitHub Actions CI/CD.

Commands:
  gcsetup init   - Initialize project with workflow and .env.gcloud template
  gcsetup setup  - Set up GCloud project and GitHub repository`,
}

// Execute runs the root command and exits on error.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is .env.gcloud)")

	// GCP flags
	rootCmd.PersistentFlags().String("gcp-project-id", "", "GCP Project ID")
	rootCmd.PersistentFlags().String("gcp-project-number", "", "GCP Project Number")
	rootCmd.PersistentFlags().String("service-account-name", "", "Service Account Name")
	rootCmd.PersistentFlags().String("artifact-registry-name", "", "Artifact Registry Name")
	rootCmd.PersistentFlags().String("artifact-registry-location", "", "Artifact Registry Location")

	// GitHub flags
	rootCmd.PersistentFlags().String("github-org", "", "GitHub Organization")
	rootCmd.PersistentFlags().String("github-repo", "", "GitHub Repository")

	// Cloud Run flags
	rootCmd.PersistentFlags().String("cloud-run-service", "", "Cloud Run Service Name")
	rootCmd.PersistentFlags().String("cloud-run-region", "", "Cloud Run Region")

	// Bind flags to viper
	_ = viper.BindPFlag("GCP_PROJECT_ID", rootCmd.PersistentFlags().Lookup("gcp-project-id"))
	_ = viper.BindPFlag("GCP_PROJECT_NUMBER", rootCmd.PersistentFlags().Lookup("gcp-project-number"))
	_ = viper.BindPFlag("SERVICE_ACCOUNT_NAME", rootCmd.PersistentFlags().Lookup("service-account-name"))
	_ = viper.BindPFlag("ARTIFACT_REGISTRY_NAME", rootCmd.PersistentFlags().Lookup("artifact-registry-name"))
	_ = viper.BindPFlag("ARTIFACT_REGISTRY_LOCATION", rootCmd.PersistentFlags().Lookup("artifact-registry-location"))
	_ = viper.BindPFlag("GITHUB_ORGANIZATION", rootCmd.PersistentFlags().Lookup("github-org"))
	_ = viper.BindPFlag("GITHUB_REPOSITORY", rootCmd.PersistentFlags().Lookup("github-repo"))
	_ = viper.BindPFlag("CLOUD_RUN_SERVICE", rootCmd.PersistentFlags().Lookup("cloud-run-service"))
	_ = viper.BindPFlag("CLOUD_RUN_REGION", rootCmd.PersistentFlags().Lookup("cloud-run-region"))
}

// initConfig initializes viper configuration from file and environment variables.
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName(".env.gcloud")
		viper.SetConfigType("env")
		viper.AddConfigPath(".")
	}

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

// RequiredVars lists all environment variables required for setup.
var RequiredVars = []string{
	"GCP_PROJECT_ID",
	"GCP_PROJECT_NUMBER",
	"GITHUB_ORGANIZATION",
	"GITHUB_REPOSITORY",
	"SERVICE_ACCOUNT_NAME",
	"ARTIFACT_REGISTRY_NAME",
	"ARTIFACT_REGISTRY_LOCATION",
	"CLOUD_RUN_SERVICE",
	"CLOUD_RUN_REGION",
}

// ValidateConfig verifies that all required environment variables are set.
func ValidateConfig() error {
	var missing []string
	for _, v := range RequiredVars {
		if viper.GetString(v) == "" {
			missing = append(missing, v)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required variables:\n  - %s", strings.Join(missing, "\n  - "))
	}
	return nil
}
