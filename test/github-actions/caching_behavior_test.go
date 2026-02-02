package github_actions

import (
	"regexp"
	"strings"
	"testing"
	"testing/quick"
)

// CacheConfiguration represents the caching configuration for the workflow
type CacheConfiguration struct {
	DockerCacheRules []DockerCacheRule
	GoCacheRules     []GoCacheRule
	CacheScoping     []CacheScopeRule
}

// DockerCacheRule represents Docker layer caching configuration
type DockerCacheRule struct {
	Type     string // "gha", "registry"
	Scope    string // cache scope
	Mode     string // "min", "max"
	Ref      string // registry reference for registry cache
	Fallback bool   // whether this is a fallback cache
}

// GoCacheRule represents Go module caching configuration
type GoCacheRule struct {
	Path        []string // cache paths
	Key         string   // cache key pattern
	RestoreKeys []string // restore key patterns
	SaveAlways  bool     // whether to save cache always
}

// CacheScopeRule represents cache scoping strategy
type CacheScopeRule struct {
	EventType    string // "push", "pull_request", "release"
	RefName      string // "main", "feature-branch"
	Scope        string // cache scope name
	Fallback     string // fallback scope
	Mode         string // cache mode
}

// WorkflowEvent represents different workflow execution scenarios for testing
type WorkflowEvent struct {
	EventName string // "push", "pull_request", "release"
	RefType   string // "branch", "tag"
	RefName   string // "main", "v1.0.0", "feature-branch"
	PRNumber  int    // pull request number (for PR events)
}

// extractCacheConfiguration extracts caching configuration from workflow
func extractCacheConfiguration(config *WorkflowConfig) (*CacheConfiguration, error) {
	cacheConfig := &CacheConfiguration{
		DockerCacheRules: []DockerCacheRule{},
		GoCacheRules:     []GoCacheRule{},
		CacheScoping:     []CacheScopeRule{},
	}

	for _, job := range config.Jobs {
		for _, step := range job.Steps {
			// Extract Go module caching from actions/cache
			if strings.Contains(step.Uses, "actions/cache") {
				rule := GoCacheRule{}
				
				if pathVal, ok := step.With["path"]; ok {
					if pathStr, ok := pathVal.(string); ok {
						rule.Path = strings.Split(pathStr, "\n")
						// Clean up paths
						cleanPaths := []string{}
						for _, path := range rule.Path {
							path = strings.TrimSpace(path)
							if path != "" {
								cleanPaths = append(cleanPaths, path)
							}
						}
						rule.Path = cleanPaths
					}
				}
				
				if keyVal, ok := step.With["key"]; ok {
					if keyStr, ok := keyVal.(string); ok {
						rule.Key = keyStr
					}
				}
				
				if restoreKeysVal, ok := step.With["restore-keys"]; ok {
					if restoreKeysStr, ok := restoreKeysVal.(string); ok {
						rule.RestoreKeys = strings.Split(restoreKeysStr, "\n")
						// Clean up restore keys
						cleanKeys := []string{}
						for _, key := range rule.RestoreKeys {
							key = strings.TrimSpace(key)
							if key != "" {
								cleanKeys = append(cleanKeys, key)
							}
						}
						rule.RestoreKeys = cleanKeys
					}
				}
				
				if saveAlwaysVal, ok := step.With["save-always"]; ok {
					if saveAlwaysStr, ok := saveAlwaysVal.(string); ok {
						rule.SaveAlways = saveAlwaysStr == "true"
					}
				}
				
				cacheConfig.GoCacheRules = append(cacheConfig.GoCacheRules, rule)
			}

			// Extract Docker caching from docker/build-push-action
			if strings.Contains(step.Uses, "docker/build-push-action") {
				// Extract cache-from configuration
				if cacheFromVal, ok := step.With["cache-from"]; ok {
					if cacheFromStr, ok := cacheFromVal.(string); ok {
						lines := strings.Split(cacheFromStr, "\n")
						for _, line := range lines {
							line = strings.TrimSpace(line)
							if line == "" {
								continue
							}
							
							rule := DockerCacheRule{}
							
							if strings.HasPrefix(line, "type=gha") {
								rule.Type = "gha"
								// Look for scope patterns (including GitHub Actions expressions)
								if strings.Contains(line, "scope=") {
									// Handle both static scopes and GitHub Actions expressions
									if strings.Contains(line, "${{") {
										rule.Scope = "dynamic" // Mark as dynamic scope
									} else {
										scopeRegex := regexp.MustCompile(`scope=([^,\s]+)`)
										if matches := scopeRegex.FindStringSubmatch(line); len(matches) > 1 {
											rule.Scope = matches[1]
										}
									}
								}
								rule.Fallback = strings.Contains(line, "cache_fallback") || strings.Contains(line, "steps.cache-config.outputs.cache_fallback")
							} else if strings.HasPrefix(line, "type=registry") {
								rule.Type = "registry"
								if strings.Contains(line, "ref=") {
									// Handle both static refs and GitHub Actions expressions
									if strings.Contains(line, "${{") {
										rule.Ref = "dynamic" // Mark as dynamic ref
									} else {
										refRegex := regexp.MustCompile(`ref=([^,\s]+)`)
										if matches := refRegex.FindStringSubmatch(line); len(matches) > 1 {
											rule.Ref = matches[1]
										}
									}
								}
								rule.Fallback = strings.Contains(line, "cache_fallback") || strings.Contains(line, "steps.cache-config.outputs.cache_fallback")
							}
							
							cacheConfig.DockerCacheRules = append(cacheConfig.DockerCacheRules, rule)
						}
					}
				}
				
				// Extract cache-to configuration
				if cacheToVal, ok := step.With["cache-to"]; ok {
					if cacheToStr, ok := cacheToVal.(string); ok {
						lines := strings.Split(cacheToStr, "\n")
						for _, line := range lines {
							line = strings.TrimSpace(line)
							if line == "" {
								continue
							}
							
							rule := DockerCacheRule{}
							
							if strings.HasPrefix(line, "type=gha") {
								rule.Type = "gha"
								// Look for scope patterns (including GitHub Actions expressions)
								if strings.Contains(line, "scope=") {
									if strings.Contains(line, "${{") {
										rule.Scope = "dynamic" // Mark as dynamic scope
									} else {
										scopeRegex := regexp.MustCompile(`scope=([^,\s]+)`)
										if matches := scopeRegex.FindStringSubmatch(line); len(matches) > 1 {
											rule.Scope = matches[1]
										}
									}
								}
								if strings.Contains(line, "mode=") {
									if strings.Contains(line, "${{") {
										rule.Mode = "dynamic" // Mark as dynamic mode
									} else {
										modeRegex := regexp.MustCompile(`mode=([^,\s]+)`)
										if matches := modeRegex.FindStringSubmatch(line); len(matches) > 1 {
											rule.Mode = matches[1]
										}
									}
								}
							} else if strings.HasPrefix(line, "type=registry") {
								rule.Type = "registry"
								if strings.Contains(line, "ref=") {
									if strings.Contains(line, "${{") {
										rule.Ref = "dynamic" // Mark as dynamic ref
									} else {
										refRegex := regexp.MustCompile(`ref=([^,\s]+)`)
										if matches := refRegex.FindStringSubmatch(line); len(matches) > 1 {
											rule.Ref = matches[1]
										}
									}
								}
								if strings.Contains(line, "mode=") {
									if strings.Contains(line, "${{") {
										rule.Mode = "dynamic" // Mark as dynamic mode
									} else {
										modeRegex := regexp.MustCompile(`mode=([^,\s]+)`)
										if matches := modeRegex.FindStringSubmatch(line); len(matches) > 1 {
											rule.Mode = matches[1]
										}
									}
								}
							}
							
							cacheConfig.DockerCacheRules = append(cacheConfig.DockerCacheRules, rule)
						}
					}
				}
			}
		}
	}

	return cacheConfig, nil
}

// validateDockerLayerCaching checks if Docker layer caching is properly configured
func validateDockerLayerCaching(event WorkflowEvent, config *CacheConfiguration) bool {
	// Check that Docker layer caching is configured
	hasGHACache := false
	hasScopedCache := false
	
	for _, rule := range config.DockerCacheRules {
		if rule.Type == "gha" {
			hasGHACache = true
			if rule.Scope != "" {
				hasScopedCache = true
			}
		}
	}
	
	// Docker layer caching should use GHA cache with proper scoping
	return hasGHACache && hasScopedCache
}

// validateGoModuleCaching checks if Go module caching is properly configured
func validateGoModuleCaching(event WorkflowEvent, config *CacheConfiguration) bool {
	if len(config.GoCacheRules) == 0 {
		return false // No Go caching configured
	}
	
	for _, rule := range config.GoCacheRules {
		// Check that Go cache paths include both build cache and module cache
		hasBuildCache := false
		hasModCache := false
		
		for _, path := range rule.Path {
			if strings.Contains(path, "go-build") {
				hasBuildCache = true
			}
			if strings.Contains(path, "pkg/mod") {
				hasModCache = true
			}
		}
		
		// Check that cache key is based on go.sum hash
		hasGoSumKey := strings.Contains(rule.Key, "go.sum")
		
		// Check that restore keys provide fallback strategy
		hasRestoreKeys := len(rule.RestoreKeys) > 0
		
		if hasBuildCache && hasModCache && hasGoSumKey && hasRestoreKeys {
			return true
		}
	}
	
	return false
}

// validateCacheScoping checks if cache scoping strategies are appropriate
func validateCacheScoping(event WorkflowEvent, config *CacheConfiguration) bool {
	// For this validation, we check that different event types would result in appropriate cache scoping
	// This is inferred from the cache rule configurations
	
	// Check that cache scopes are event-specific or dynamic
	hasEventSpecificScoping := false
	
	for _, rule := range config.DockerCacheRules {
		if rule.Scope != "" {
			// Cache scoping should differentiate between different workflow contexts
			// Accept both static scopes and dynamic scopes (GitHub Actions expressions)
			if strings.Contains(rule.Scope, "pr-") || 
			   strings.Contains(rule.Scope, "main") || 
			   strings.Contains(rule.Scope, "branch-") ||
			   rule.Scope == "dynamic" { // Dynamic scopes from GitHub Actions expressions
				hasEventSpecificScoping = true
				break
			}
		}
	}
	
	return hasEventSpecificScoping
}

// validateCacheInvalidation checks if cache invalidation strategies are proper
func validateCacheInvalidation(event WorkflowEvent, config *CacheConfiguration) bool {
	// Check that Go module cache uses proper invalidation (go.sum hash)
	hasProperGoInvalidation := false
	
	for _, rule := range config.GoCacheRules {
		if strings.Contains(rule.Key, "hashFiles") && strings.Contains(rule.Key, "go.sum") {
			hasProperGoInvalidation = true
			break
		}
	}
	
	// Check that Docker cache has appropriate mode settings (including dynamic modes)
	hasProperDockerInvalidation := false
	
	for _, rule := range config.DockerCacheRules {
		if rule.Mode == "max" || rule.Mode == "min" || rule.Mode == "dynamic" {
			hasProperDockerInvalidation = true
			break
		}
	}
	
	return hasProperGoInvalidation && hasProperDockerInvalidation
}

// TestCachingBehavior tests Property 8: Caching Behavior
// **Feature: github-actions-docker-publish, Property 8: Caching Behavior**
// **Validates: Requirements 5.1, 5.2, 5.3**
//
// Property: For any workflow execution, Docker build layers and Go module dependencies 
// should be cached when possible, with cache hits occurring for unchanged dependencies
func TestCachingBehavior(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	cacheConfig, err := extractCacheConfiguration(config)
	if err != nil {
		t.Fatalf("Failed to extract cache configuration: %v", err)
	}

	// Property-based test: For any workflow event, caching should be properly configured
	property := func(eventName, refType, refName string, prNumber int) bool {
		// Sanitize inputs to simulate realistic workflow events
		if eventName == "" {
			eventName = "push"
		}
		if refType == "" {
			refType = "branch"
		}
		if refName == "" {
			refName = "main"
		}
		if prNumber < 0 {
			prNumber = 0
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

		event := WorkflowEvent{
			EventName: eventName,
			RefType:   refType,
			RefName:   refName,
			PRNumber:  prNumber,
		}

		// Check all caching requirements
		dockerCacheOK := validateDockerLayerCaching(event, cacheConfig)
		goCacheOK := validateGoModuleCaching(event, cacheConfig)
		cacheScopingOK := validateCacheScoping(event, cacheConfig)
		cacheInvalidationOK := validateCacheInvalidation(event, cacheConfig)

		return dockerCacheOK && goCacheOK && cacheScopingOK && cacheInvalidationOK
	}

	// Run the property test
	if err := quick.Check(property, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Caching behavior property failed: %v", err)
		t.Logf("Docker cache rules: %+v", cacheConfig.DockerCacheRules)
		t.Logf("Go cache rules: %+v", cacheConfig.GoCacheRules)
		t.Logf("Cache scoping rules: %+v", cacheConfig.CacheScoping)
	}
}

// TestDockerLayerCaching validates Docker layer caching configuration
func TestDockerLayerCaching(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	cacheConfig, err := extractCacheConfiguration(config)
	if err != nil {
		t.Fatalf("Failed to extract cache configuration: %v", err)
	}

	// Check that Docker layer caching is configured
	hasDockerCache := len(cacheConfig.DockerCacheRules) > 0
	if !hasDockerCache {
		t.Error("Docker layer caching should be configured")
	}

	// Check that GitHub Actions cache is used
	hasGHACache := false
	for _, rule := range cacheConfig.DockerCacheRules {
		if rule.Type == "gha" {
			hasGHACache = true
			break
		}
	}
	if !hasGHACache {
		t.Error("Docker layer caching should use GitHub Actions cache (type=gha)")
	}

	// Check that cache scoping is configured
	hasScopedCache := false
	for _, rule := range cacheConfig.DockerCacheRules {
		if rule.Scope != "" {
			hasScopedCache = true
			break
		}
	}
	if !hasScopedCache {
		t.Error("Docker layer caching should use scoped caching for better isolation")
	}
}

// TestGoModuleCaching validates Go module dependency caching configuration
func TestGoModuleCaching(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	cacheConfig, err := extractCacheConfiguration(config)
	if err != nil {
		t.Fatalf("Failed to extract cache configuration: %v", err)
	}

	// Check that Go module caching is configured
	hasGoCache := len(cacheConfig.GoCacheRules) > 0
	if !hasGoCache {
		t.Error("Go module caching should be configured")
	}

	for _, rule := range cacheConfig.GoCacheRules {
		// Check that Go build cache is included
		hasBuildCache := false
		for _, path := range rule.Path {
			if strings.Contains(path, "go-build") {
				hasBuildCache = true
				break
			}
		}
		if !hasBuildCache {
			t.Error("Go caching should include build cache (~/.cache/go-build)")
		}

		// Check that Go module cache is included
		hasModCache := false
		for _, path := range rule.Path {
			if strings.Contains(path, "pkg/mod") {
				hasModCache = true
				break
			}
		}
		if !hasModCache {
			t.Error("Go caching should include module cache (~/go/pkg/mod)")
		}

		// Check that cache key is based on go.sum
		if !strings.Contains(rule.Key, "go.sum") {
			t.Error("Go cache key should be based on go.sum hash for proper invalidation")
		}

		// Check that restore keys provide fallback
		if len(rule.RestoreKeys) == 0 {
			t.Error("Go caching should have restore keys for fallback scenarios")
		}
	}
}

// TestCacheInvalidationStrategy validates cache invalidation strategies
func TestCacheInvalidationStrategy(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	cacheConfig, err := extractCacheConfiguration(config)
	if err != nil {
		t.Fatalf("Failed to extract cache configuration: %v", err)
	}

	// Check Go module cache invalidation
	hasGoSumInvalidation := false
	for _, rule := range cacheConfig.GoCacheRules {
		if strings.Contains(rule.Key, "hashFiles") && strings.Contains(rule.Key, "go.sum") {
			hasGoSumInvalidation = true
			break
		}
	}
	if !hasGoSumInvalidation {
		t.Error("Go module cache should use go.sum hash for proper invalidation")
	}

	// Check Docker cache mode configuration (including dynamic modes)
	hasDockerCacheMode := false
	for _, rule := range cacheConfig.DockerCacheRules {
		if rule.Mode == "max" || rule.Mode == "min" || rule.Mode == "dynamic" {
			hasDockerCacheMode = true
			break
		}
	}
	if !hasDockerCacheMode {
		t.Error("Docker cache should have mode configuration for proper cache management")
	}
}

// TestCacheScopingStrategy validates cache scoping strategies
func TestCacheScopingStrategy(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	cacheConfig, err := extractCacheConfiguration(config)
	if err != nil {
		t.Fatalf("Failed to extract cache configuration: %v", err)
	}

	// Check that cache scoping differentiates between contexts
	hasContextualScoping := false
	for _, rule := range cacheConfig.DockerCacheRules {
		if rule.Scope != "" {
			// Look for evidence of contextual scoping (PR, main, branch patterns or dynamic scoping)
			if strings.Contains(rule.Scope, "pr-") || 
			   strings.Contains(rule.Scope, "main") || 
			   strings.Contains(rule.Scope, "branch-") ||
			   rule.Scope == "dynamic" { // Accept dynamic scoping from GitHub Actions expressions
				hasContextualScoping = true
				break
			}
		}
	}
	if !hasContextualScoping {
		t.Error("Docker cache scoping should differentiate between different workflow contexts")
	}
}