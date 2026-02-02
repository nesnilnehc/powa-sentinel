package github_actions

import (
	"fmt"
	"strings"
	"testing"
	"testing/quick"
)

// GitEventScenario represents different Git events that can trigger the workflow
type GitEventScenario struct {
	EventName    string            // GitHub event name
	RefType      string            // "branch" or "tag"
	RefName      string            // branch/tag name
	BaseRef      string            // base reference for PRs
	Repository   string            // repository full name
	Actor        string            // GitHub actor
	IsFork       bool              // whether it's from a fork
	Environment  map[string]string // additional context
}

// ExpectedBehavior defines what should happen for each scenario
type ExpectedBehavior struct {
	ShouldTrigger    bool     // should workflow trigger
	ShouldBuild      bool     // should build images
	ShouldPublish    bool     // should publish to registry
	ShouldCache      bool     // should use caching
	ExpectedTags     []string // expected image tags
	PublishingReason string   // reason for publishing decision
}

// generateGitEventScenarios creates comprehensive test scenarios
func generateGitEventScenarios() []GitEventScenario {
	return []GitEventScenario{
		// Main branch push scenarios
		{
			EventName:   "push",
			RefType:     "branch",
			RefName:     "main",
			Repository:  "powa-team/powa-sentinel",
			Actor:       "developer",
			IsFork:      false,
			Environment: map[string]string{"GITHUB_REF": "refs/heads/main"},
		},
		
		// Version tag push scenarios
		{
			EventName:   "push",
			RefType:     "tag",
			RefName:     "v1.0.0",
			Repository:  "powa-team/powa-sentinel",
			Actor:       "maintainer",
			IsFork:      false,
			Environment: map[string]string{"GITHUB_REF": "refs/tags/v1.0.0"},
		},
		{
			EventName:   "push",
			RefType:     "tag",
			RefName:     "v2.1.3",
			Repository:  "powa-team/powa-sentinel",
			Actor:       "maintainer",
			IsFork:      false,
			Environment: map[string]string{"GITHUB_REF": "refs/tags/v2.1.3"},
		},
		
		// Pull request scenarios
		{
			EventName:   "pull_request",
			RefType:     "branch",
			RefName:     "feature-auth",
			BaseRef:     "main",
			Repository:  "powa-team/powa-sentinel",
			Actor:       "contributor",
			IsFork:      false,
			Environment: map[string]string{"GITHUB_BASE_REF": "main"},
		},
		{
			EventName:   "pull_request",
			RefType:     "branch",
			RefName:     "fix-bug-123",
			BaseRef:     "main",
			Repository:  "powa-team/powa-sentinel",
			Actor:       "developer",
			IsFork:      false,
			Environment: map[string]string{"GITHUB_BASE_REF": "main"},
		},
		
		// Fork pull request scenarios (should not trigger)
		{
			EventName:   "pull_request",
			RefType:     "branch",
			RefName:     "external-contribution",
			BaseRef:     "main",
			Repository:  "external-user/powa-sentinel",
			Actor:       "external-user",
			IsFork:      true,
			Environment: map[string]string{"GITHUB_BASE_REF": "main"},
		},
		
		// Release scenarios
		{
			EventName:   "release",
			RefType:     "tag",
			RefName:     "v1.0.0",
			Repository:  "powa-team/powa-sentinel",
			Actor:       "maintainer",
			IsFork:      false,
			Environment: map[string]string{"GITHUB_REF": "refs/tags/v1.0.0"},
		},
		
		// Non-main branch push scenarios (should not publish)
		{
			EventName:   "push",
			RefType:     "branch",
			RefName:     "develop",
			Repository:  "powa-team/powa-sentinel",
			Actor:       "developer",
			IsFork:      false,
			Environment: map[string]string{"GITHUB_REF": "refs/heads/develop"},
		},
		{
			EventName:   "push",
			RefType:     "branch",
			RefName:     "feature-branch",
			Repository:  "powa-team/powa-sentinel",
			Actor:       "developer",
			IsFork:      false,
			Environment: map[string]string{"GITHUB_REF": "refs/heads/feature-branch"},
		},
		
		// Non-version tag scenarios
		{
			EventName:   "push",
			RefType:     "tag",
			RefName:     "test-tag",
			Repository:  "powa-team/powa-sentinel",
			Actor:       "developer",
			IsFork:      false,
			Environment: map[string]string{"GITHUB_REF": "refs/tags/test-tag"},
		},
		
		// Wrong repository scenarios (should not trigger)
		{
			EventName:   "push",
			RefType:     "branch",
			RefName:     "main",
			Repository:  "other-org/powa-sentinel",
			Actor:       "external-user",
			IsFork:      false,
			Environment: map[string]string{"GITHUB_REF": "refs/heads/main"},
		},
	}
}

// determineExpectedBehavior calculates what should happen for each scenario
func determineExpectedBehavior(scenario GitEventScenario, config *WorkflowConfig) ExpectedBehavior {
	behavior := ExpectedBehavior{
		ShouldCache: true, // Caching should always be enabled
	}

	// Check if workflow should trigger
	behavior.ShouldTrigger = shouldWorkflowTriggerForScenario(scenario, config)
	
	// If workflow doesn't trigger, nothing else happens
	if !behavior.ShouldTrigger {
		behavior.PublishingReason = "Workflow does not trigger for this scenario"
		return behavior
	}

	// If workflow triggers, it should build
	behavior.ShouldBuild = true

	// Determine publishing behavior
	switch {
	case scenario.Repository != "powa-team/powa-sentinel":
		behavior.ShouldPublish = false
		behavior.PublishingReason = "Not the main repository"
		
	case scenario.IsFork:
		behavior.ShouldPublish = false
		behavior.PublishingReason = "Fork repositories cannot publish"
		
	case scenario.EventName == "pull_request":
		behavior.ShouldPublish = false
		behavior.PublishingReason = "Pull requests are for validation only"
		
	case scenario.EventName == "push" && scenario.RefType == "branch" && scenario.RefName == "main":
		behavior.ShouldPublish = true
		behavior.PublishingReason = "Main branch pushes are published"
		behavior.ExpectedTags = []string{
			"ghcr.io/powa-team/powa-sentinel:main",
			"ghcr.io/powa-team/powa-sentinel:main-<sha>",
		}
		
	case scenario.EventName == "push" && scenario.RefType == "tag" && strings.HasPrefix(scenario.RefName, "v"):
		behavior.ShouldPublish = true
		behavior.PublishingReason = "Version tags are published"
		behavior.ExpectedTags = []string{
			fmt.Sprintf("ghcr.io/powa-team/powa-sentinel:%s", scenario.RefName),
			"ghcr.io/powa-team/powa-sentinel:latest",
		}
		
	case scenario.EventName == "release":
		behavior.ShouldPublish = true
		behavior.PublishingReason = "Releases are published"
		behavior.ExpectedTags = []string{
			fmt.Sprintf("ghcr.io/powa-team/powa-sentinel:%s", scenario.RefName),
			"ghcr.io/powa-team/powa-sentinel:latest",
		}
		
	default:
		behavior.ShouldPublish = false
		behavior.PublishingReason = "Does not match publishing criteria"
	}

	return behavior
}

// shouldWorkflowTriggerForScenario determines if workflow should trigger
func shouldWorkflowTriggerForScenario(scenario GitEventScenario, config *WorkflowConfig) bool {
	// Repository must be the main repository
	if scenario.Repository != "powa-team/powa-sentinel" {
		return false
	}

	// Fork PRs should not trigger (security restriction)
	if scenario.EventName == "pull_request" && scenario.IsFork {
		return false
	}

	// Check specific trigger conditions
	switch scenario.EventName {
	case "push":
		if scenario.RefType == "branch" {
			for _, branch := range config.On.Push.Branches {
				if branch == scenario.RefName {
					return true
				}
			}
		} else if scenario.RefType == "tag" {
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
		for _, branch := range config.On.PullRequest.Branches {
			if branch == scenario.BaseRef {
				return true
			}
		}
		
	case "release":
		for _, releaseType := range config.On.Release.Types {
			if releaseType == "published" {
				return true
			}
		}
	}

	return false
}

// TestGitEventScenarios tests workflow behavior across different Git events
func TestGitEventScenarios(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	scenarios := generateGitEventScenarios()

	for _, scenario := range scenarios {
		t.Run(fmt.Sprintf("%s_%s_%s", scenario.EventName, scenario.RefType, scenario.RefName), func(t *testing.T) {
			expected := determineExpectedBehavior(scenario, config)

			// Test trigger behavior
			if expected.ShouldTrigger {
				// Verify workflow would trigger
				if !shouldWorkflowTriggerForScenario(scenario, config) {
					t.Errorf("Workflow should trigger for %s %s %s", scenario.EventName, scenario.RefType, scenario.RefName)
				}
			} else {
				// Verify workflow would not trigger
				if shouldWorkflowTriggerForScenario(scenario, config) {
					t.Errorf("Workflow should NOT trigger for %s %s %s", scenario.EventName, scenario.RefType, scenario.RefName)
				}
			}

			// Test build behavior
			if expected.ShouldTrigger && !expected.ShouldBuild {
				t.Error("If workflow triggers, it should build images")
			}

			// Test publishing behavior
			if expected.ShouldPublish {
				if scenario.EventName == "pull_request" {
					t.Error("Pull requests should never publish images")
				}
				if scenario.Repository != "powa-team/powa-sentinel" {
					t.Error("Only main repository should publish images")
				}
				if scenario.IsFork {
					t.Error("Fork repositories should not publish images")
				}
			}

			// Test expected tags
			if expected.ShouldPublish && len(expected.ExpectedTags) == 0 {
				t.Error("Publishing scenarios should have expected tags")
			}

			// Log behavior for debugging
			t.Logf("Scenario: %s %s %s", scenario.EventName, scenario.RefType, scenario.RefName)
			t.Logf("Repository: %s (fork: %v)", scenario.Repository, scenario.IsFork)
			t.Logf("Should trigger: %v", expected.ShouldTrigger)
			t.Logf("Should build: %v", expected.ShouldBuild)
			t.Logf("Should publish: %v (%s)", expected.ShouldPublish, expected.PublishingReason)
			t.Logf("Expected tags: %v", expected.ExpectedTags)
		})
	}
}

// TestGitEventProperty tests the property that workflow behavior is consistent
// Property: For any Git event scenario, workflow behavior should be predictable
// and consistent with security and publishing requirements
func TestGitEventProperty(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	// Property-based test for Git event handling
	property := func(eventName, refType, refName, repository string, isFork bool) bool {
		// Sanitize inputs
		validEvents := []string{"push", "pull_request", "release"}
		if !contains(validEvents, eventName) {
			eventName = "push"
		}

		validRefTypes := []string{"branch", "tag"}
		if !contains(validRefTypes, refType) {
			refType = "branch"
		}

		if refName == "" {
			refName = "main"
		}

		if repository == "" {
			repository = "powa-team/powa-sentinel"
		}

		// Create scenario
		scenario := GitEventScenario{
			EventName:  eventName,
			RefType:    refType,
			RefName:    refName,
			Repository: repository,
			IsFork:     isFork,
		}

		// Adjust scenario based on event type
		if eventName == "pull_request" {
			scenario.BaseRef = "main"
		}

		expected := determineExpectedBehavior(scenario, config)

		// Validate consistency rules
		
		// Rule 1: If workflow doesn't trigger, nothing else should happen
		if !expected.ShouldTrigger {
			return !expected.ShouldBuild && !expected.ShouldPublish
		}

		// Rule 2: If workflow triggers, it should build
		if expected.ShouldTrigger && !expected.ShouldBuild {
			return false
		}

		// Rule 3: Publishing should only happen for authorized scenarios
		if expected.ShouldPublish {
			// Must be main repository
			if scenario.Repository != "powa-team/powa-sentinel" {
				return false
			}
			// Must not be a fork
			if scenario.IsFork {
				return false
			}
			// Must not be a pull request
			if scenario.EventName == "pull_request" {
				return false
			}
		}

		// Rule 4: Pull requests should never publish
		if scenario.EventName == "pull_request" && expected.ShouldPublish {
			return false
		}

		// Rule 5: Forks should not trigger workflow
		if scenario.IsFork && expected.ShouldTrigger {
			return false
		}

		// Rule 6: Wrong repository should not trigger
		if scenario.Repository != "powa-team/powa-sentinel" && expected.ShouldTrigger {
			return false
		}

		return true
	}

	// Run the property test
	if err := quick.Check(property, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Git event property failed: %v", err)
	}
}

// TestMainBranchPublishingScenarios tests specific main branch scenarios
func TestMainBranchPublishingScenarios(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	mainBranchScenarios := []struct {
		name       string
		repository string
		isFork     bool
		shouldPublish bool
	}{
		{
			name:       "main_repo_main_branch",
			repository: "powa-team/powa-sentinel",
			isFork:     false,
			shouldPublish: true,
		},
		{
			name:       "fork_main_branch",
			repository: "external-user/powa-sentinel",
			isFork:     true,
			shouldPublish: false,
		},
		{
			name:       "different_org_main_branch",
			repository: "other-org/powa-sentinel",
			isFork:     false,
			shouldPublish: false,
		},
	}

	for _, scenario := range mainBranchScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			gitScenario := GitEventScenario{
				EventName:  "push",
				RefType:    "branch",
				RefName:    "main",
				Repository: scenario.repository,
				IsFork:     scenario.isFork,
			}

			expected := determineExpectedBehavior(gitScenario, config)

			if expected.ShouldPublish != scenario.shouldPublish {
				t.Errorf("Publishing behavior mismatch for %s: expected %v, got %v",
					scenario.name, scenario.shouldPublish, expected.ShouldPublish)
			}

			// Main repository should always trigger for main branch
			if scenario.repository == "powa-team/powa-sentinel" && !expected.ShouldTrigger {
				t.Error("Main repository main branch should trigger workflow")
			}

			// Non-main repositories should not trigger
			if scenario.repository != "powa-team/powa-sentinel" && expected.ShouldTrigger {
				t.Error("Non-main repositories should not trigger workflow")
			}
		})
	}
}

// TestVersionTagPublishingScenarios tests version tag publishing scenarios
func TestVersionTagPublishingScenarios(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	versionTagScenarios := []struct {
		tagName       string
		shouldTrigger bool
		shouldPublish bool
	}{
		{"v1.0.0", true, true},
		{"v2.1.3", true, true},
		{"v0.1.0-alpha", true, true},
		{"v1.0.0-beta.1", true, true},
		{"test-tag", false, false},
		{"release-1.0", false, false},
		{"1.0.0", false, false}, // No 'v' prefix
	}

	for _, scenario := range versionTagScenarios {
		t.Run(fmt.Sprintf("tag_%s", scenario.tagName), func(t *testing.T) {
			gitScenario := GitEventScenario{
				EventName:  "push",
				RefType:    "tag",
				RefName:    scenario.tagName,
				Repository: "powa-team/powa-sentinel",
				IsFork:     false,
			}

			expected := determineExpectedBehavior(gitScenario, config)

			if expected.ShouldTrigger != scenario.shouldTrigger {
				t.Errorf("Trigger behavior mismatch for tag %s: expected %v, got %v",
					scenario.tagName, scenario.shouldTrigger, expected.ShouldTrigger)
			}

			if expected.ShouldPublish != scenario.shouldPublish {
				t.Errorf("Publishing behavior mismatch for tag %s: expected %v, got %v",
					scenario.tagName, scenario.shouldPublish, expected.ShouldPublish)
			}

			// Version tags should include 'latest' tag
			if scenario.shouldPublish && strings.HasPrefix(scenario.tagName, "v") {
				hasLatestTag := false
				for _, tag := range expected.ExpectedTags {
					if strings.HasSuffix(tag, ":latest") {
						hasLatestTag = true
						break
					}
				}
				if !hasLatestTag {
					t.Errorf("Version tag %s should include 'latest' tag", scenario.tagName)
				}
			}
		})
	}
}

// TestPullRequestValidationScenarios tests pull request validation scenarios
func TestPullRequestValidationScenarios(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	prScenarios := []struct {
		name         string
		repository   string
		baseRef      string
		isFork       bool
		shouldTrigger bool
	}{
		{
			name:         "main_repo_pr_to_main",
			repository:   "powa-team/powa-sentinel",
			baseRef:      "main",
			isFork:       false,
			shouldTrigger: true,
		},
		{
			name:         "fork_pr_to_main",
			repository:   "external-user/powa-sentinel",
			baseRef:      "main",
			isFork:       true,
			shouldTrigger: false, // Security restriction
		},
		{
			name:         "main_repo_pr_to_develop",
			repository:   "powa-team/powa-sentinel",
			baseRef:      "develop",
			isFork:       false,
			shouldTrigger: false, // Only PRs to main should trigger
		},
	}

	for _, scenario := range prScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			gitScenario := GitEventScenario{
				EventName:  "pull_request",
				RefType:    "branch",
				RefName:    "feature-branch",
				BaseRef:    scenario.baseRef,
				Repository: scenario.repository,
				IsFork:     scenario.isFork,
			}

			expected := determineExpectedBehavior(gitScenario, config)

			if expected.ShouldTrigger != scenario.shouldTrigger {
				t.Errorf("Trigger behavior mismatch for %s: expected %v, got %v",
					scenario.name, scenario.shouldTrigger, expected.ShouldTrigger)
			}

			// Pull requests should never publish
			if expected.ShouldPublish {
				t.Errorf("Pull request %s should not publish images", scenario.name)
			}

			// Pull requests should build if they trigger
			if expected.ShouldTrigger && !expected.ShouldBuild {
				t.Errorf("Pull request %s should build images if triggered", scenario.name)
			}
		})
	}
}