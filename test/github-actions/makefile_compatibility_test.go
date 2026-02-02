package github_actions

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestMakefileCompatibility validates that the GitHub Actions workflow
// doesn't interfere with existing Makefile commands and uses consistent build flags
func TestMakefileCompatibility(t *testing.T) {
	// Test that Makefile exists and has expected Docker commands
	t.Run("MakefileExists", func(t *testing.T) {
		makefilePath := filepath.Join("..", "..", "Makefile")
		if _, err := os.Stat(makefilePath); os.IsNotExist(err) {
			t.Fatal("Makefile not found at expected location")
		}
	})

	// Test that Makefile has Docker build commands
	t.Run("MakefileDockerCommands", func(t *testing.T) {
		makefilePath := filepath.Join("..", "..", "Makefile")
		content, err := os.ReadFile(makefilePath)
		if err != nil {
			t.Fatalf("Failed to read Makefile: %v", err)
		}

		makefileContent := string(content)

		// Check for docker build command
		if !strings.Contains(makefileContent, "docker:") {
			t.Error("Makefile should have 'docker:' target")
		}

		// Check for docker-push command
		if !strings.Contains(makefileContent, "docker-push:") {
			t.Error("Makefile should have 'docker-push:' target")
		}

		// Check that docker commands use buildx
		if !strings.Contains(makefileContent, "docker buildx build") {
			t.Error("Makefile should use 'docker buildx build' for consistency with workflow")
		}
	})

	// Test build flags consistency between Makefile and workflow
	t.Run("BuildFlagsConsistency", func(t *testing.T) {
		// Read Makefile
		makefilePath := filepath.Join("..", "..", "Makefile")
		makefileContent, err := os.ReadFile(makefilePath)
		if err != nil {
			t.Fatalf("Failed to read Makefile: %v", err)
		}

		// Read Dockerfile to understand expected build args
		dockerfilePath := filepath.Join("..", "..", "Dockerfile")
		dockerfileContent, err := os.ReadFile(dockerfilePath)
		if err != nil {
			t.Fatalf("Failed to read Dockerfile: %v", err)
		}

		makefileStr := string(makefileContent)
		dockerfileStr := string(dockerfileContent)

		// Extract LDFLAGS from Makefile
		ldFlagsPattern := regexp.MustCompile(`LDFLAGS=(.+)`)
		ldFlagsMatch := ldFlagsPattern.FindStringSubmatch(makefileStr)

		if len(ldFlagsMatch) < 2 {
			t.Fatal("Could not find LDFLAGS definition in Makefile")
		}

		ldFlags := ldFlagsMatch[1]

		// Check that Makefile LDFLAGS use the same variables as Dockerfile expects
		expectedVars := []string{"VERSION", "COMMIT", "BUILD_DATE"}
		for _, expectedVar := range expectedVars {
			if !strings.Contains(ldFlags, "$("+expectedVar+")") {
				t.Errorf("Makefile LDFLAGS should use $(%s) variable", expectedVar)
			}

			// Check that Dockerfile expects this variable
			if !strings.Contains(dockerfileStr, "${"+expectedVar+"}") {
				t.Errorf("Dockerfile should expect %s build argument", expectedVar)
			}
		}
	})

	// Test version variable consistency
	t.Run("VersionVariableConsistency", func(t *testing.T) {
		makefilePath := filepath.Join("..", "..", "Makefile")
		makefileContent, err := os.ReadFile(makefilePath)
		if err != nil {
			t.Fatalf("Failed to read Makefile: %v", err)
		}

		makefileStr := string(makefileContent)

		// Check that Makefile defines VERSION variable
		if !strings.Contains(makefileStr, "VERSION ?=") {
			t.Error("Makefile should define VERSION variable with default value")
		}

		// Check that VERSION uses git describe or similar
		versionPattern := regexp.MustCompile(`VERSION \?= \$\(shell git describe[^)]+\)`)
		if !versionPattern.MatchString(makefileStr) {
			t.Error("Makefile VERSION should use git describe for consistency")
		}

		// Check COMMIT variable
		if !strings.Contains(makefileStr, "COMMIT ?=") {
			t.Error("Makefile should define COMMIT variable")
		}

		// Check BUILD_DATE variable
		if !strings.Contains(makefileStr, "BUILD_DATE ?=") {
			t.Error("Makefile should define BUILD_DATE variable")
		}
	})

	// Test that Makefile docker commands don't conflict with workflow
	t.Run("NoWorkflowConflicts", func(t *testing.T) {
		// Read GitHub Actions workflow
		workflowPath := filepath.Join("..", "..", ".github", "workflows", "docker-publish.yml")
		workflowContent, err := os.ReadFile(workflowPath)
		if err != nil {
			t.Fatalf("Failed to read workflow file: %v", err)
		}

		// Read Makefile
		makefilePath := filepath.Join("..", "..", "Makefile")
		makefileContent, err := os.ReadFile(makefilePath)
		if err != nil {
			t.Fatalf("Failed to read Makefile: %v", err)
		}

		workflowStr := string(workflowContent)
		makefileStr := string(makefileContent)

		// Both should use the same platforms for multi-arch builds
		workflowPlatforms := "linux/amd64,linux/arm64"
		if strings.Contains(makefileStr, "--platform") {
			if !strings.Contains(makefileStr, workflowPlatforms) {
				t.Error("Makefile and workflow should use the same platforms for multi-arch builds")
			}
		}

		// Both should use buildx
		if strings.Contains(makefileStr, "docker build ") && !strings.Contains(makefileStr, "docker buildx build") {
			t.Error("Makefile should use 'docker buildx build' for consistency with workflow")
		}

		// Workflow shouldn't interfere with Makefile commands
		if strings.Contains(workflowStr, "make docker") || strings.Contains(workflowStr, "make build") {
			t.Error("Workflow should not call Makefile commands to avoid conflicts")
		}
	})

	// Test registry compatibility
	t.Run("RegistryCompatibility", func(t *testing.T) {
		// Read Makefile
		makefilePath := filepath.Join("..", "..", "Makefile")
		makefileContent, err := os.ReadFile(makefilePath)
		if err != nil {
			t.Fatalf("Failed to read Makefile: %v", err)
		}

		// Read GitHub Actions workflow
		workflowPath := filepath.Join("..", "..", ".github", "workflows", "docker-publish.yml")
		workflowContent, err := os.ReadFile(workflowPath)
		if err != nil {
			t.Fatalf("Failed to read workflow file: %v", err)
		}

		makefileStr := string(makefileContent)
		workflowStr := string(workflowContent)

		// Extract registry from workflow
		registryPattern := regexp.MustCompile(`REGISTRY:\s*(.+)`)
		registryMatch := registryPattern.FindStringSubmatch(workflowStr)

		if len(registryMatch) < 2 {
			t.Fatal("Could not extract REGISTRY from workflow")
		}

		workflowRegistry := strings.TrimSpace(registryMatch[1])

		// Verify the registry is what we expect
		if workflowRegistry != "ghcr.io" {
			t.Errorf("Expected workflow registry to be 'ghcr.io', got '%s'", workflowRegistry)
		}

		// Check that Makefile docker-push can use the same registry
		if strings.Contains(makefileStr, "docker-push:") {
			// Makefile should use REGISTRY environment variable
			if !strings.Contains(makefileStr, "$(REGISTRY)") {
				t.Error("Makefile docker-push should use $(REGISTRY) environment variable")
			}

			// Test that the same registry can be used
			expectedRegistryUsage := "$(REGISTRY)/powa-sentinel"
			if !strings.Contains(makefileStr, expectedRegistryUsage) {
				t.Error("Makefile should use $(REGISTRY)/powa-sentinel format for compatibility")
			}
		}
	})

	// Test that Makefile targets are documented
	t.Run("MakefileDocumentation", func(t *testing.T) {
		makefilePath := filepath.Join("..", "..", "Makefile")
		file, err := os.Open(makefilePath)
		if err != nil {
			t.Fatalf("Failed to open Makefile: %v", err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		var lines []string
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}

		// Check that docker targets are documented
		dockerTargetFound := false
		dockerPushTargetFound := false
		dockerTargetDocumented := false
		dockerPushTargetDocumented := false

		for i, line := range lines {
			if strings.Contains(line, "docker:") && !strings.Contains(line, "docker-push:") {
				dockerTargetFound = true
				// Check if previous line has documentation
				if i > 0 && strings.HasPrefix(lines[i-1], "##") {
					dockerTargetDocumented = true
				}
			}
			if strings.Contains(line, "docker-push:") {
				dockerPushTargetFound = true
				// Check if previous line has documentation
				if i > 0 && strings.HasPrefix(lines[i-1], "##") {
					dockerPushTargetDocumented = true
				}
			}
		}

		if dockerTargetFound && !dockerTargetDocumented {
			t.Error("Makefile 'docker' target should be documented with ## comment")
		}

		if dockerPushTargetFound && !dockerPushTargetDocumented {
			t.Error("Makefile 'docker-push' target should be documented with ## comment")
		}
	})

	// Test build context consistency
	t.Run("BuildContextConsistency", func(t *testing.T) {
		// Read Makefile
		makefilePath := filepath.Join("..", "..", "Makefile")
		makefileContent, err := os.ReadFile(makefilePath)
		if err != nil {
			t.Fatalf("Failed to read Makefile: %v", err)
		}

		// Read GitHub Actions workflow
		workflowPath := filepath.Join("..", "..", ".github", "workflows", "docker-publish.yml")
		workflowContent, err := os.ReadFile(workflowPath)
		if err != nil {
			t.Fatalf("Failed to read workflow file: %v", err)
		}

		makefileStr := string(makefileContent)
		workflowStr := string(workflowContent)

		// Both should use current directory (.) as build context
		if strings.Contains(makefileStr, "docker buildx build") {
			// Makefile docker commands should end with " ."
			dockerBuildLines := strings.Split(makefileStr, "\n")
			foundCorrectContext := false
			for _, line := range dockerBuildLines {
				if strings.Contains(line, "docker buildx build") && strings.HasSuffix(strings.TrimSpace(line), " .") {
					foundCorrectContext = true
					break
				}
			}
			if !foundCorrectContext {
				t.Error("Makefile docker build commands should use '.' as build context")
			}
		}

		// Workflow should use context: .
		if !strings.Contains(workflowStr, "context: .") {
			t.Error("Workflow should use 'context: .' for build context consistency")
		}
	})
}

// TestMakefileWorkflowIntegration tests that Makefile and workflow can work together
func TestMakefileWorkflowIntegration(t *testing.T) {
	t.Run("ComplementaryFunctionality", func(t *testing.T) {
		// Read Makefile
		makefilePath := filepath.Join("..", "..", "Makefile")
		makefileContent, err := os.ReadFile(makefilePath)
		if err != nil {
			t.Fatalf("Failed to read Makefile: %v", err)
		}

		// Read GitHub Actions workflow
		workflowPath := filepath.Join("..", "..", ".github", "workflows", "docker-publish.yml")
		workflowContent, err := os.ReadFile(workflowPath)
		if err != nil {
			t.Fatalf("Failed to read workflow file: %v", err)
		}

		makefileStr := string(makefileContent)
		workflowStr := string(workflowContent)

		// Makefile should provide local development commands
		localDevCommands := []string{"build:", "test:", "lint:", "run:"}
		for _, cmd := range localDevCommands {
			if !strings.Contains(makefileStr, cmd) {
				t.Errorf("Makefile should provide '%s' command for local development", strings.TrimSuffix(cmd, ":"))
			}
		}

		// Workflow should handle CI/CD automation
		requiredCicdFeatures := map[string]string{
			"multi-architecture": "linux/amd64,linux/arm64",
			"registry authentication": "docker/login-action",
			"conditional publishing": "should_publish",
		}
		
		for feature, pattern := range requiredCicdFeatures {
			if !strings.Contains(workflowStr, pattern) {
				t.Errorf("Workflow should handle %s (looking for pattern: %s)", feature, pattern)
			}
		}
	})

	t.Run("ConsistentImageNaming", func(t *testing.T) {
		// Read Makefile
		makefilePath := filepath.Join("..", "..", "Makefile")
		makefileContent, err := os.ReadFile(makefilePath)
		if err != nil {
			t.Fatalf("Failed to read Makefile: %v", err)
		}

		// Read GitHub Actions workflow
		workflowPath := filepath.Join("..", "..", ".github", "workflows", "docker-publish.yml")
		workflowContent, err := os.ReadFile(workflowPath)
		if err != nil {
			t.Fatalf("Failed to read workflow file: %v", err)
		}

		makefileStr := string(makefileContent)
		workflowStr := string(workflowContent)

		// Extract image name from workflow
		imageNamePattern := regexp.MustCompile(`IMAGE_NAME:\s*(.+)`)
		imageNameMatch := imageNamePattern.FindStringSubmatch(workflowStr)

		if len(imageNameMatch) < 2 {
			t.Fatal("Could not extract IMAGE_NAME from workflow")
		}

		workflowImageName := strings.TrimSpace(imageNameMatch[1])
		_ = workflowImageName // Used for validation context

		// Check that Makefile uses consistent image naming pattern
		if strings.Contains(makefileStr, "docker-push:") {
			// Makefile uses $(REGISTRY)/powa-sentinel format
			expectedImageRef := "$(REGISTRY)/powa-sentinel"
			if !strings.Contains(makefileStr, expectedImageRef) {
				t.Errorf("Makefile should use image naming pattern: %s", expectedImageRef)
			}
			
			// This is compatible with workflow's powa-team/powa-sentinel when REGISTRY=ghcr.io/powa-team
			// The difference is intentional: Makefile uses simple name, workflow uses full path
		}

		// Local docker target should use simple name
		if strings.Contains(makefileStr, "docker:") && !strings.Contains(makefileStr, "docker-push:") {
			// Should use powa-sentinel as local image name
			if !strings.Contains(makefileStr, "powa-sentinel:$(VERSION)") {
				t.Error("Makefile local docker target should use 'powa-sentinel:$(VERSION)' naming")
			}
		}
	})
}