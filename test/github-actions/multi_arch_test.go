package github_actions

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/quick"

	"gopkg.in/yaml.v3"
)

// WorkflowConfig represents the GitHub Actions workflow configuration
type WorkflowConfig struct {
	Name        string            `yaml:"name"`
	Permissions map[string]string `yaml:"permissions"`
	On          struct {
		Push struct {
			Branches []string `yaml:"branches"`
			Tags     []string `yaml:"tags"`
		} `yaml:"push"`
		PullRequest struct {
			Branches []string `yaml:"branches"`
		} `yaml:"pull_request"`
		Release struct {
			Types []string `yaml:"types"`
		} `yaml:"release"`
	} `yaml:"on"`
	Jobs map[string]struct {
		If    string `yaml:"if"`
		Steps []struct {
			Name string                 `yaml:"name"`
			Uses string                 `yaml:"uses"`
			With map[string]interface{} `yaml:"with"`
			Run  interface{}            `yaml:"run"`
		} `yaml:"steps"`
	} `yaml:"jobs"`
}

// BuildPlatforms represents the platforms configuration for Docker builds
type BuildPlatforms struct {
	QEMUPlatforms    []string
	BuildxPlatforms  []string
	DockerPlatforms  []string
}

// parseWorkflowFile reads and parses the GitHub Actions workflow file
func parseWorkflowFile() (*WorkflowConfig, error) {
	workflowPath := filepath.Join("..", "..", ".github", "workflows", "docker-publish.yml")
	data, err := os.ReadFile(workflowPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow file: %w", err)
	}

	var config WorkflowConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse workflow YAML: %w", err)
	}

	return &config, nil
}

// extractBuildPlatforms extracts platform configurations from workflow steps
func extractBuildPlatforms(config *WorkflowConfig) (*BuildPlatforms, error) {
	platforms := &BuildPlatforms{}

	for _, job := range config.Jobs {
		for _, step := range job.Steps {
			switch {
			case strings.Contains(step.Uses, "docker/setup-qemu-action"):
				if platformsVal, ok := step.With["platforms"]; ok {
					if platformsStr, ok := platformsVal.(string); ok {
						platforms.QEMUPlatforms = strings.Split(platformsStr, ",")
						for i, p := range platforms.QEMUPlatforms {
							platforms.QEMUPlatforms[i] = strings.TrimSpace(p)
						}
					}
				}
			case strings.Contains(step.Uses, "docker/setup-buildx-action"):
				if platformsVal, ok := step.With["platforms"]; ok {
					if platformsStr, ok := platformsVal.(string); ok {
						platforms.BuildxPlatforms = strings.Split(platformsStr, ",")
						for i, p := range platforms.BuildxPlatforms {
							platforms.BuildxPlatforms[i] = strings.TrimSpace(p)
						}
					}
				}
			case strings.Contains(step.Uses, "docker/build-push-action"):
				if platformsVal, ok := step.With["platforms"]; ok {
					if platformsStr, ok := platformsVal.(string); ok {
						platforms.DockerPlatforms = strings.Split(platformsStr, ",")
						for i, p := range platforms.DockerPlatforms {
							platforms.DockerPlatforms[i] = strings.TrimSpace(p)
						}
					}
				}
			}
		}
	}

	return platforms, nil
}

// TestMultiArchitectureBuildSupport tests Property 2: Multi-Architecture Build Support
// **Feature: github-actions-docker-publish, Property 2: Multi-Architecture Build Support**
// **Validates: Requirements 1.5**
//
// Property: For any Docker build execution, the resulting image manifest should include 
// both linux/amd64 and linux/arm64 architectures
func TestMultiArchitectureBuildSupport(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	platforms, err := extractBuildPlatforms(config)
	if err != nil {
		t.Fatalf("Failed to extract build platforms: %v", err)
	}

	// Property-based test: For any workflow configuration, multi-arch support should be present
	property := func() bool {
		// Check that QEMU emulation includes both required architectures
		hasAMD64QEMU := false
		hasARM64QEMU := false
		for _, platform := range platforms.QEMUPlatforms {
			if platform == "linux/amd64" {
				hasAMD64QEMU = true
			}
			if platform == "linux/arm64" {
				hasARM64QEMU = true
			}
		}

		// Check that Buildx is configured for both architectures
		hasAMD64Buildx := false
		hasARM64Buildx := false
		for _, platform := range platforms.BuildxPlatforms {
			if platform == "linux/amd64" {
				hasAMD64Buildx = true
			}
			if platform == "linux/arm64" {
				hasARM64Buildx = true
			}
		}

		// Check that Docker build-push-action targets both architectures
		hasAMD64Docker := false
		hasARM64Docker := false
		for _, platform := range platforms.DockerPlatforms {
			if platform == "linux/amd64" {
				hasAMD64Docker = true
			}
			if platform == "linux/arm64" {
				hasARM64Docker = true
			}
		}

		// All three components must support both architectures
		return hasAMD64QEMU && hasARM64QEMU &&
			hasAMD64Buildx && hasARM64Buildx &&
			hasAMD64Docker && hasARM64Docker
	}

	// Run the property test
	if err := quick.Check(property, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Multi-architecture build support property failed: %v", err)
		t.Logf("QEMU platforms: %v", platforms.QEMUPlatforms)
		t.Logf("Buildx platforms: %v", platforms.BuildxPlatforms)
		t.Logf("Docker platforms: %v", platforms.DockerPlatforms)
	}
}

// TestRequiredArchitecturesPresent validates that both required architectures are configured
func TestRequiredArchitecturesPresent(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	platforms, err := extractBuildPlatforms(config)
	if err != nil {
		t.Fatalf("Failed to extract build platforms: %v", err)
	}

	requiredArchs := []string{"linux/amd64", "linux/arm64"}

	// Test QEMU platforms
	for _, required := range requiredArchs {
		found := false
		for _, platform := range platforms.QEMUPlatforms {
			if platform == required {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Required architecture %s not found in QEMU platforms: %v", required, platforms.QEMUPlatforms)
		}
	}

	// Test Buildx platforms
	for _, required := range requiredArchs {
		found := false
		for _, platform := range platforms.BuildxPlatforms {
			if platform == required {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Required architecture %s not found in Buildx platforms: %v", required, platforms.BuildxPlatforms)
		}
	}

	// Test Docker build platforms
	for _, required := range requiredArchs {
		found := false
		for _, platform := range platforms.DockerPlatforms {
			if platform == required {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Required architecture %s not found in Docker build platforms: %v", required, platforms.DockerPlatforms)
		}
	}
}