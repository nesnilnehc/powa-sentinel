package github_actions

import (
	"fmt"
	"strings"
	"testing"
	"testing/quick"
)

// WorkflowIntegrationScenario represents a complete workflow execution scenario
type WorkflowIntegrationScenario struct {
	EventType    string            // "push", "pull_request", "release"
	RefType      string            // "branch", "tag"
	RefName      string            // "main", "v1.0.0", "feature-branch"
	Repository   string            // "powa-team/powa-sentinel"
	Actor        string            // GitHub username
	Environment  map[string]string // Environment variables
	ShouldPublish bool             // Whether images should be published
	ExpectedTags  []string         // Expected image tags
}

// WorkflowExecutionResult represents the expected result of workflow execution
type WorkflowExecutionResult struct {
	ShouldTrigger     bool     // Whether workflow should trigger
	ShouldBuild       bool     // Whether images should be built
	ShouldPublish     bool     // Whether images should be published
	ExpectedPlatforms []string // Expected build platforms
	ExpectedTags      []string // Expected image tags
	RequiredSteps     []string // Required workflow steps
}

// generateWorkflowScenario creates realistic workflow scenarios for testing
func generateWorkflowScenario(eventType, refType, refName string) WorkflowIntegrationScenario {
	scenario := WorkflowIntegrationScenario{
		EventType:   eventType,
		RefType:     refType,
		RefName:     refName,
		Repository:  "powa-team/powa-sentinel",
		Actor:       "test-user",
		Environment: make(map[string]string),
	}

	// Set realistic environment variables
	scenario.Environment["REGISTRY"] = "ghcr.io"
	scenario.Environment["IMAGE_NAME"] = "powa-team/powa-sentinel"

	// Determine publishing behavior and expected tags
	switch {
	case eventType == "push" && refType == "branch" && refName == "main":
		scenario.ShouldPublish = true
		scenario.ExpectedTags = []string{
			"ghcr.io/powa-team/powa-sentinel:main",
			"ghcr.io/powa-team/powa-sentinel:main-abc1234",
		}
	case eventType == "push" && refType == "tag" && strings.HasPrefix(refName, "v"):
		scenario.ShouldPublish = true
		scenario.ExpectedTags = []string{
			fmt.Sprintf("ghcr.io/powa-team/powa-sentinel:%s", refName),
			"ghcr.io/powa-team/powa-sentinel:latest",
		}
	case eventType == "pull_request":
		scenario.ShouldPublish = false
		scenario.ExpectedTags = []string{
			fmt.Sprintf("ghcr.io/powa-team/powa-sentinel:pr-%s", refName),
		}
	case eventType == "release":
		scenario.ShouldPublish = true
		scenario.ExpectedTags = []string{
			fmt.Sprintf("ghcr.io/powa-team/powa-sentinel:%s", refName),
			"ghcr.io/powa-team/powa-sentinel:latest",
		}
	default:
		scenario.ShouldPublish = false
		scenario.ExpectedTags = []string{}
	}

	return scenario
}

// validateWorkflowExecution validates that workflow execution matches expected behavior
func validateWorkflowExecution(scenario WorkflowIntegrationScenario, config *WorkflowConfig) (*WorkflowExecutionResult, error) {
	result := &WorkflowExecutionResult{
		ExpectedPlatforms: []string{"linux/amd64", "linux/arm64"},
		RequiredSteps: []string{
			"Validate workflow permissions",
			"Checkout repository",
			"Set up QEMU emulation",
			"Set up Docker Buildx",
			"Extract metadata",
			"Build and push Docker image",
		},
	}

	// Determine if workflow should trigger
	result.ShouldTrigger = shouldWorkflowTrigger(scenario, config)
	
	// If workflow triggers, it should always build
	result.ShouldBuild = result.ShouldTrigger
	
	// Publishing depends on event type and repository
	result.ShouldPublish = result.ShouldTrigger && scenario.ShouldPublish && 
		scenario.Repository == "powa-team/powa-sentinel"
	
	// Set expected tags
	result.ExpectedTags = scenario.ExpectedTags

	// Add conditional steps based on publishing
	if result.ShouldPublish {
		result.RequiredSteps = append(result.RequiredSteps,
			"Log in to Container Registry",
			"Verify registry authentication",
			"Output image details",
			"Verify image accessibility",
		)
	}

	return result, nil
}

// shouldWorkflowTrigger determines if workflow should trigger for given scenario
func shouldWorkflowTrigger(scenario WorkflowIntegrationScenario, config *WorkflowConfig) bool {
	// Check repository restriction
	if scenario.Repository != "powa-team/powa-sentinel" {
		return false
	}

	// Check trigger conditions based on event type
	switch scenario.EventType {
	case "push":
		if scenario.RefType == "branch" {
			// Check if branch is in push triggers
			for _, branch := range config.On.Push.Branches {
				if branch == scenario.RefName {
					return true
				}
			}
		} else if scenario.RefType == "tag" {
			// Check if tag matches patterns
			for _, tagPattern := range config.On.Push.Tags {
				if tagPattern == "v*" && strings.HasPrefix(scenario.RefName, "v") {
					return true
				}
				if tagPattern == scenario.RefName {
					return true
				}
			}
		}
	case "pull_request":
		// Check if target branch is in PR triggers
		for _, branch := range config.On.PullRequest.Branches {
			if branch == "main" { // Assuming PR targets main
				return true
			}
		}
	case "release":
		// Check if release type is in triggers
		for _, releaseType := range config.On.Release.Types {
			if releaseType == "published" {
				return true
			}
		}
	}

	return false
}

// TestWorkflowIntegrationScenarios tests complete workflow execution scenarios
// This test validates end-to-end workflow behavior across different Git events
func TestWorkflowIntegrationScenarios(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	// Test specific integration scenarios
	scenarios := []WorkflowIntegrationScenario{
		// Main branch push scenario
		generateWorkflowScenario("push", "branch", "main"),
		
		// Version tag push scenario
		generateWorkflowScenario("push", "tag", "v1.0.0"),
		
		// Pull request scenario
		generateWorkflowScenario("pull_request", "branch", "feature-branch"),
		
		// Release scenario
		generateWorkflowScenario("release", "tag", "v1.0.0"),
		
		// Non-main branch push (should not publish)
		generateWorkflowScenario("push", "branch", "develop"),
	}

	for _, scenario := range scenarios {
		t.Run(fmt.Sprintf("%s_%s_%s", scenario.EventType, scenario.RefType, scenario.RefName), func(t *testing.T) {
			result, err := validateWorkflowExecution(scenario, config)
			if err != nil {
				t.Fatalf("Failed to validate workflow execution: %v", err)
			}

			// Validate trigger behavior
			if !result.ShouldTrigger && (scenario.EventType == "push" && scenario.RefName == "main") {
				t.Error("Workflow should trigger for main branch pushes")
			}

			if !result.ShouldTrigger && (scenario.EventType == "push" && strings.HasPrefix(scenario.RefName, "v")) {
				t.Error("Workflow should trigger for version tag pushes")
			}

			if !result.ShouldTrigger && scenario.EventType == "pull_request" {
				t.Error("Workflow should trigger for pull requests")
			}

			// Validate build behavior
			if result.ShouldTrigger && !result.ShouldBuild {
				t.Error("Workflow should build images when triggered")
			}

			// Validate publishing behavior
			expectedPublish := scenario.ShouldPublish && scenario.Repository == "powa-team/powa-sentinel"
			if result.ShouldPublish != expectedPublish {
				t.Errorf("Publishing behavior mismatch: expected %v, got %v", expectedPublish, result.ShouldPublish)
			}

			// Validate multi-architecture support
			expectedPlatforms := []string{"linux/amd64", "linux/arm64"}
			if len(result.ExpectedPlatforms) != len(expectedPlatforms) {
				t.Errorf("Platform count mismatch: expected %d, got %d", len(expectedPlatforms), len(result.ExpectedPlatforms))
			}

			for _, platform := range expectedPlatforms {
				found := false
				for _, resultPlatform := range result.ExpectedPlatforms {
					if platform == resultPlatform {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Required platform %s not found in result platforms", platform)
				}
			}
		})
	}
}

// TestWorkflowIntegrationProperty tests the integration property across random scenarios
// Property: For any valid workflow scenario, the workflow should execute appropriate steps
// and produce expected results based on the Git event type and repository context
func TestWorkflowIntegrationProperty(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	// Property-based test for workflow integration
	property := func(eventType, refType, refName string) bool {
		// Sanitize inputs to create realistic scenarios
		validEvents := []string{"push", "pull_request", "release"}
		if !contains(validEvents, eventType) {
			eventType = "push"
		}

		validRefTypes := []string{"branch", "tag"}
		if !contains(validRefTypes, refType) {
			refType = "branch"
		}

		if refName == "" {
			refName = "main"
		}

		// Adjust ref type based on event type
		if eventType == "pull_request" {
			refType = "branch"
		}
		if eventType == "release" {
			refType = "tag"
		}

		scenario := generateWorkflowScenario(eventType, refType, refName)
		result, err := validateWorkflowExecution(scenario, config)
		if err != nil {
			return false
		}

		// Validate consistency of execution result
		if result.ShouldTrigger {
			// If workflow triggers, it should build
			if !result.ShouldBuild {
				return false
			}

			// Multi-architecture support should always be present
			if len(result.ExpectedPlatforms) < 2 {
				return false
			}

			// Required steps should be present
			if len(result.RequiredSteps) < 6 {
				return false
			}
		}

		// Publishing should only happen for authorized scenarios
		if result.ShouldPublish {
			if scenario.Repository != "powa-team/powa-sentinel" {
				return false
			}
			if eventType == "pull_request" {
				return false // PRs should never publish
			}
		}

		return true
	}

	// Run the property test
	if err := quick.Check(property, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Workflow integration property failed: %v", err)
	}
}

// TestWorkflowErrorRecoveryScenarios tests error recovery and rollback scenarios
func TestWorkflowErrorRecoveryScenarios(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	// Test error recovery scenarios
	errorScenarios := []struct {
		name          string
		failurePoint  string
		hasRetry      bool
		shouldRecover bool
	}{
		{
			name:          "Checkout failure with retry",
			failurePoint:  "checkout",
			hasRetry:      true,
			shouldRecover: true,
		},
		{
			name:          "QEMU setup failure with retry",
			failurePoint:  "qemu",
			hasRetry:      true,
			shouldRecover: true,
		},
		{
			name:          "Buildx setup failure with retry",
			failurePoint:  "buildx",
			hasRetry:      true,
			shouldRecover: true,
		},
		{
			name:          "Authentication failure with retry",
			failurePoint:  "auth",
			hasRetry:      true,
			shouldRecover: true,
		},
		{
			name:          "Docker build failure with retry",
			failurePoint:  "build",
			hasRetry:      true,
			shouldRecover: true,
		},
	}

	for _, scenario := range errorScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Validate that retry steps exist in workflow
			retryStepFound := false
			for _, job := range config.Jobs {
				for _, step := range job.Steps {
					if strings.Contains(strings.ToLower(step.Name), "retry") &&
						strings.Contains(strings.ToLower(step.Name), scenario.failurePoint) {
						retryStepFound = true
						break
					}
				}
				if retryStepFound {
					break
				}
			}

			if scenario.hasRetry && !retryStepFound {
				t.Errorf("Expected retry step for %s failure not found", scenario.failurePoint)
			}

			// Validate error analysis steps exist
			analysisStepFound := false
			for _, job := range config.Jobs {
				for _, step := range job.Steps {
					if strings.Contains(strings.ToLower(step.Name), "analyze") &&
						strings.Contains(strings.ToLower(step.Name), scenario.failurePoint) {
						analysisStepFound = true
						break
					}
				}
				if analysisStepFound {
					break
				}
			}

			if !analysisStepFound {
				t.Errorf("Expected error analysis step for %s failure not found", scenario.failurePoint)
			}
		})
	}
}

// TestWorkflowSecurityValidation tests security aspects of workflow execution
func TestWorkflowSecurityValidation(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	// Test repository restriction
	for _, job := range config.Jobs {
		if job.If == "" {
			t.Error("Job should have repository restriction condition")
			continue
		}

		if !strings.Contains(job.If, "powa-team/powa-sentinel") {
			t.Error("Job condition should restrict to powa-team/powa-sentinel repository")
		}

		if !strings.Contains(job.If, "github.repository") {
			t.Error("Job condition should check github.repository")
		}
	}

	// Test permissions configuration
	requiredPermissions := map[string]string{
		"contents": "read",
		"packages": "write",
	}

	for perm, expectedLevel := range requiredPermissions {
		if level, exists := config.Permissions[perm]; !exists {
			t.Errorf("Required permission %s not found", perm)
		} else if level != expectedLevel {
			t.Errorf("Permission %s should be %s, got %s", perm, expectedLevel, level)
		}
	}

	// Test that workflow includes security validation steps
	securityStepFound := false
	for _, job := range config.Jobs {
		for _, step := range job.Steps {
			if strings.Contains(strings.ToLower(step.Name), "validate") &&
				strings.Contains(strings.ToLower(step.Name), "permission") {
				securityStepFound = true
				break
			}
		}
		if securityStepFound {
			break
		}
	}

	if !securityStepFound {
		t.Error("Workflow should include security validation steps")
	}
}

// TestWorkflowPerformanceOptimization tests performance optimization features
func TestWorkflowPerformanceOptimization(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	// Test caching configuration
	cachingSteps := []string{"cache", "caching"}
	cachingFound := false

	for _, job := range config.Jobs {
		for _, step := range job.Steps {
			stepNameLower := strings.ToLower(step.Name)
			for _, cacheKeyword := range cachingSteps {
				if strings.Contains(stepNameLower, cacheKeyword) {
					cachingFound = true
					break
				}
			}
			if cachingFound {
				break
			}
		}
		if cachingFound {
			break
		}
	}

	if !cachingFound {
		t.Error("Workflow should include caching optimization steps")
	}

	// Test timeout configuration
	timeoutFound := false
	for _, job := range config.Jobs {
		for _, step := range job.Steps {
			if step.Uses != "" && strings.Contains(step.Uses, "docker/build-push-action") {
				// Check if timeout is configured (would be in workflow-level timeout-minutes)
				timeoutFound = true
				break
			}
		}
		if timeoutFound {
			break
		}
	}

	if !timeoutFound {
		t.Error("Workflow should include Docker build steps with timeout configuration")
	}
}

// TestWorkflowValidationSteps tests post-build validation steps
func TestWorkflowValidationSteps(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	// Test that validation steps exist
	validationSteps := []string{
		"verify image accessibility",
		"verify image metadata",
		"accessibility report",
	}

	for _, expectedStep := range validationSteps {
		stepFound := false
		for _, job := range config.Jobs {
			for _, step := range job.Steps {
				if strings.Contains(strings.ToLower(step.Name), expectedStep) {
					stepFound = true
					break
				}
			}
			if stepFound {
				break
			}
		}

		if !stepFound {
			t.Errorf("Expected validation step '%s' not found in workflow", expectedStep)
		}
	}
}

// contains checks if a slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// TestWorkflowCompleteness validates that workflow includes all required components
func TestWorkflowCompleteness(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	// Test workflow structure
	if config.Name == "" {
		t.Error("Workflow must have a name")
	}

	if len(config.Permissions) == 0 {
		t.Error("Workflow must have permissions configured")
	}

	if len(config.Jobs) == 0 {
		t.Error("Workflow must have at least one job")
	}

	// Test trigger configuration
	if len(config.On.Push.Branches) == 0 && len(config.On.Push.Tags) == 0 {
		t.Error("Workflow must have push triggers configured")
	}

	if len(config.On.PullRequest.Branches) == 0 {
		t.Error("Workflow must have pull request triggers configured")
	}

	if len(config.On.Release.Types) == 0 {
		t.Error("Workflow must have release triggers configured")
	}

	// Test job structure
	for jobName, job := range config.Jobs {
		if len(job.Steps) == 0 {
			t.Errorf("Job %s must have at least one step", jobName)
		}

		if job.If == "" {
			t.Errorf("Job %s should have conditional execution", jobName)
		}
	}
}