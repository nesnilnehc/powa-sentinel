package github_actions

import (
	"regexp"
	"strings"
	"testing"
	"testing/quick"
)

// BuildMetadata represents the build metadata extracted from the workflow
type BuildMetadata struct {
	VersionArg   string
	CommitArg    string
	BuildDateArg string
	Labels       map[string]string
}

// GitContext represents different Git contexts for testing
type GitContext struct {
	RefType string // "tag", "branch"
	RefName string // "v1.0.0", "main", "feature-branch"
	SHA     string // commit SHA
}

// extractBuildMetadata extracts build metadata configuration from workflow
func extractBuildMetadata(config *WorkflowConfig) (*BuildMetadata, error) {
	metadata := &BuildMetadata{
		Labels: make(map[string]string),
	}

	for _, job := range config.Jobs {
		for _, step := range job.Steps {
			// Extract build arguments from docker/build-push-action
			if strings.Contains(step.Uses, "docker/build-push-action") {
				if buildArgsVal, ok := step.With["build-args"]; ok {
					if buildArgsStr, ok := buildArgsVal.(string); ok {
						lines := strings.Split(buildArgsStr, "\n")
						for _, line := range lines {
							line = strings.TrimSpace(line)
							if strings.HasPrefix(line, "VERSION=") {
								metadata.VersionArg = strings.TrimPrefix(line, "VERSION=")
							} else if strings.HasPrefix(line, "COMMIT=") {
								metadata.CommitArg = strings.TrimPrefix(line, "COMMIT=")
							} else if strings.HasPrefix(line, "BUILD_DATE=") {
								metadata.BuildDateArg = strings.TrimPrefix(line, "BUILD_DATE=")
							}
						}
					}
				}
			}

			// Extract labels from docker/metadata-action
			if strings.Contains(step.Uses, "docker/metadata-action") {
				if labelsVal, ok := step.With["labels"]; ok {
					if labelsStr, ok := labelsVal.(string); ok {
						lines := strings.Split(labelsStr, "\n")
						for _, line := range lines {
							line = strings.TrimSpace(line)
							if strings.Contains(line, "=") {
								parts := strings.SplitN(line, "=", 2)
								if len(parts) == 2 {
									metadata.Labels[parts[0]] = parts[1]
								}
							}
						}
					}
				}
			}
		}
	}

	return metadata, nil
}

// validateVersionConsistency checks if version is consistently derived from Git context
func validateVersionConsistency(gitCtx GitContext, metadata *BuildMetadata) bool {
	// Check if version argument references the correct Git context variables
	versionArg := metadata.VersionArg

	// For tagged releases, version should reference the tag name
	if gitCtx.RefType == "tag" {
		return strings.Contains(versionArg, "build-meta.outputs.version") ||
			strings.Contains(versionArg, "github.ref_name")
	}

	// For main branch, version should reference commit SHA
	if gitCtx.RefName == "main" {
		return strings.Contains(versionArg, "build-meta.outputs.version") ||
			strings.Contains(versionArg, "github.sha")
	}

	// For other branches, version should include branch name or SHA
	return strings.Contains(versionArg, "build-meta.outputs.version") ||
		strings.Contains(versionArg, "github.ref_name") ||
		strings.Contains(versionArg, "github.sha")
}

// validateCommitConsistency checks if commit is consistently set to SHA
func validateCommitConsistency(metadata *BuildMetadata) bool {
	commitArg := metadata.CommitArg
	return strings.Contains(commitArg, "build-meta.outputs.commit") ||
		strings.Contains(commitArg, "github.sha")
}

// validateBuildDateConsistency checks if build date is properly set
func validateBuildDateConsistency(metadata *BuildMetadata) bool {
	buildDateArg := metadata.BuildDateArg
	return strings.Contains(buildDateArg, "build-meta.outputs.build_date") ||
		strings.Contains(buildDateArg, "github.run_started_at")
}

// validateLabelConsistency checks if OCI labels contain proper metadata
func validateLabelConsistency(metadata *BuildMetadata) bool {
	requiredLabels := []string{
		"org.opencontainers.image.title",
		"org.opencontainers.image.description",
		"org.opencontainers.image.vendor",
		"org.opencontainers.image.source",
		"org.opencontainers.image.version",
		"org.opencontainers.image.revision",
		"org.opencontainers.image.created",
	}

	for _, label := range requiredLabels {
		if _, exists := metadata.Labels[label]; !exists {
			return false
		}
	}

	// Check that version label references Git context
	versionLabel := metadata.Labels["org.opencontainers.image.version"]
	if !strings.Contains(versionLabel, "github.ref_name") && !strings.Contains(versionLabel, "github.sha") {
		return false
	}

	// Check that revision label references commit SHA
	revisionLabel := metadata.Labels["org.opencontainers.image.revision"]
	if !strings.Contains(revisionLabel, "github.sha") {
		return false
	}

	// Check that created label references build time
	createdLabel := metadata.Labels["org.opencontainers.image.created"]
	if !strings.Contains(createdLabel, "github.run_started_at") {
		return false
	}

	return true
}

// TestBuildMetadataConsistency tests Property 5: Build Metadata Consistency
// **Feature: github-actions-docker-publish, Property 5: Build Metadata Consistency**
// **Validates: Requirements 3.1, 3.2, 3.3, 3.4**
//
// Property: For any Docker image build, the build arguments should include correct 
// version information (VERSION, COMMIT, BUILD_DATE) derived from the Git context
func TestBuildMetadataConsistency(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	metadata, err := extractBuildMetadata(config)
	if err != nil {
		t.Fatalf("Failed to extract build metadata: %v", err)
	}

	// Property-based test: For any Git context, metadata should be consistently derived
	property := func(refType, refName, sha string) bool {
		// Sanitize inputs to simulate realistic Git contexts
		if refType == "" {
			refType = "branch"
		}
		if refName == "" {
			refName = "main"
		}
		if sha == "" {
			sha = "abc123def456"
		}

		// Ensure SHA looks like a commit hash
		shaRegex := regexp.MustCompile(`^[a-f0-9]{7,40}$`)
		if !shaRegex.MatchString(sha) {
			sha = "abc123def456789"
		}

		gitCtx := GitContext{
			RefType: refType,
			RefName: refName,
			SHA:     sha,
		}

		// Check all consistency requirements
		versionOK := validateVersionConsistency(gitCtx, metadata)
		commitOK := validateCommitConsistency(metadata)
		buildDateOK := validateBuildDateConsistency(metadata)
		labelsOK := validateLabelConsistency(metadata)

		return versionOK && commitOK && buildDateOK && labelsOK
	}

	// Run the property test
	if err := quick.Check(property, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Build metadata consistency property failed: %v", err)
		t.Logf("Version arg: %s", metadata.VersionArg)
		t.Logf("Commit arg: %s", metadata.CommitArg)
		t.Logf("Build date arg: %s", metadata.BuildDateArg)
		t.Logf("Labels: %+v", metadata.Labels)
	}
}

// TestBuildArgumentsPresent validates that all required build arguments are configured
func TestBuildArgumentsPresent(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	metadata, err := extractBuildMetadata(config)
	if err != nil {
		t.Fatalf("Failed to extract build metadata: %v", err)
	}

	// Check that all required build arguments are present
	if metadata.VersionArg == "" {
		t.Error("VERSION build argument is missing")
	}
	if metadata.CommitArg == "" {
		t.Error("COMMIT build argument is missing")
	}
	if metadata.BuildDateArg == "" {
		t.Error("BUILD_DATE build argument is missing")
	}

	// Check that build arguments reference appropriate GitHub context variables
	if !strings.Contains(metadata.VersionArg, "github.") && !strings.Contains(metadata.VersionArg, "steps.") {
		t.Errorf("VERSION argument should reference GitHub context: %s", metadata.VersionArg)
	}
	if !strings.Contains(metadata.CommitArg, "github.sha") && !strings.Contains(metadata.CommitArg, "steps.") {
		t.Errorf("COMMIT argument should reference github.sha: %s", metadata.CommitArg)
	}
	if !strings.Contains(metadata.BuildDateArg, "github.") && !strings.Contains(metadata.BuildDateArg, "steps.") {
		t.Errorf("BUILD_DATE argument should reference GitHub context: %s", metadata.BuildDateArg)
	}
}

// TestOCILabelsCompliance validates that OCI labels are properly configured
func TestOCILabelsCompliance(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	metadata, err := extractBuildMetadata(config)
	if err != nil {
		t.Fatalf("Failed to extract build metadata: %v", err)
	}

	// Check required OCI labels
	requiredLabels := map[string]string{
		"org.opencontainers.image.title":       "powa-sentinel",
		"org.opencontainers.image.description": "PostgreSQL Workload Analyzer Sentinel",
		"org.opencontainers.image.vendor":      "powa-team",
		"org.opencontainers.image.source":      "https://github.com/powa-team/powa-sentinel",
	}

	for label, expectedValue := range requiredLabels {
		if actualValue, exists := metadata.Labels[label]; !exists {
			t.Errorf("Required OCI label %s is missing", label)
		} else if actualValue != expectedValue {
			t.Errorf("OCI label %s has incorrect value: expected %s, got %s", label, expectedValue, actualValue)
		}
	}

	// Check dynamic OCI labels
	dynamicLabels := []string{
		"org.opencontainers.image.version",
		"org.opencontainers.image.revision",
		"org.opencontainers.image.created",
	}

	for _, label := range dynamicLabels {
		if _, exists := metadata.Labels[label]; !exists {
			t.Errorf("Dynamic OCI label %s is missing", label)
		}
	}
}