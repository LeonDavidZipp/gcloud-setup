package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var loadbalancerCmd = &cobra.Command{
	Use:   "loadbalancer",
	Short: "Manage Google Cloud Load Balancers",
	Long:  `Commands for setting up and managing Google Cloud Load Balancers with multiple backend services.`,
}

var lbSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Configure a load balancer for multiple services",
	Long: `Set up a Google Cloud Load Balancer with multiple backend services:
  1. Create health checks for each service
  2. Configure backend services with custom URL routing
  3. Set up URL maps for path-based routing
  4. Create target HTTP(S) proxies
  5. Configure frontend IPs and forwarding rules`,
	RunE: runLoadBalancer,
}

var lbDryRun bool
var lbNonInteractive bool

func init() {
	rootCmd.AddCommand(loadbalancerCmd)
	loadbalancerCmd.AddCommand(lbSetupCmd)
	lbSetupCmd.Flags().BoolVar(&lbDryRun, "dry-run", false, "Print commands without executing")
	lbSetupCmd.Flags().BoolVarP(&lbNonInteractive, "yes", "y", false, "Non-interactive mode (accept all defaults)")
}

type LoadBalancerService struct {
	Name        string
	Protocol    string
	Port        int
	Path        string
	HealthCheck string
}

type LoadBalancerConfig struct {
	ProjectID       string
	ProjectNumber   string
	LBName          string
	Region          string
	Network         string
	Subnet          string
	Services        []LoadBalancerService
	HealthCheckPort int
	UseSSL          bool
	SSLCertificate  string
}

func runLoadBalancer(cmd *cobra.Command, args []string) error {
	if err := checkGcloud(); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("==============================================")
	fmt.Println("  Load Balancer Configuration")
	fmt.Println("==============================================")
	fmt.Println()

	cfg := LoadBalancerConfig{
		ProjectID:     viper.GetString("GCP_PROJECT_ID"),
		ProjectNumber: viper.GetString("GCP_PROJECT_NUMBER"),
		Region:        viper.GetString("CLOUD_RUN_REGION"),
	}

	if cfg.ProjectID == "" {
		return fmt.Errorf("GCP_PROJECT_ID is required")
	}

	if !lbNonInteractive {
		if err := interactiveLBConfig(&cfg); err != nil {
			return err
		}
	} else {
		cfg.LBName = promptLB("Load Balancer Name", "gcloud-lb")
		cfg.Network = promptLB("Network", "default")
		cfg.Subnet = promptLB("Subnet (leave empty for auto)", "")
		cfg.HealthCheckPort = 8080
		cfg.UseSSL = false
	}

	fmt.Println()
	fmt.Println("==============================================")
	fmt.Println("  Load Balancer Configuration Summary")
	fmt.Println("==============================================")
	fmt.Printf("  Name:                 %s\n", cfg.LBName)
	fmt.Printf("  Network:              %s\n", cfg.Network)
	fmt.Printf("  Health Check Port:    %d\n", cfg.HealthCheckPort)
	fmt.Printf("  Use SSL:              %v\n", cfg.UseSSL)
	fmt.Printf("  Number of Services:   %d\n", len(cfg.Services))
	fmt.Println()
	for i, svc := range cfg.Services {
		fmt.Printf("  Service %d: %s\n", i+1, svc.Name)
		fmt.Printf("    Protocol: %s, Port: %d, Path: %s\n", svc.Protocol, svc.Port, svc.Path)
	}
	fmt.Println("==============================================")
	fmt.Println()

	if !lbNonInteractive {
		if !promptConfirm("Proceed with load balancer configuration?") {
			fmt.Println("Configuration cancelled.")
			return nil
		}
		fmt.Println()
	}

	steps := []struct {
		name string
		fn   func(LoadBalancerConfig) error
	}{
		{"Creating Health Checks", createHealthChecks},
		{"Creating Backend Services", createBackendServices},
		{"Creating URL Map", createURLMap},
		{"Creating HTTP(S) Proxy", createHTTPSProxy},
		{"Creating Forwarding Rule", createForwardingRule},
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
	fmt.Println("  Load Balancer Configuration Complete!")
	fmt.Println("==============================================")
	fmt.Println()
	fmt.Printf("Load Balancer Name: %s\n", cfg.LBName)
	fmt.Println("Next steps:")
	fmt.Println("  1. Get the load balancer IP:")
	fmt.Printf("     gcloud compute forwarding-rules describe %s --global\n", cfg.LBName+"-forwarding-rule")
	fmt.Println("  2. Create a DNS record pointing to the load balancer IP")
	fmt.Println("  3. Test the configuration with curl")

	return nil
}

func interactiveLBConfig(cfg *LoadBalancerConfig) error {
	reader := bufio.NewReader(os.Stdin)

	cfg.LBName = promptLB("Load Balancer Name", "gcloud-lb")
	cfg.Network = promptLB("Network", "default")
	cfg.Subnet = promptLB("Subnet (leave empty for auto)", "")

	fmt.Println()
	fmt.Print("Health Check Port (default 8080): ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		cfg.HealthCheckPort = 8080
	} else {
		if _, err := fmt.Sscanf(input, "%d", &cfg.HealthCheckPort); err != nil {
			cfg.HealthCheckPort = 8080
		}
	}

	fmt.Println()
	fmt.Print("Use SSL/TLS? (y/n, default: n): ")
	input, _ = reader.ReadString('\n')
	cfg.UseSSL = strings.ToLower(strings.TrimSpace(input)) == "y"

	if cfg.UseSSL {
		cfg.SSLCertificate = promptLB("SSL Certificate Name", "")
		if cfg.SSLCertificate == "" {
			fmt.Println("⚠ SSL certificate name is required for HTTPS. Using HTTP instead.")
			cfg.UseSSL = false
		}
	}

	fmt.Println()
	fmt.Println("Configure backend services:")
	fmt.Print("How many services? (default: 1): ")
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(input)
	numServices := 1
	if input != "" {
		if _, err := fmt.Sscanf(input, "%d", &numServices); err != nil {
			numServices = 1
		}
	}

	fmt.Println()
	for i := 0; i < numServices; i++ {
		fmt.Printf("Service %d:\n", i+1)
		service := LoadBalancerService{}
		service.Name = promptLB("  Service name", fmt.Sprintf("service-%d", i+1))
		service.Protocol = promptLB("  Protocol (HTTP/HTTPS)", "HTTP")
		fmt.Printf("  Port (default 8080): ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		service.Port = 8080
		if input != "" {
			if _, err := fmt.Sscanf(input, "%d", &service.Port); err != nil {
				service.Port = 8080
			}
		}
		service.Path = promptLB("  URL Path (e.g., /api/*)", fmt.Sprintf("/%s/*", service.Name))
		service.HealthCheck = fmt.Sprintf("%s-hc", service.Name)

		cfg.Services = append(cfg.Services, service)
		fmt.Println()
	}

	return nil
}

func createHealthChecks(cfg LoadBalancerConfig) error {
	for _, service := range cfg.Services {
		if lbDryRun {
			cmd := fmt.Sprintf(
				`gcloud compute health-checks create http "%s" \
  --global \
  --port=%d \
  --request-path="/healthz" \
  --project="%s"`,
				service.HealthCheck,
				cfg.HealthCheckPort,
				cfg.ProjectID,
			)
			fmt.Println(cmd)
			continue
		}

		if err := exec.Command("gcloud", "compute", "health-checks", "describe",
			service.HealthCheck, "--global", "--project", cfg.ProjectID).Run(); err == nil {
			fmt.Printf("  ✓ Health check '%s' already exists\n", service.HealthCheck)
			continue
		}

		fmt.Printf("  Creating health check '%s'...\n", service.HealthCheck)
		parts := []string{
			"compute", "health-checks", "create", "http", service.HealthCheck,
			"--global",
			fmt.Sprintf("--port=%d", cfg.HealthCheckPort),
			"--request-path=/healthz",
			"--project=" + cfg.ProjectID,
		}
		if err := exec.Command("gcloud", parts...).Run(); err != nil {
			return fmt.Errorf("failed to create health check: %w", err)
		}
		fmt.Printf("  ✓ Health check '%s' created\n", service.HealthCheck)
	}

	return nil
}

func createBackendServices(cfg LoadBalancerConfig) error {
	for i, service := range cfg.Services {
		backendName := fmt.Sprintf("%s-backend", service.Name)

		protocol := "HTTP"
		if service.Protocol == "HTTPS" {
			protocol = "HTTPS"
		}

		if lbDryRun {
			fmt.Printf("  [dry-run] Creating backend service '%s'\n", backendName)
			continue
		}

		if err := exec.Command("gcloud", "compute", "backend-services", "describe",
			backendName, "--global", "--project", cfg.ProjectID).Run(); err == nil {
			fmt.Printf("  ✓ Backend service '%s' already exists\n", backendName)
			cfg.Services[i].HealthCheck = backendName
			continue
		}

		fmt.Printf("  Creating backend service '%s'...\n", backendName)
		parts := []string{
			"compute", "backend-services", "create", backendName,
			"--global",
			"--protocol=" + protocol,
			"--port-name=http",
			"--health-checks=" + service.HealthCheck,
			"--load-balancing-scheme=EXTERNAL",
			"--enable-cdn",
			"--project=" + cfg.ProjectID,
		}
		if err := exec.Command("gcloud", parts...).Run(); err != nil {
			return fmt.Errorf("failed to create backend service: %w", err)
		}
		fmt.Printf("  ✓ Backend service '%s' created\n", backendName)
		cfg.Services[i].HealthCheck = backendName
	}

	return nil
}

func createURLMap(cfg LoadBalancerConfig) error {
	urlMapName := fmt.Sprintf("%s-url-map", cfg.LBName)

	if lbDryRun {
		fmt.Printf("  [dry-run] Creating URL map '%s'\n", urlMapName)
		return nil
	}

	if err := exec.Command("gcloud", "compute", "url-maps", "describe",
		urlMapName, "--project", cfg.ProjectID).Run(); err == nil {
		fmt.Printf("  ✓ URL map '%s' already exists\n", urlMapName)
		return nil
	}

	defaultBackend := fmt.Sprintf("%s-backend", cfg.Services[0].Name)

	fmt.Printf("  Creating URL map '%s'...\n", urlMapName)
	parts := []string{
		"compute", "url-maps", "create", urlMapName,
		"--default-service=" + defaultBackend,
		"--project=" + cfg.ProjectID,
	}

	if err := exec.Command("gcloud", parts...).Run(); err != nil {
		return fmt.Errorf("failed to create URL map: %w", err)
	}

	// Add path rules for each service
	for _, service := range cfg.Services {
		backendName := fmt.Sprintf("%s-backend", service.Name)

		parts := []string{
			"compute", "url-maps", "add-path-rule", urlMapName,
			"--service=" + backendName,
			"--path-pattern=" + service.Path,
			"--project=" + cfg.ProjectID,
		}

		fmt.Printf("  Adding path rule '%s' -> %s\n", service.Path, backendName)
		if err := exec.Command("gcloud", parts...).Run(); err != nil {
			fmt.Printf("  ⚠ Could not add path rule (may already exist): %v\n", err)
		}
	}

	fmt.Printf("  ✓ URL map '%s' created\n", urlMapName)
	return nil
}

func createHTTPSProxy(cfg LoadBalancerConfig) error {
	proxyName := fmt.Sprintf("%s-proxy", cfg.LBName)
	urlMapName := fmt.Sprintf("%s-url-map", cfg.LBName)

	if lbDryRun {
		fmt.Printf("  [dry-run] Creating HTTP(S) proxy '%s'\n", proxyName)
		return nil
	}

	protocol := "HTTP"
	if cfg.UseSSL {
		protocol = "HTTPS"
	}

	if err := exec.Command("gcloud", "compute",
		fmt.Sprintf("%s-proxies", strings.ToLower(protocol)), "describe",
		proxyName, "--global", "--project", cfg.ProjectID).Run(); err == nil {
		fmt.Printf("  ✓ %s proxy '%s' already exists\n", protocol, proxyName)
		return nil
	}

	fmt.Printf("  Creating %s proxy '%s'...\n", protocol, proxyName)

	if protocol == "HTTP" {
		parts := []string{
			"compute", "http-proxies", "create", proxyName,
			"--url-map=" + urlMapName,
			"--project=" + cfg.ProjectID,
		}
		if err := exec.Command("gcloud", parts...).Run(); err != nil {
			return fmt.Errorf("failed to create HTTP proxy: %w", err)
		}
	} else {
		parts := []string{
			"compute", "https-proxies", "create", proxyName,
			"--url-map=" + urlMapName,
			"--ssl-certificates=" + cfg.SSLCertificate,
			"--project=" + cfg.ProjectID,
		}
		if err := exec.Command("gcloud", parts...).Run(); err != nil {
			return fmt.Errorf("failed to create HTTPS proxy: %w", err)
		}
	}

	fmt.Printf("  ✓ %s proxy '%s' created\n", protocol, proxyName)
	return nil
}

func createForwardingRule(cfg LoadBalancerConfig) error {
	ruleName := fmt.Sprintf("%s-forwarding-rule", cfg.LBName)
	proxyName := fmt.Sprintf("%s-proxy", cfg.LBName)
	protocol := "HTTP"
	port := 80

	if cfg.UseSSL {
		protocol = "HTTPS"
		port = 443
	}

	if lbDryRun {
		fmt.Printf("  [dry-run] Creating forwarding rule '%s'\n", ruleName)
		return nil
	}

	if err := exec.Command("gcloud", "compute", "forwarding-rules", "describe",
		ruleName, "--global", "--project", cfg.ProjectID).Run(); err == nil {
		fmt.Printf("  ✓ Forwarding rule '%s' already exists\n", ruleName)
		return nil
	}

	fmt.Printf("  Creating forwarding rule '%s'...\n", ruleName)

	parts := []string{
		"compute", "forwarding-rules", "create", ruleName,
		"--global",
		fmt.Sprintf("--target-%s-proxy=%s", strings.ToLower(protocol), proxyName),
		fmt.Sprintf("--address=%s-ip", cfg.LBName),
		fmt.Sprintf("--ports=%d", port),
		"--project=" + cfg.ProjectID,
	}

	if err := exec.Command("gcloud", parts...).Run(); err != nil {
		return fmt.Errorf("failed to create forwarding rule: %w", err)
	}

	fmt.Printf("  ✓ Forwarding rule '%s' created\n", ruleName)
	return nil
}

func promptLB(label, defaultVal string) string {
	if lbNonInteractive {
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
