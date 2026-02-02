package github_actions

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestDockerfileIntegration validates that the GitHub Actions workflow
// uses the existing Dockerfile without modifications and with correct build arguments
func TestDockerfileIntegration(t *testing.T) {
	// Test that Dockerfile exists and has expected structure
	t.Run("DockerfileExists", func(t *testing.T) {
		dockerfilePath := filepath.Join("..", "..", "Dockerfile")
		if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
			t.Fatal("Dockerfile not found at expected location")
		}
	})

	// Test that Dockerfile declares expected build arguments
	t.Run("DockerfileBuildArgs", func(t *testing.T) {
		dockerfilePath := filepath.Join("..", "..", "Dockerfile")
		content, err := os.ReadFile(dockerfilePath)
		if err != nil {
			t.Fatalf("Failed to read Dockerfile: %v", err)
		}

		dockerfileContent := string(content)
		
		// Check for required ARG declarations
		expectedArgs := []string{"VERSION", "COMMIT", "BUILD_DATE"}
		for _, arg := range expectedArgs {
			argPattern := regexp.MustCompile(`ARG\s+` + arg + `\s*=`)
			if !argPattern.MatchString(dockerfileContent) {
				t.Errorf("Dockerfile missing ARG declaration for %s", arg)
			}
		}
	})

	// Test that Dockerfile uses build arguments in ldflags
	t.Run("DockerfileLdflags", func(t *testing.T) {
		dockerfilePath := filepath.Join("..", "..", "Dockerfile")
		content, err := os.ReadFile(dockerfilePath)
		if err != nil {
			t.Fatalf("Failed to read Dockerfile: %v", err)
		}

		dockerfileContent := string(content)
		
		// Check that ldflags uses the build arguments
		ldFlagsPattern := regexp.MustCompile(`-ldflags.*-X main\.version=\$\{VERSION\}.*-X main\.commit=\$\{COMMIT\}.*-X main\.buildDate=\$\{BUILD_DATE\}`)
		if !ldFlagsPattern.MatchString(dockerfileContent) {
			t.Error("Dockerfile ldflags do not properly use VERSION, COMMIT, and BUILD_DATE build arguments")
		}
	})

	// Test that workflow uses correct build arguments
	t.Run("WorkflowBuildArgs", func(t *testing.T) {
		workflowPath := filepath.Join("..", "..", ".github", "workflows", "docker-publish.yml")
		content, err := os.ReadFile(workflowPath)
		if err != nil {
			t.Fatalf("Failed to read workflow file: %v", err)
		}

		var workflow map[string]interface{}
		if err := yaml.Unmarshal(content, &workflow); err != nil {
			t.Fatalf("Failed to parse workflow YAML: %v", err)
		}

		// Extract build-args from docker build steps
		jobs, ok := workflow["jobs"].(map[string]interface{})
		if !ok {
			t.Fatal("Workflow missing jobs section")
		}

		dockerJob, ok := jobs["docker"].(map[string]interface{})
		if !ok {
			t.Fatal("Workflow missing docker job")
		}

		steps, ok := dockerJob["steps"].([]interface{})
		if !ok {
			t.Fatal("Docker job missing steps")
		}

		// Find docker build steps and validate build-args
		buildArgsFound := false
		for _, step := range steps {
			stepMap, ok := step.(map[string]interface{})
			if !ok {
				continue
			}

			// Check if this is a docker build step
			uses, hasUses := stepMap["uses"].(string)
			if !hasUses || !strings.Contains(uses, "docker/build-push-action") {
				continue
			}

			with, hasWith := stepMap["with"].(map[string]interface{})
			if !hasWith {
				continue
			}

			buildArgs, hasBuildArgs := with["build-args"].(string)
			if !hasBuildArgs {
				continue
			}

			buildArgsFound = true

			// Validate that build-args contain expected arguments
			expectedBuildArgs := []string{"VERSION=", "COMMIT=", "BUILD_DATE="}
			for _, expectedArg := range expectedBuildArgs {
				if !strings.Contains(buildArgs, expectedArg) {
					t.Errorf("Workflow build-args missing %s", expectedArg)
				}
			}
		}

		if !buildArgsFound {
			t.Error("No docker build steps with build-args found in workflow")
		}
	})

	// Test that workflow doesn't modify Dockerfile
	t.Run("WorkflowUsesExistingDockerfile", func(t *testing.T) {
		workflowPath := filepath.Join("..", "..", ".github", "workflows", "docker-publish.yml")
		content, err := os.ReadFile(workflowPath)
		if err != nil {
			t.Fatalf("Failed to read workflow file: %v", err)
		}

		workflowContent := string(content)

		// Check that workflow doesn't create or modify Dockerfile by looking for specific patterns
		dockerfileModificationPatterns := []string{
			"echo.*>.*Dockerfile",
			"cat.*>.*Dockerfile", 
			"tee.*Dockerfile",
			"sed.*Dockerfile",
		}
		
		for _, pattern := range dockerfileModificationPatterns {
			matched, _ := regexp.MatchString(pattern, workflowContent)
			if matched {
				t.Errorf("Workflow appears to modify Dockerfile with pattern: %s", pattern)
			}
		}

		// Check that workflow uses context: . (current directory with existing Dockerfile)
		contextPattern := regexp.MustCompile(`context:\s*\.`)
		if !contextPattern.MatchString(workflowContent) {
			t.Error("Workflow should use 'context: .' to use existing Dockerfile")
		}

		// Verify workflow doesn't specify a custom dockerfile path (uses default)
		customDockerfilePattern := regexp.MustCompile(`dockerfile:\s*\S+`)
		if customDockerfilePattern.MatchString(workflowContent) {
			t.Error("Workflow should use default Dockerfile, not specify custom dockerfile path")
		}
	})

	// Test build argument consistency between Dockerfile and Makefile
	t.Run("BuildArgConsistency", func(t *testing.T) {
		// Read Dockerfile
		dockerfilePath := filepath.Join("..", "..", "Dockerfile")
		dockerfileContent, err := os.ReadFile(dockerfilePath)
		if err != nil {
			t.Fatalf("Failed to read Dockerfile: %v", err)
		}

		// Read Makefile
		makefilePath := filepath.Join("..", "..", "Makefile")
		makefileContent, err := os.ReadFile(makefilePath)
		if err != nil {
			t.Fatalf("Failed to read Makefile: %v", err)
		}

		dockerfileStr := string(dockerfileContent)
		makefileStr := string(makefileContent)

		// Both should use the same ldflags pattern for version info
		// Extract variable names from both files
		dockerfileVars := extractLdflagsVars(dockerfileStr)
		makefileVars := extractLdflagsVars(makefileStr)

		if len(dockerfileVars) == 0 {
			t.Error("No ldflags variables found in Dockerfile")
		}

		if len(makefileVars) == 0 {
			t.Error("No ldflags variables found in Makefile")
		}

		// Check that both use the same variable names
		for _, dockerVar := range dockerfileVars {
			found := false
			for _, makeVar := range makefileVars {
				if dockerVar == makeVar {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Dockerfile uses ldflags variable %s but Makefile doesn't", dockerVar)
			}
		}
	})
}

// extractLdflagsVars extracts variable names from ldflags in the format -X main.variable=
func extractLdflagsVars(content string) []string {
	var vars []string
	
	// Pattern to match -X main.variable= in ldflags
	pattern := regexp.MustCompile(`-X\s+main\.(\w+)=`)
	matches := pattern.FindAllStringSubmatch(content, -1)
	
	for _, match := range matches {
		if len(match) > 1 {
			vars = append(vars, match[1])
		}
	}
	
	return vars
}

// TestDockerfileBuildProcess validates that the Docker build process works correctly
func TestDockerfileBuildProcess(t *testing.T) {
	t.Run("DockerfileStructure", func(t *testing.T) {
		dockerfilePath := filepath.Join("..", "..", "Dockerfile")
		file, err := os.Open(dockerfilePath)
		if err != nil {
			t.Fatalf("Failed to open Dockerfile: %v", err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		var lines []string
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}

		// Validate multi-stage build structure
		hasBuilderStage := false
		hasRuntimeStage := false
		
		for _, line := range lines {
			if strings.Contains(line, "FROM golang:") && strings.Contains(line, "AS builder") {
				hasBuilderStage = true
			}
			if strings.Contains(line, "FROM gcr.io/distroless/static") {
				hasRuntimeStage = true
			}
		}

		if !hasBuilderStage {
			t.Error("Dockerfile missing builder stage")
		}

		if !hasRuntimeStage {
			t.Error("Dockerfile missing runtime stage")
		}
	})

	t.Run("DockerfileBinaryPath", func(t *testing.T) {
		dockerfilePath := filepath.Join("..", "..", "Dockerfile")
		content, err := os.ReadFile(dockerfilePath)
		if err != nil {
			t.Fatalf("Failed to read Dockerfile: %v", err)
		}

		dockerfileContent := string(content)

		// Check that binary is built to expected location
		if !strings.Contains(dockerfileContent, "-o powa-sentinel ./cmd/powa-sentinel") {
			t.Error("Dockerfile should build binary as 'powa-sentinel' from './cmd/powa-sentinel'")
		}

		// Check that binary is copied to runtime stage
		if !strings.Contains(dockerfileContent, "COPY --from=builder /app/powa-sentinel /powa-sentinel") {
			t.Error("Dockerfile should copy binary from builder stage to runtime stage")
		}
	})
}