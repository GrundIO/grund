package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Saturn-Fintech/grund/internal/application/ports"
	"github.com/Saturn-Fintech/grund/internal/config"
	"github.com/Saturn-Fintech/grund/internal/domain/service"
	"github.com/Saturn-Fintech/grund/internal/ui"
)

const (
	// ProjectName is the static project name for all docker compose commands
	ProjectName = "grund"
)

// dockerComposeService represents the JSON output from docker compose ps
type dockerComposeService struct {
	Name       string `json:"Name"`
	State      string `json:"State"`
	Health     string `json:"Health"`
	Publishers []struct {
		URL           string `json:"URL"`
		TargetPort    int    `json:"TargetPort"`
		PublishedPort int    `json:"PublishedPort"`
		Protocol      string `json:"Protocol"`
	} `json:"Publishers"`
}

// DockerOrchestrator implements ContainerOrchestrator using Docker Compose
// This is an infrastructure adapter
type DockerOrchestrator struct {
	infrastructureFile string   // infrastructure compose file (creates the network)
	serviceFiles       []string // service compose files (use external network)
	workingDir         string
	projectName        string
}

// NewDockerOrchestrator creates a new Docker orchestrator
// projectName ensures consistent container naming across runs
func NewDockerOrchestrator(workingDir string) ports.ContainerOrchestrator {
	return &DockerOrchestrator{
		infrastructureFile: "",
		serviceFiles:       []string{},
		workingDir:         workingDir,
		projectName:        ProjectName,
	}
}

// SetComposeFiles updates the compose files to use
// The first file containing "infrastructure" in the path is treated as the infrastructure file
func (d *DockerOrchestrator) SetComposeFiles(files []string) {
	d.infrastructureFile = ""
	d.serviceFiles = []string{}

	for _, file := range files {
		if strings.Contains(file, "infrastructure") {
			d.infrastructureFile = file
		} else {
			d.serviceFiles = append(d.serviceFiles, file)
		}
	}
}

// composeArgs returns the base docker compose arguments with project name and all compose files
func (d *DockerOrchestrator) composeArgs() []string {
	return d.composeArgsForFiles(d.allComposeFiles())
}

// composeArgsForFiles returns docker compose arguments for specific files
func (d *DockerOrchestrator) composeArgsForFiles(files []string) []string {
	args := []string{"compose", "-p", d.projectName}
	for _, file := range files {
		args = append(args, "-f", file)
	}
	return args
}

// allComposeFiles returns all compose files (infrastructure + services)
func (d *DockerOrchestrator) allComposeFiles() []string {
	files := []string{}
	if d.infrastructureFile != "" {
		files = append(files, d.infrastructureFile)
	}
	files = append(files, d.serviceFiles...)
	return files
}

// infrastructureServices are the services that should be started first
var infrastructureServices = []string{"postgres", "mongodb", "redis", "localstack"}

// StartInfrastructure starts infrastructure containers and waits for them to be healthy
// Only uses the infrastructure compose file to avoid network dependency issues
func (d *DockerOrchestrator) StartInfrastructure(ctx context.Context) error {
	if d.infrastructureFile == "" {
		ui.Debug("No infrastructure compose file, skipping infrastructure startup")
		return nil
	}

	ui.Debug("Reading infrastructure compose file: %s", d.infrastructureFile)

	// Use only infrastructure file for this operation
	infraArgs := d.composeArgsForFiles([]string{d.infrastructureFile})

	// First, get the list of services defined in the infrastructure compose file
	configArgs := append(infraArgs, "config", "--services")
	configCmd := exec.CommandContext(ctx, "docker", configArgs...)
	configCmd.Dir = d.workingDir
	configOutput, err := configCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to read compose config: %w", err)
	}

	// Parse available services
	availableServices := make(map[string]bool)
	for _, line := range strings.Split(string(configOutput), "\n") {
		svc := strings.TrimSpace(line)
		if svc != "" {
			availableServices[svc] = true
		}
	}
	ui.Debug("Available infrastructure services: %v", availableServices)

	// Filter infrastructure services to only those in compose file
	var servicesToStart []string
	for _, svc := range infrastructureServices {
		if availableServices[svc] {
			servicesToStart = append(servicesToStart, svc)
		}
	}

	if len(servicesToStart) == 0 {
		ui.Debug("No infrastructure services to start")
		return nil
	}

	ui.SubStep("Starting: %s", strings.Join(servicesToStart, ", "))

	// Start infrastructure services with --wait to ensure they're healthy
	// Only use infrastructure file to create the network first
	args := append(infraArgs, "up", "-d", "--wait")
	args = append(args, servicesToStart...)

	ui.Debug("Running: docker %s", strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = d.workingDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		ui.Errorf("Docker compose output:\n%s", string(output))
		return fmt.Errorf("failed to start infrastructure: %w", err)
	}

	ui.Debug("Infrastructure started successfully")
	return nil
}

// StartServices starts services using docker-compose
func (d *DockerOrchestrator) StartServices(ctx context.Context, services []service.ServiceName) error {
	serviceNames := make([]string, len(services))
	for i, svc := range services {
		serviceNames[i] = svc.String()
	}

	ui.SubStep("Building and starting: %s", strings.Join(serviceNames, ", "))

	args := append(d.composeArgs(), "up", "-d", "--build")
	args = append(args, serviceNames...)

	ui.Debug("Running: docker %s", strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = d.workingDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		ui.Errorf("Docker compose output:\n%s", string(output))
		return fmt.Errorf("failed to start services: %w", err)
	}

	return nil
}

// StopServices stops all services
func (d *DockerOrchestrator) StopServices(ctx context.Context) error {
	// If no compose files, nothing to stop
	if len(d.allComposeFiles()) == 0 {
		ui.Infof("No services to stop")
		return nil
	}

	ui.Step("Stopping services...")

	args := append(d.composeArgs(), "down")
	ui.Debug("Running: docker %s", strings.Join(args, " "))

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = d.workingDir
	output, err := cmd.CombinedOutput()

	// Show output regardless of error
	if len(output) > 0 {
		// Print each line with proper formatting
		for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
			if line != "" {
				ui.SubStep("%s", line)
			}
		}
	}

	if err != nil {
		return fmt.Errorf("failed to stop services: %w", err)
	}

	ui.Successf("All services stopped")
	return nil
}

// RestartService restarts a specific service
func (d *DockerOrchestrator) RestartService(ctx context.Context, name service.ServiceName) error {
	ui.Step("Restarting %s...", name.String())

	args := append(d.composeArgs(), "restart", name.String())
	ui.Debug("Running: docker %s", strings.Join(args, " "))

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = d.workingDir
	output, err := cmd.CombinedOutput()

	// Show output regardless of error
	if len(output) > 0 {
		for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
			if line != "" {
				ui.SubStep("%s", line)
			}
		}
	}

	if err != nil {
		return fmt.Errorf("failed to restart service %s: %w", name, err)
	}

	ui.Successf("Service %s restarted", name.String())
	return nil
}

// GetServiceStatus gets the status of a service
func (d *DockerOrchestrator) GetServiceStatus(ctx context.Context, name service.ServiceName) (ports.ServiceStatus, error) {
	// Use docker compose ps to get status
	args := append(d.composeArgs(), "ps", "--format", "json", name.String())
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = d.workingDir
	output, err := cmd.Output()
	if err != nil {
		return ports.ServiceStatus{
			Name:   name.String(),
			Status: "not running",
			Health: "unknown",
		}, nil
	}

	outputStr := strings.TrimSpace(string(output))
	if outputStr == "" {
		return ports.ServiceStatus{
			Name:   name.String(),
			Status: "not running",
			Health: "unknown",
		}, nil
	}

	// Parse JSON output
	var svc dockerComposeService
	if err := json.Unmarshal([]byte(outputStr), &svc); err != nil {
		// Fallback to simple parsing if JSON fails
		return d.parseStatusFallback(name.String(), outputStr), nil
	}

	status := ports.ServiceStatus{
		Name:   name.String(),
		Status: strings.ToLower(svc.State),
		Health: svc.Health,
	}

	if status.Health == "" {
		status.Health = "-"
	}

	// Build localhost URL from publishers
	if len(svc.Publishers) > 0 {
		for _, pub := range svc.Publishers {
			if pub.PublishedPort > 0 {
				status.Endpoint = fmt.Sprintf("http://localhost:%d", pub.PublishedPort)
				break // Use first published port
			}
		}
	}

	return status, nil
}

// parseStatusFallback handles cases where JSON parsing fails
func (d *DockerOrchestrator) parseStatusFallback(name, outputStr string) ports.ServiceStatus {
	status := ports.ServiceStatus{
		Name:   name,
		Status: "unknown",
		Health: "unknown",
	}

	if strings.Contains(outputStr, `"State":"running"`) || strings.Contains(outputStr, `"State": "running"`) {
		status.Status = "running"
	} else if strings.Contains(outputStr, `"State":"exited"`) || strings.Contains(outputStr, `"State": "exited"`) {
		status.Status = "exited"
	}

	if strings.Contains(outputStr, `"Health":"healthy"`) || strings.Contains(outputStr, `"Health": "healthy"`) {
		status.Health = "healthy"
	} else if strings.Contains(outputStr, `"Health":"unhealthy"`) || strings.Contains(outputStr, `"Health": "unhealthy"`) {
		status.Health = "unhealthy"
	} else if strings.Contains(outputStr, `"Health":""`) || strings.Contains(outputStr, `"Health": ""`) {
		status.Health = "-"
	}

	return status
}

// GetAllServiceStatuses gets status of all services in the compose file
func (d *DockerOrchestrator) GetAllServiceStatuses(ctx context.Context) ([]ports.ServiceStatus, error) {
	// If no compose files, return empty list
	allFiles := d.allComposeFiles()
	if len(allFiles) == 0 {
		return []ports.ServiceStatus{}, nil
	}

	// Get all services from compose file
	configArgs := append(d.composeArgs(), "config", "--services")
	configCmd := exec.CommandContext(ctx, "docker", configArgs...)
	configCmd.Dir = d.workingDir
	configOutput, err := configCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to read compose config: %w", err)
	}

	var statuses []ports.ServiceStatus
	for _, line := range strings.Split(string(configOutput), "\n") {
		svcName := strings.TrimSpace(line)
		if svcName == "" {
			continue
		}
		status, _ := d.GetServiceStatus(ctx, service.ServiceName(svcName))
		statuses = append(statuses, status)
	}

	return statuses, nil
}

// GetLogs gets logs for a service
func (d *DockerOrchestrator) GetLogs(ctx context.Context, name service.ServiceName, follow bool, tail int) (ports.LogStream, error) {
	// TODO: Implement log streaming
	return nil, fmt.Errorf("not implemented")
}

// GetGrundTmpDir returns the path to ~/.grund/tmp
func GetGrundTmpDir() (string, error) {
	grundHome, err := config.GetGrundHome()
	if err != nil {
		return "", fmt.Errorf("failed to get grund home: %w", err)
	}
	return filepath.Join(grundHome, "tmp"), nil
}

// GetInfrastructureComposePath returns the path to the infrastructure compose file
func GetInfrastructureComposePath() (string, error) {
	tmpDir, err := GetGrundTmpDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(tmpDir, "infrastructure", "docker-compose.yaml"), nil
}

// GetServiceComposePath returns the path to a service's compose file
func GetServiceComposePath(serviceName string) (string, error) {
	tmpDir, err := GetGrundTmpDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(tmpDir, serviceName, "docker-compose.yaml"), nil
}

// DiscoverComposeFiles scans ~/.grund/tmp/ and returns all compose files found
func DiscoverComposeFiles() (*ports.ComposeFileSet, error) {
	tmpDir, err := GetGrundTmpDir()
	if err != nil {
		return nil, err
	}

	fileSet := &ports.ComposeFileSet{
		ServicePaths: make(map[string]string),
	}

	// Check if tmp directory exists
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		return fileSet, nil
	}

	// Scan all subdirectories for docker-compose.yaml
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read tmp directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		composePath := filepath.Join(tmpDir, entry.Name(), "docker-compose.yaml")
		if _, err := os.Stat(composePath); os.IsNotExist(err) {
			continue
		}

		if entry.Name() == "infrastructure" {
			fileSet.InfrastructurePath = composePath
		} else {
			fileSet.ServicePaths[entry.Name()] = composePath
		}
	}

	return fileSet, nil
}

// StopAllProjects stops all services using discovered compose files
func StopAllProjects(ctx context.Context) error {
	fileSet, err := DiscoverComposeFiles()
	if err != nil {
		return fmt.Errorf("failed to discover compose files: %w", err)
	}

	allPaths := fileSet.AllPaths()
	if len(allPaths) == 0 {
		ui.Infof("No services found to stop")
		return nil
	}

	ui.Infof("Stopping all Grund services...")

	// Build docker compose command with all files
	args := []string{"compose", "-p", ProjectName}
	for _, path := range allPaths {
		args = append(args, "-f", path)
	}
	args = append(args, "down")

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		ui.Warnf("Failed to stop services: %s", string(output))
		return fmt.Errorf("failed to stop services: %w", err)
	}

	ui.Successf("All services stopped")
	return nil
}

// GetProjectName returns the static project name "grund"
// This is used as the docker compose project name for consistent container naming
func GetProjectName() string {
	return ProjectName
}
