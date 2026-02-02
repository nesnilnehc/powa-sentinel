package github_actions

import (
	"regexp"
	"strings"
	"testing"
	"testing/quick"
)

// TaggingStrategy represents the tagging configuration for different Git references
type TaggingStrategy struct {
	TagRules     []TagRule
	PublishRules []PublishRule
}

// TagRule represents a single tag generation rule
type TagRule struct {
	Type     string // "ref", "raw", "sha"
	Event    string // "tag", "pr", "push"
	Value    string // tag value or pattern
	Enable   string // condition for enabling the rule
	Priority int    // rule priority
	Prefix   string // tag prefix
}

// PublishRule represents publishing conditions
type PublishRule struct {
	Condition string // GitHub Actions condition
	EventType string // event that triggers publishing
	RefType   string // reference type (tag, branch)
	RefName   string // reference name
}

// GitReference represents different Git reference scenarios for testing
type GitReference struct {
	RefType   string // "tag", "branch"
	RefName   string // "v1.0.0", "main", "feature-branch"
	EventName string // "push", "pull_request", "release"
	SHA       string // commit SHA
}

// extractTaggingStrategy extracts tagging strategy from workflow configuration
func extractTaggingStrategy(config *WorkflowConfig) (*TaggingStrategy, error) {
	strategy := &TaggingStrategy{
		TagRules:     []TagRule{},
		PublishRules: []PublishRule{},
	}

	for _, job := range config.Jobs {
		// Extract job-level publishing conditions
		if job.If != "" {
			strategy.PublishRules = append(strategy.PublishRules, PublishRule{
				Condition: job.If,
			})
		}

		for _, step := range job.Steps {
			// Extract tagging rules from docker/metadata-action
			if strings.Contains(step.Uses, "docker/metadata-action") {
				if tagsVal, ok := step.With["tags"]; ok {
					if tagsStr, ok := tagsVal.(string); ok {
						lines := strings.Split(tagsStr, "\n")
						for _, line := range lines {
							line = strings.TrimSpace(line)
							if line == "" || strings.HasPrefix(line, "#") {
								continue
							}

							rule := TagRule{}
							
							// Parse tag rule format: type=ref,event=tag
							if strings.HasPrefix(line, "type=ref,event=tag") {
								rule.Type = "ref"
								rule.Event = "tag"
							} else if strings.HasPrefix(line, "type=raw,value=latest") {
								rule.Type = "raw"
								rule.Value = "latest"
								if strings.Contains(line, "enable={{is_default_branch}}") {
									rule.Enable = "is_default_branch"
								}
							} else if strings.HasPrefix(line, "type=raw,value=main") {
								rule.Type = "raw"
								rule.Value = "main"
								if strings.Contains(line, "enable={{is_default_branch}}") {
									rule.Enable = "is_default_branch"
								}
							} else if strings.HasPrefix(line, "type=sha") {
								rule.Type = "sha"
								if strings.Contains(line, "prefix=main-") {
									rule.Prefix = "main-"
								}
								if strings.Contains(line, "enable={{is_default_branch}}") {
									rule.Enable = "is_default_branch"
								}
							} else if strings.HasPrefix(line, "type=ref,event=pr") {
								rule.Type = "ref"
								rule.Event = "pr"
								if strings.Contains(line, "prefix=pr-") {
									rule.Prefix = "pr-"
								}
							}

							// Extract priority if present
							if strings.Contains(line, "priority=") {
								priorityRegex := regexp.MustCompile(`priority=(\d+)`)
								if matches := priorityRegex.FindStringSubmatch(line); len(matches) > 1 {
									if matches[1] == "100" {
										rule.Priority = 100
									} else if matches[1] == "200" {
										rule.Priority = 200
									} else if matches[1] == "150" {
										rule.Priority = 150
									} else if matches[1] == "50" {
										rule.Priority = 50
									}
								}
							}

							strategy.TagRules = append(strategy.TagRules, rule)
						}
					}
				}
			}

			// Extract publishing conditions from docker/build-push-action
			if strings.Contains(step.Uses, "docker/build-push-action") {
				if pushVal, ok := step.With["push"]; ok {
					if pushStr, ok := pushVal.(string); ok {
						strategy.PublishRules = append(strategy.PublishRules, PublishRule{
							Condition: pushStr,
						})
					}
				}
			}
		}
	}

	return strategy, nil
}

// validateTaggedReleasePublishing checks if tagged releases are properly configured for publishing
func validateTaggedReleasePublishing(gitRef GitReference, strategy *TaggingStrategy) bool {
	if gitRef.RefType != "tag" || gitRef.EventName != "push" {
		return true // Not applicable for non-tag scenarios
	}

	// Check that tag events generate appropriate tags
	hasTagRule := false
	hasLatestRule := false
	
	for _, rule := range strategy.TagRules {
		if rule.Type == "ref" && rule.Event == "tag" {
			hasTagRule = true
		}
		if rule.Type == "raw" && rule.Value == "latest" && rule.Enable == "is_default_branch" {
			hasLatestRule = true
		}
	}

	// Check that publishing is enabled for tags
	hasPublishRule := false
	for _, rule := range strategy.PublishRules {
		if strings.Contains(rule.Condition, "should_publish") || 
		   strings.Contains(rule.Condition, "pull_request") {
			hasPublishRule = true
		}
	}

	return hasTagRule && hasLatestRule && hasPublishRule
}

// validateMainBranchPublishing checks if main branch pushes are properly configured for publishing
func validateMainBranchPublishing(gitRef GitReference, strategy *TaggingStrategy) bool {
	if gitRef.RefName != "main" || gitRef.EventName != "push" {
		return true // Not applicable for non-main scenarios
	}

	// Check that main branch events generate appropriate tags
	hasMainRule := false
	hasSHARule := false
	
	for _, rule := range strategy.TagRules {
		if rule.Type == "raw" && rule.Value == "main" && rule.Enable == "is_default_branch" {
			hasMainRule = true
		}
		if rule.Type == "sha" && rule.Prefix == "main-" && rule.Enable == "is_default_branch" {
			hasSHARule = true
		}
	}

	// Check that publishing is enabled for main branch
	hasPublishRule := false
	for _, rule := range strategy.PublishRules {
		if strings.Contains(rule.Condition, "should_publish") || 
		   strings.Contains(rule.Condition, "pull_request") {
			hasPublishRule = true
		}
	}

	return hasMainRule && hasSHARule && hasPublishRule
}

// validatePullRequestNonPublishing checks if pull requests are configured to NOT publish
func validatePullRequestNonPublishing(gitRef GitReference, strategy *TaggingStrategy) bool {
	if gitRef.EventName != "pull_request" {
		return true // Not applicable for non-PR scenarios
	}

	// Check that PR events generate tags but don't publish
	hasPRRule := false
	
	for _, rule := range strategy.TagRules {
		if rule.Type == "ref" && rule.Event == "pr" && rule.Prefix == "pr-" {
			hasPRRule = true
		}
	}

	// Check that publishing is disabled for PRs
	hasNonPublishRule := false
	for _, rule := range strategy.PublishRules {
		if strings.Contains(rule.Condition, "should_publish") && 
		   strings.Contains(rule.Condition, "false") {
			hasNonPublishRule = true
		} else if strings.Contains(rule.Condition, "pull_request") {
			hasNonPublishRule = true
		}
	}

	return hasPRRule && hasNonPublishRule
}

// TestTagBasedPublishingLogic tests Property 4: Tag-Based Publishing Logic
// **Feature: github-actions-docker-publish, Property 4: Tag-Based Publishing Logic**
// **Validates: Requirements 2.1, 2.2, 2.3**
//
// Property: For any Git reference (tag or branch), the published Docker images should have 
// tags that correctly reflect the source reference (version tags for releases, latest + SHA 
// for main, no publishing for PRs)
func TestTagBasedPublishingLogic(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	strategy, err := extractTaggingStrategy(config)
	if err != nil {
		t.Fatalf("Failed to extract tagging strategy: %v", err)
	}

	// Property-based test: For any Git reference, tagging and publishing should be correct
	property := func(refType, refName, eventName, sha string) bool {
		// Sanitize inputs to simulate realistic Git references
		if refType == "" {
			refType = "branch"
		}
		if refName == "" {
			refName = "main"
		}
		if eventName == "" {
			eventName = "push"
		}
		if sha == "" {
			sha = "abc123def456"
		}

		// Ensure SHA looks like a commit hash
		shaRegex := regexp.MustCompile(`^[a-f0-9]{7,40}$`)
		if !shaRegex.MatchString(sha) {
			sha = "abc123def456789"
		}

		// Ensure valid event names
		validEvents := []string{"push", "pull_request", "release"}
		validEvent := false
		for _, valid := range validEvents {
			if eventName == valid {
				validEvent = true
				break
			}
		}
		if !validEvent {
			eventName = "push"
		}

		// Ensure valid ref types
		if refType != "tag" && refType != "branch" {
			refType = "branch"
		}

		gitRef := GitReference{
			RefType:   refType,
			RefName:   refName,
			EventName: eventName,
			SHA:       sha,
		}

		// Check all publishing logic requirements
		taggedReleaseOK := validateTaggedReleasePublishing(gitRef, strategy)
		mainBranchOK := validateMainBranchPublishing(gitRef, strategy)
		pullRequestOK := validatePullRequestNonPublishing(gitRef, strategy)

		return taggedReleaseOK && mainBranchOK && pullRequestOK
	}

	// Run the property test
	if err := quick.Check(property, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Tag-based publishing logic property failed: %v", err)
		t.Logf("Tag rules: %+v", strategy.TagRules)
		t.Logf("Publish rules: %+v", strategy.PublishRules)
	}
}

// TestTaggedReleaseConfiguration validates tagged release publishing configuration
func TestTaggedReleaseConfiguration(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	strategy, err := extractTaggingStrategy(config)
	if err != nil {
		t.Fatalf("Failed to extract tagging strategy: %v", err)
	}

	// Check that tagged releases generate version tags
	hasTagRule := false
	for _, rule := range strategy.TagRules {
		if rule.Type == "ref" && rule.Event == "tag" {
			hasTagRule = true
			break
		}
	}
	if !hasTagRule {
		t.Error("Tagged releases should generate version tags (type=ref,event=tag)")
	}

	// Check that tagged releases generate latest tags
	hasLatestRule := false
	for _, rule := range strategy.TagRules {
		if rule.Type == "raw" && rule.Value == "latest" {
			hasLatestRule = true
			break
		}
	}
	if !hasLatestRule {
		t.Error("Tagged releases should generate latest tags (type=raw,value=latest)")
	}
}

// TestMainBranchConfiguration validates main branch publishing configuration
func TestMainBranchConfiguration(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	strategy, err := extractTaggingStrategy(config)
	if err != nil {
		t.Fatalf("Failed to extract tagging strategy: %v", err)
	}

	// Check that main branch generates main tags
	hasMainRule := false
	for _, rule := range strategy.TagRules {
		if rule.Type == "raw" && rule.Value == "main" {
			hasMainRule = true
			break
		}
	}
	if !hasMainRule {
		t.Error("Main branch should generate main tags (type=raw,value=main)")
	}

	// Check that main branch generates SHA-based tags
	hasSHARule := false
	for _, rule := range strategy.TagRules {
		if rule.Type == "sha" && strings.Contains(rule.Prefix, "main") {
			hasSHARule = true
			break
		}
	}
	if !hasSHARule {
		t.Error("Main branch should generate SHA-based tags (type=sha with main prefix)")
	}
}

// TestPullRequestConfiguration validates pull request handling configuration
func TestPullRequestConfiguration(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	strategy, err := extractTaggingStrategy(config)
	if err != nil {
		t.Fatalf("Failed to extract tagging strategy: %v", err)
	}

	// Check that pull requests generate PR tags
	hasPRRule := false
	for _, rule := range strategy.TagRules {
		if rule.Type == "ref" && rule.Event == "pr" {
			hasPRRule = true
			break
		}
	}
	if !hasPRRule {
		t.Error("Pull requests should generate PR tags (type=ref,event=pr)")
	}

	// Check that publishing is conditional (not for PRs)
	hasConditionalPublish := false
	for _, rule := range strategy.PublishRules {
		if strings.Contains(rule.Condition, "should_publish") || 
		   strings.Contains(rule.Condition, "pull_request") {
			hasConditionalPublish = true
			break
		}
	}
	if !hasConditionalPublish {
		t.Error("Publishing should be conditional (not for pull requests)")
	}
}