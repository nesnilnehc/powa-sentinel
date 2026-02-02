package github_actions

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// GoReleaserConfig represents the structure of .goreleaser.yaml
type GoReleaserConfig struct {
	Dockers []DockerConfig `yaml:"dockers"`
	Release ReleaseConfig  `yaml:"release"`
}

type DockerConfig struct {
	ImageTemplates      []string `yaml:"image_templates"`
	Dockerfile          string   `yaml:"dockerfile"`
	BuildFlagTemplates  []string `yaml:"build_flag_templates"`
}

type ReleaseConfig struct {
	GitHub struct {
		Owner string `yaml:"owner"`
		Name  string `yaml:"name"`
	} `yaml:"github"`
}

// TestGoReleaserCoordination validates that GitHub Actions workflow
// coordinates properly with GoReleaser Docker publishing
func TestGoReleaserCoordination(t *testing.T) {
	// Test that GoReleaser configuration exists and is valid
	t.Run("GoReleaserConfigExists", func(t *testing.T) {
		goreleaserPath := filepath.Join("..", "..", ".goreleaser.yaml")
		if _, err := os.Stat(goreleaserPath); os.IsNotExist(err) {
			t.Fatal("GoReleaser configuration not found at expected location")
		}
	})

	// Test that GoReleaser and GitHub Actions use same registry and repository
	t.Run("RegistryConsistency", func(t *testing.T) {
		// Read GoReleaser config
		goreleaserPath := filepath.Join("..", "..", ".goreleaser.yaml")
		goreleaserContent, err := os.ReadFile(goreleaserPath)
		if err != nil {
			t.Fatalf("Failed to read GoReleaser config: %v", err)
		}

		var goreleaserConfig GoReleaserConfig
		if err := yaml.Unmarshal(goreleaserContent, &goreleaserConfig); err != nil {
			t.Fatalf("Failed to parse GoReleaser config: %v", err)
		}

		// Read GitHub Actions workflow
		workflowPath := filepath.Join("..", "..", ".github", "workflows", "docker-publish.yml")
		workflowContent, err := os.ReadFile(workflowPath)
		if err != nil {
			t.Fatalf("Failed to read workflow file: %v", err)
		}

		workflowStr := string(workflowContent)

		// Extract registry and image name from workflow
		registryPattern := regexp.MustCompile(`REGISTRY:\s*(.+)`)
		imageNamePattern := regexp.MustCompile(`IMAGE_NAME:\s*(.+)`)

		registryMatch := registryPattern.FindStringSubmatch(workflowStr)
		imageNameMatch := imageNamePattern.FindStringSubmatch(workflowStr)

		if len(registryMatch) < 2 || len(imageNameMatch) < 2 {
			t.Fatal("Could not extract registry and image name from workflow")
		}

		workflowRegistry := strings.TrimSpace(registryMatch[1])
		workflowImageName := strings.TrimSpace(imageNameMatch[1])

		// Check GoReleaser image templates match workflow configuration
		if len(goreleaserConfig.Dockers) == 0 {
			t.Fatal("GoReleaser has no Docker configuration")
		}

		dockerConfig := goreleaserConfig.Dockers[0]
		for _, template := range dockerConfig.ImageTemplates {
			expectedPrefix := workflowRegistry + "/" + workflowImageName + ":"
			if !strings.HasPrefix(template, expectedPrefix) {
				t.Errorf("GoReleaser image template %s doesn't match workflow registry/image (%s)", template, expectedPrefix)
			}
		}
	})

	// Test that both use the same Dockerfile
	t.Run("DockerfileConsistency", func(t *testing.T) {
		goreleaserPath := filepath.Join("..", "..", ".goreleaser.yaml")
		goreleaserContent, err := os.ReadFile(goreleaserPath)
		if err != nil {
			t.Fatalf("Failed to read GoReleaser config: %v", err)
		}

		var goreleaserConfig GoReleaserConfig
		if err := yaml.Unmarshal(goreleaserContent, &goreleaserConfig); err != nil {
			t.Fatalf("Failed to parse GoReleaser config: %v", err)
		}

		// Check that GoReleaser uses the same Dockerfile
		if len(goreleaserConfig.Dockers) == 0 {
			t.Fatal("GoReleaser has no Docker configuration")
		}

		dockerConfig := goreleaserConfig.Dockers[0]
		if dockerConfig.Dockerfile != "Dockerfile" {
			t.Errorf("GoReleaser should use 'Dockerfile', but uses '%s'", dockerConfig.Dockerfile)
		}

		// Verify the Dockerfile exists
		dockerfilePath := filepath.Join("..", "..", dockerConfig.Dockerfile)
		if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
			t.Errorf("Dockerfile specified in GoReleaser (%s) does not exist", dockerConfig.Dockerfile)
		}
	})

	// Test that build arguments are consistent
	t.Run("BuildArgsConsistency", func(t *testing.T) {
		goreleaserPath := filepath.Join("..", "..", ".goreleaser.yaml")
		goreleaserContent, err := os.ReadFile(goreleaserPath)
		if err != nil {
			t.Fatalf("Failed to read GoReleaser config: %v", err)
		}

		var goreleaserConfig GoReleaserConfig
		if err := yaml.Unmarshal(goreleaserContent, &goreleaserConfig); err != nil {
			t.Fatalf("Failed to parse GoReleaser config: %v", err)
		}

		if len(goreleaserConfig.Dockers) == 0 {
			t.Fatal("GoReleaser has no Docker configuration")
		}

		dockerConfig := goreleaserConfig.Dockers[0]

		// Check that GoReleaser uses the same build arguments as the workflow
		expectedBuildArgs := []string{"VERSION", "COMMIT", "BUILD_DATE"}
		
		for _, expectedArg := range expectedBuildArgs {
			found := false
			for _, buildFlag := range dockerConfig.BuildFlagTemplates {
				if strings.Contains(buildFlag, "--build-arg="+expectedArg+"=") {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("GoReleaser missing build argument for %s", expectedArg)
			}
		}
	})

	// Test tagging strategy coordination
	t.Run("TaggingStrategyCoordination", func(t *testing.T) {
		// Read GoReleaser config
		goreleaserPath := filepath.Join("..", "..", ".goreleaser.yaml")
		goreleaserContent, err := os.ReadFile(goreleaserPath)
		if err != nil {
			t.Fatalf("Failed to read GoReleaser config: %v", err)
		}

		var goreleaserConfig GoReleaserConfig
		if err := yaml.Unmarshal(goreleaserContent, &goreleaserConfig); err != nil {
			t.Fatalf("Failed to parse GoReleaser config: %v", err)
		}

		// Read GitHub Actions workflow
		workflowPath := filepath.Join("..", "..", ".github", "workflows", "docker-publish.yml")
		workflowContent, err := os.ReadFile(workflowPath)
		if err != nil {
			t.Fatalf("Failed to read workflow file: %v", err)
		}

		workflowStr := string(workflowContent)

		if len(goreleaserConfig.Dockers) == 0 {
			t.Fatal("GoReleaser has no Docker configuration")
		}

		dockerConfig := goreleaserConfig.Dockers[0]

		// GoReleaser should create tags for releases: {{ .Tag }} and latest
		hasTagTemplate := false
		hasLatestTemplate := false

		for _, template := range dockerConfig.ImageTemplates {
			if strings.Contains(template, "{{ .Tag }}") {
				hasTagTemplate = true
			}
			if strings.Contains(template, ":latest") {
				hasLatestTemplate = true
			}
		}

		if !hasTagTemplate {
			t.Error("GoReleaser should have image template with {{ .Tag }} for version tags")
		}

		if !hasLatestTemplate {
			t.Error("GoReleaser should have image template with :latest tag")
		}

		// Workflow should handle tag-based publishing
		if !strings.Contains(workflowStr, "type=ref,event=tag") {
			t.Error("Workflow should handle tag-based publishing with type=ref,event=tag")
		}

		// Workflow should create latest tag for default branch
		if !strings.Contains(workflowStr, "type=raw,value=latest") {
			t.Error("Workflow should create latest tag for default branch")
		}
	})

	// Test that workflow coordinates with GoReleaser releases
	t.Run("ReleaseCoordination", func(t *testing.T) {
		// Read GitHub Actions workflow
		workflowPath := filepath.Join("..", "..", ".github", "workflows", "docker-publish.yml")
		workflowContent, err := os.ReadFile(workflowPath)
		if err != nil {
			t.Fatalf("Failed to read workflow file: %v", err)
		}

		workflowStr := string(workflowContent)

		// Workflow should trigger on release events to coordinate with GoReleaser
		if !strings.Contains(workflowStr, "release:") {
			t.Error("Workflow should trigger on release events to coordinate with GoReleaser")
		}

		if !strings.Contains(workflowStr, "types: [published]") {
			t.Error("Workflow should trigger on published releases")
		}

		// Workflow should also trigger on tag pushes (for manual tags)
		if !strings.Contains(workflowStr, "tags:") {
			t.Error("Workflow should trigger on tag pushes")
		}

		// Check for coordination logic - workflow should handle both scenarios
		if !strings.Contains(workflowStr, "github.ref_type") {
			t.Error("Workflow should check ref_type to distinguish between tag and release events")
		}
	})

	// Test repository ownership consistency
	t.Run("RepositoryOwnershipConsistency", func(t *testing.T) {
		// Read GoReleaser config
		goreleaserPath := filepath.Join("..", "..", ".goreleaser.yaml")
		goreleaserContent, err := os.ReadFile(goreleaserPath)
		if err != nil {
			t.Fatalf("Failed to read GoReleaser config: %v", err)
		}

		var goreleaserConfig GoReleaserConfig
		if err := yaml.Unmarshal(goreleaserContent, &goreleaserConfig); err != nil {
			t.Fatalf("Failed to parse GoReleaser config: %v", err)
		}

		// Read GitHub Actions workflow
		workflowPath := filepath.Join("..", "..", ".github", "workflows", "docker-publish.yml")
		workflowContent, err := os.ReadFile(workflowPath)
		if err != nil {
			t.Fatalf("Failed to read workflow file: %v", err)
		}

		workflowStr := string(workflowContent)

		// Extract repository owner from GoReleaser
		goreleaserOwner := goreleaserConfig.Release.GitHub.Owner
		goreleaserRepo := goreleaserConfig.Release.GitHub.Name

		// Check that workflow validates the same repository
		expectedRepoCheck := "github.repository == '" + goreleaserOwner + "/" + goreleaserRepo + "'"
		if !strings.Contains(workflowStr, expectedRepoCheck) {
			t.Errorf("Workflow should validate repository as '%s/%s'", goreleaserOwner, goreleaserRepo)
		}

		// Check that image name in workflow matches repository structure
		expectedImageName := goreleaserOwner + "/" + goreleaserRepo
		imageNamePattern := regexp.MustCompile(`IMAGE_NAME:\s*(.+)`)
		imageNameMatch := imageNamePattern.FindStringSubmatch(workflowStr)

		if len(imageNameMatch) < 2 {
			t.Fatal("Could not extract IMAGE_NAME from workflow")
		}

		workflowImageName := strings.TrimSpace(imageNameMatch[1])
		if workflowImageName != expectedImageName {
			t.Errorf("Workflow IMAGE_NAME (%s) should match GoReleaser repository (%s)", workflowImageName, expectedImageName)
		}
	})
}

// TestGoReleaserConflictPrevention tests that the workflow prevents conflicts with GoReleaser
func TestGoReleaserConflictPrevention(t *testing.T) {
	t.Run("NoConflictingPublishing", func(t *testing.T) {
		// Read GitHub Actions workflow
		workflowPath := filepath.Join("..", "..", ".github", "workflows", "docker-publish.yml")
		workflowContent, err := os.ReadFile(workflowPath)
		if err != nil {
			t.Fatalf("Failed to read workflow file: %v", err)
		}

		workflowStr := string(workflowContent)

		// The workflow should have logic to prevent publishing when GoReleaser will handle it
		// This could be through conditional publishing or coordination logic

		// Check that workflow has conditional publishing logic
		if !strings.Contains(workflowStr, "should_publish") {
			t.Error("Workflow should have conditional publishing logic to coordinate with GoReleaser")
		}

		// Check that workflow considers event type for publishing decisions
		if !strings.Contains(workflowStr, "github.event_name") {
			t.Error("Workflow should consider event type for publishing coordination")
		}

		// Workflow should handle both tag and release events appropriately
		if !strings.Contains(workflowStr, "github.ref_type") {
			t.Error("Workflow should check ref_type to coordinate tag vs release publishing")
		}
	})

	t.Run("CoordinatedTagging", func(t *testing.T) {
		// Read GitHub Actions workflow
		workflowPath := filepath.Join("..", "..", ".github", "workflows", "docker-publish.yml")
		workflowContent, err := os.ReadFile(workflowPath)
		if err != nil {
			t.Fatalf("Failed to read workflow file: %v", err)
		}

		workflowStr := string(workflowContent)

		// Both GoReleaser and GitHub Actions should be able to create the same tags
		// without conflicts, but they should coordinate when both might run

		// Check that workflow handles tag-based publishing
		if !strings.Contains(workflowStr, "type=ref,event=tag") {
			t.Error("Workflow should handle tag-based publishing")
		}

		// Check that workflow handles release-based publishing
		hasReleaseEvent := strings.Contains(workflowStr, "release:")
		hasPublishedType := strings.Contains(workflowStr, "types: [published]")
		
		if !hasReleaseEvent || !hasPublishedType {
			t.Error("Workflow should handle release-based publishing with 'release:' and 'types: [published]'")
		}

		// The workflow should be idempotent - publishing the same tag multiple times should be safe
		// This is handled by the registry itself, but the workflow should not fail if tags exist
	})
}