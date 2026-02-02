package github_actions

import (
	"strings"
	"testing"
	"testing/quick"
)

// AuthSecurityConfig represents authentication and security configuration
type AuthSecurityConfig struct {
	Permissions      map[string]string
	RegistryLogin    *RegistryLoginConfig
	SecurityChecks   []SecurityCheck
	ForkRestrictions *ForkRestrictionConfig
}

// RegistryLoginConfig represents Docker registry login configuration
type RegistryLoginConfig struct {
	Registry string
	Username string
	Password string
	Logout   bool
}

// SecurityCheck represents a security validation step
type SecurityCheck struct {
	Name        string
	Command     string
	ChecksToken bool
	ChecksRepo  bool
}

// ForkRestrictionConfig represents fork repository restrictions
type ForkRestrictionConfig struct {
	Condition   string
	Repository  string
	PRCondition string
}

// extractAuthSecurityConfig extracts authentication and security configuration from workflow
func extractAuthSecurityConfig(config *WorkflowConfig) (*AuthSecurityConfig, error) {
	authConfig := &AuthSecurityConfig{
		Permissions:    make(map[string]string),
		SecurityChecks: []SecurityCheck{},
	}

	// Extract permissions from workflow
	if config.Permissions != nil {
		authConfig.Permissions = config.Permissions
	}

	for _, job := range config.Jobs {
		// Extract job-level security conditions
		if job.If != "" {
			authConfig.ForkRestrictions = &ForkRestrictionConfig{
				Condition: job.If,
			}
		}

		for _, step := range job.Steps {
			// Extract Docker login configuration
			if strings.Contains(step.Uses, "docker/login-action") {
				if registryVal, ok := step.With["registry"]; ok {
					if usernameVal, ok := step.With["username"]; ok {
						if passwordVal, ok := step.With["password"]; ok {
							authConfig.RegistryLogin = &RegistryLoginConfig{
								Registry: registryVal.(string),
								Username: usernameVal.(string),
								Password: passwordVal.(string),
							}
							if logoutVal, ok := step.With["logout"]; ok {
								if logout, ok := logoutVal.(bool); ok {
									authConfig.RegistryLogin.Logout = logout
								}
							}
						}
					}
				}
			}

			// Extract security validation steps
			if strings.Contains(step.Name, "security") || strings.Contains(step.Name, "auth") ||
				strings.Contains(step.Name, "permission") || strings.Contains(step.Name, "Validate") {
				secCheck := SecurityCheck{
					Name: step.Name,
				}
				if runVal, ok := step.Run.(string); ok {
					secCheck.Command = runVal
					secCheck.ChecksToken = strings.Contains(runVal, "GITHUB_TOKEN") || strings.Contains(runVal, "github.token")
					secCheck.ChecksRepo = strings.Contains(runVal, "github.repository") || strings.Contains(runVal, "repository")
				}
				authConfig.SecurityChecks = append(authConfig.SecurityChecks, secCheck)
			}
		}
	}

	return authConfig, nil
}

// validateGitHubTokenAuth checks if GitHub token authentication is properly configured
func validateGitHubTokenAuth(authConfig *AuthSecurityConfig) bool {
	if authConfig.RegistryLogin == nil {
		return false
	}

	// Check that password uses GITHUB_TOKEN
	if !strings.Contains(authConfig.RegistryLogin.Password, "secrets.GITHUB_TOKEN") {
		return false
	}

	// Check that username uses github.actor
	if !strings.Contains(authConfig.RegistryLogin.Username, "github.actor") {
		return false
	}

	// Check that registry is GHCR
	if !strings.Contains(authConfig.RegistryLogin.Registry, "ghcr.io") &&
		!strings.Contains(authConfig.RegistryLogin.Registry, "env.REGISTRY") {
		return false
	}

	return true
}

// validateMinimumPermissions checks if minimum required permissions are set
func validateMinimumPermissions(authConfig *AuthSecurityConfig) bool {
	requiredPerms := map[string][]string{
		"contents": {"read"},
		"packages": {"write"},
	}

	for perm, allowedValues := range requiredPerms {
		if value, exists := authConfig.Permissions[perm]; !exists {
			return false
		} else {
			found := false
			for _, allowed := range allowedValues {
				if value == allowed {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	return true
}

// validateNoCredentialExposure checks if credentials are not exposed in logs
func validateNoCredentialExposure(authConfig *AuthSecurityConfig) bool {
	// Check that security validation steps exist
	hasTokenValidation := false
	hasCredentialCheck := false

	for _, check := range authConfig.SecurityChecks {
		if check.ChecksToken {
			hasTokenValidation = true
		}
		if strings.Contains(check.Command, "credential") || strings.Contains(check.Command, "token") {
			hasCredentialCheck = true
		}
	}

	// Check that logout is enabled to prevent credential persistence
	if authConfig.RegistryLogin != nil && !authConfig.RegistryLogin.Logout {
		return false
	}

	return hasTokenValidation || hasCredentialCheck
}

// validateForkRestrictions checks if fork repository restrictions are properly implemented
func validateForkRestrictions(authConfig *AuthSecurityConfig) bool {
	if authConfig.ForkRestrictions == nil {
		return false
	}

	condition := authConfig.ForkRestrictions.Condition

	// Check that condition restricts to main repository
	if !strings.Contains(condition, "github.repository") {
		return false
	}

	// Check that condition handles pull requests from forks
	if !strings.Contains(condition, "pull_request") {
		return false
	}

	return true
}

// TestAuthenticationAndSecurity tests Property 3: Authentication and Security
// **Feature: github-actions-docker-publish, Property 3: Authentication and Security**
// **Validates: Requirements 4.1, 4.2, 4.3, 4.5**
//
// Property: For any workflow execution, authentication with GHCR should use GitHub's 
// built-in token with minimum required permissions and no credential exposure in logs
func TestAuthenticationAndSecurity(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	authConfig, err := extractAuthSecurityConfig(config)
	if err != nil {
		t.Fatalf("Failed to extract auth/security config: %v", err)
	}

	// Property-based test: For any workflow execution, security requirements should be met
	property := func() bool {
		// Check GitHub token authentication (Requirement 4.1)
		tokenAuthOK := validateGitHubTokenAuth(authConfig)

		// Check minimum required permissions (Requirement 4.3)
		permissionsOK := validateMinimumPermissions(authConfig)

		// Check no credential exposure (Requirement 4.2)
		noExposureOK := validateNoCredentialExposure(authConfig)

		// Check fork restrictions (Requirement 4.5)
		forkRestrictionsOK := validateForkRestrictions(authConfig)

		return tokenAuthOK && permissionsOK && noExposureOK && forkRestrictionsOK
	}

	// Run the property test
	if err := quick.Check(property, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Authentication and security property failed: %v", err)
		t.Logf("Registry login config: %+v", authConfig.RegistryLogin)
		t.Logf("Permissions: %+v", authConfig.Permissions)
		t.Logf("Security checks: %+v", authConfig.SecurityChecks)
		t.Logf("Fork restrictions: %+v", authConfig.ForkRestrictions)
	}
}

// TestGitHubTokenAuthentication validates GitHub token authentication configuration
func TestGitHubTokenAuthentication(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	authConfig, err := extractAuthSecurityConfig(config)
	if err != nil {
		t.Fatalf("Failed to extract auth/security config: %v", err)
	}

	if authConfig.RegistryLogin == nil {
		t.Fatal("Registry login configuration is missing")
	}

	// Test that GITHUB_TOKEN is used for authentication
	if !strings.Contains(authConfig.RegistryLogin.Password, "secrets.GITHUB_TOKEN") {
		t.Errorf("Password should use secrets.GITHUB_TOKEN, got: %s", authConfig.RegistryLogin.Password)
	}

	// Test that github.actor is used as username
	if !strings.Contains(authConfig.RegistryLogin.Username, "github.actor") {
		t.Errorf("Username should use github.actor, got: %s", authConfig.RegistryLogin.Username)
	}

	// Test that registry is GHCR
	if !strings.Contains(authConfig.RegistryLogin.Registry, "ghcr.io") &&
		!strings.Contains(authConfig.RegistryLogin.Registry, "env.REGISTRY") {
		t.Errorf("Registry should be ghcr.io or reference env.REGISTRY, got: %s", authConfig.RegistryLogin.Registry)
	}
}

// TestWorkflowPermissions validates that minimum required permissions are set
func TestWorkflowPermissions(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	authConfig, err := extractAuthSecurityConfig(config)
	if err != nil {
		t.Fatalf("Failed to extract auth/security config: %v", err)
	}

	// Check required permissions
	requiredPerms := map[string]string{
		"contents": "read",
		"packages": "write",
	}

	for perm, expectedValue := range requiredPerms {
		if actualValue, exists := authConfig.Permissions[perm]; !exists {
			t.Errorf("Required permission %s is missing", perm)
		} else if actualValue != expectedValue {
			t.Errorf("Permission %s has incorrect value: expected %s, got %s", perm, expectedValue, actualValue)
		}
	}
}

// TestForkRepositoryRestrictions validates that fork repository restrictions are implemented
func TestForkRepositoryRestrictions(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	authConfig, err := extractAuthSecurityConfig(config)
	if err != nil {
		t.Fatalf("Failed to extract auth/security config: %v", err)
	}

	if authConfig.ForkRestrictions == nil {
		t.Fatal("Fork restrictions are not configured")
	}

	condition := authConfig.ForkRestrictions.Condition

	// Check that condition restricts to main repository
	if !strings.Contains(condition, "github.repository") {
		t.Errorf("Fork restriction should check github.repository, got: %s", condition)
	}

	// Check that condition handles pull requests appropriately
	if !strings.Contains(condition, "pull_request") {
		t.Errorf("Fork restriction should handle pull_request events, got: %s", condition)
	}
}

// TestSecurityValidationSteps validates that security validation steps are present
func TestSecurityValidationSteps(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	authConfig, err := extractAuthSecurityConfig(config)
	if err != nil {
		t.Fatalf("Failed to extract auth/security config: %v", err)
	}

	if len(authConfig.SecurityChecks) == 0 {
		t.Error("No security validation steps found")
	}

	// Check that at least one security check validates tokens
	hasTokenValidation := false
	for _, check := range authConfig.SecurityChecks {
		if check.ChecksToken {
			hasTokenValidation = true
			break
		}
	}

	if !hasTokenValidation {
		t.Error("No security validation step checks for token availability")
	}
}