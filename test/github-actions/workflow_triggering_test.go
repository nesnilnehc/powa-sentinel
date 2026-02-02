package github_actions

import (
	"strings"
	"testing"
	"testing/quick"
)

// WorkflowTriggers represents the trigger configuration for the workflow
type WorkflowTriggers struct {
	PushTriggers        []PushTrigger
	PullRequestTriggers []PullRequestTrigger
	ReleaseTriggers     []ReleaseTrigger
}

// PushTrigger represents push event triggers
type PushTrigger struct {
	Branches []string
	Tags     []string
}

// PullRequestTrigger represents pull request event triggers
type PullRequestTrigger struct {
	Branches []string
}

// ReleaseTrigger represents release event triggers
type ReleaseTrigger struct {
	Types []string
}

// GitEvent represents different Git events for testing
type GitEvent struct {
	EventType string   // "push", "pull_request", "release"
	RefType   string   // "branch", "tag"
	RefName   string   // "main", "v1.0.0", "feature-branch"
	Branches  []string // target branches for PR
	Types     []string // release types
}

// extractWorkflowTriggers extracts trigger configuration from workflow
func extractWorkflowTriggers(config *WorkflowConfig) (*WorkflowTriggers, error) {
	triggers := &WorkflowTriggers{
		PushTriggers:        []PushTrigger{},
		PullRequestTriggers: []PullRequestTrigger{},
		ReleaseTriggers:     []ReleaseTrigger{},
	}

	// Extract push triggers
	if len(config.On.Push.Branches) > 0 || len(config.On.Push.Tags) > 0 {
		triggers.PushTriggers = append(triggers.PushTriggers, PushTrigger{
			Branches: config.On.Push.Branches,
			Tags:     config.On.Push.Tags,
		})
	}

	// Extract pull request triggers
	if len(config.On.PullRequest.Branches) > 0 {
		triggers.PullRequestTriggers = append(triggers.PullRequestTriggers, PullRequestTrigger{
			Branches: config.On.PullRequest.Branches,
		})
	}

	// Extract release triggers
	if len(config.On.Release.Types) > 0 {
		triggers.ReleaseTriggers = append(triggers.ReleaseTriggers, ReleaseTrigger{
			Types: config.On.Release.Types,
		})
	}

	return triggers, nil
}

// validateMainBranchPushTrigger checks if main branch pushes trigger the workflow
func validateMainBranchPushTrigger(gitEvent GitEvent, triggers *WorkflowTriggers) bool {
	if gitEvent.EventType != "push" || gitEvent.RefType != "branch" || gitEvent.RefName != "main" {
		return true // Not applicable for non-main-branch-push scenarios
	}

	// Check that main branch is in push triggers
	for _, trigger := range triggers.PushTriggers {
		for _, branch := range trigger.Branches {
			if branch == "main" {
				return true
			}
		}
	}

	return false
}

// validateTagPushTrigger checks if tag pushes trigger the workflow
func validateTagPushTrigger(gitEvent GitEvent, triggers *WorkflowTriggers) bool {
	if gitEvent.EventType != "push" || gitEvent.RefType != "tag" {
		return true // Not applicable for non-tag-push scenarios
	}

	// Check that tags are configured in push triggers
	for _, trigger := range triggers.PushTriggers {
		if len(trigger.Tags) > 0 {
			// Check if the tag pattern matches (v* pattern should match v1.0.0, etc.)
			for _, tagPattern := range trigger.Tags {
				if tagPattern == "v*" && strings.HasPrefix(gitEvent.RefName, "v") {
					return true
				}
				if tagPattern == gitEvent.RefName {
					return true
				}
			}
		}
	}

	return false
}

// validatePullRequestTrigger checks if pull requests trigger the workflow
func validatePullRequestTrigger(gitEvent GitEvent, triggers *WorkflowTriggers) bool {
	if gitEvent.EventType != "pull_request" {
		return true // Not applicable for non-PR scenarios
	}

	// Check that pull requests targeting main branch trigger the workflow
	for _, trigger := range triggers.PullRequestTriggers {
		for _, branch := range trigger.Branches {
			if branch == "main" {
				// Check if any of the target branches match
				for _, targetBranch := range gitEvent.Branches {
					if targetBranch == "main" {
						return true
					}
				}
			}
		}
	}

	return false
}

// validateReleaseTrigger checks if release events trigger the workflow
func validateReleaseTrigger(gitEvent GitEvent, triggers *WorkflowTriggers) bool {
	if gitEvent.EventType != "release" {
		return true // Not applicable for non-release scenarios
	}

	// Check that published releases trigger the workflow
	for _, trigger := range triggers.ReleaseTriggers {
		for _, releaseType := range trigger.Types {
			if releaseType == "published" {
				// Check if any of the event types match
				for _, eventType := range gitEvent.Types {
					if eventType == "published" {
						return true
					}
				}
			}
		}
	}

	return false
}

// TestWorkflowTriggering tests Property 1: Workflow Triggering
// **Feature: github-actions-docker-publish, Property 1: Workflow Triggering**
// **Validates: Requirements 1.1, 1.2**
//
// Property: For any valid Git event (push to main, pull request, tag creation), 
// the GitHub Actions workflow should be triggered and execute the appropriate build steps
func TestWorkflowTriggering(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	triggers, err := extractWorkflowTriggers(config)
	if err != nil {
		t.Fatalf("Failed to extract workflow triggers: %v", err)
	}

	// Property-based test: For any valid Git event, workflow should be triggered appropriately
	property := func(eventType, refType, refName string) bool {
		// Sanitize inputs to simulate realistic Git events
		if eventType == "" {
			eventType = "push"
		}
		if refType == "" {
			refType = "branch"
		}
		if refName == "" {
			refName = "main"
		}

		// Ensure valid event types
		validEvents := []string{"push", "pull_request", "release"}
		validEvent := false
		for _, valid := range validEvents {
			if eventType == valid {
				validEvent = true
				break
			}
		}
		if !validEvent {
			eventType = "push"
		}

		// Ensure valid ref types
		if refType != "tag" && refType != "branch" {
			refType = "branch"
		}

		// Create realistic Git event scenarios
		gitEvent := GitEvent{
			EventType: eventType,
			RefType:   refType,
			RefName:   refName,
			Branches:  []string{"main"}, // Default target branch for PRs
			Types:     []string{"published"}, // Default release type
		}

		// Adjust event properties based on type
		if eventType == "pull_request" {
			gitEvent.RefType = "branch" // PRs are always branch-based
		}
		if eventType == "release" {
			gitEvent.RefType = "tag" // Releases are always tag-based
		}

		// Check all trigger requirements
		mainBranchOK := validateMainBranchPushTrigger(gitEvent, triggers)
		tagPushOK := validateTagPushTrigger(gitEvent, triggers)
		pullRequestOK := validatePullRequestTrigger(gitEvent, triggers)
		releaseOK := validateReleaseTrigger(gitEvent, triggers)

		return mainBranchOK && tagPushOK && pullRequestOK && releaseOK
	}

	// Run the property test
	if err := quick.Check(property, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Workflow triggering property failed: %v", err)
		t.Logf("Push triggers: %+v", triggers.PushTriggers)
		t.Logf("Pull request triggers: %+v", triggers.PullRequestTriggers)
		t.Logf("Release triggers: %+v", triggers.ReleaseTriggers)
	}
}

// TestMainBranchPushTrigger validates that pushes to main branch trigger the workflow
func TestMainBranchPushTrigger(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	triggers, err := extractWorkflowTriggers(config)
	if err != nil {
		t.Fatalf("Failed to extract workflow triggers: %v", err)
	}

	// Check that main branch is configured as a push trigger
	hasMainBranch := false
	for _, trigger := range triggers.PushTriggers {
		for _, branch := range trigger.Branches {
			if branch == "main" {
				hasMainBranch = true
				break
			}
		}
		if hasMainBranch {
			break
		}
	}

	if !hasMainBranch {
		t.Error("Workflow should be triggered by pushes to main branch")
	}
}

// TestTagPushTrigger validates that tag pushes trigger the workflow
func TestTagPushTrigger(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	triggers, err := extractWorkflowTriggers(config)
	if err != nil {
		t.Fatalf("Failed to extract workflow triggers: %v", err)
	}

	// Check that tags are configured as push triggers
	hasTagTrigger := false
	for _, trigger := range triggers.PushTriggers {
		if len(trigger.Tags) > 0 {
			hasTagTrigger = true
			break
		}
	}

	if !hasTagTrigger {
		t.Error("Workflow should be triggered by tag pushes")
	}

	// Check that v* pattern is configured for version tags
	hasVersionPattern := false
	for _, trigger := range triggers.PushTriggers {
		for _, tag := range trigger.Tags {
			if tag == "v*" {
				hasVersionPattern = true
				break
			}
		}
		if hasVersionPattern {
			break
		}
	}

	if !hasVersionPattern {
		t.Error("Workflow should be triggered by version tags (v* pattern)")
	}
}

// TestPullRequestTrigger validates that pull requests trigger the workflow
func TestPullRequestTrigger(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	triggers, err := extractWorkflowTriggers(config)
	if err != nil {
		t.Fatalf("Failed to extract workflow triggers: %v", err)
	}

	// Check that pull requests are configured as triggers
	hasPRTrigger := len(triggers.PullRequestTriggers) > 0
	if !hasPRTrigger {
		t.Error("Workflow should be triggered by pull requests")
	}

	// Check that pull requests targeting main branch trigger the workflow
	hasMainTarget := false
	for _, trigger := range triggers.PullRequestTriggers {
		for _, branch := range trigger.Branches {
			if branch == "main" {
				hasMainTarget = true
				break
			}
		}
		if hasMainTarget {
			break
		}
	}

	if !hasMainTarget {
		t.Error("Workflow should be triggered by pull requests targeting main branch")
	}
}

// TestReleaseTrigger validates that release events trigger the workflow
func TestReleaseTrigger(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	triggers, err := extractWorkflowTriggers(config)
	if err != nil {
		t.Fatalf("Failed to extract workflow triggers: %v", err)
	}

	// Check that release events are configured as triggers
	hasReleaseTrigger := len(triggers.ReleaseTriggers) > 0
	if !hasReleaseTrigger {
		t.Error("Workflow should be triggered by release events")
	}

	// Check that published releases trigger the workflow
	hasPublishedType := false
	for _, trigger := range triggers.ReleaseTriggers {
		for _, releaseType := range trigger.Types {
			if releaseType == "published" {
				hasPublishedType = true
				break
			}
		}
		if hasPublishedType {
			break
		}
	}

	if !hasPublishedType {
		t.Error("Workflow should be triggered by published release events")
	}
}

// TestWorkflowTriggerCompleteness validates that all required triggers are configured
func TestWorkflowTriggerCompleteness(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	triggers, err := extractWorkflowTriggers(config)
	if err != nil {
		t.Fatalf("Failed to extract workflow triggers: %v", err)
	}

	// Verify all required trigger types are present
	if len(triggers.PushTriggers) == 0 {
		t.Error("Workflow must have push triggers configured")
	}

	if len(triggers.PullRequestTriggers) == 0 {
		t.Error("Workflow must have pull request triggers configured")
	}

	if len(triggers.ReleaseTriggers) == 0 {
		t.Error("Workflow must have release triggers configured")
	}

	// Verify push triggers include both branches and tags
	hasBranchTrigger := false
	hasTagTrigger := false
	for _, trigger := range triggers.PushTriggers {
		if len(trigger.Branches) > 0 {
			hasBranchTrigger = true
		}
		if len(trigger.Tags) > 0 {
			hasTagTrigger = true
		}
	}

	if !hasBranchTrigger {
		t.Error("Push triggers must include branch triggers")
	}
	if !hasTagTrigger {
		t.Error("Push triggers must include tag triggers")
	}
}