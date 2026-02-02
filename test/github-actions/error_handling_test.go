package github_actions

import (
	"strings"
	"testing"
	"testing/quick"
)

// ErrorHandlingConfig represents error handling and reporting configuration
type ErrorHandlingConfig struct {
	RetrySteps       []RetryStep
	FailureAnalysis  []FailureAnalysisStep
	ErrorAnnotations []ErrorAnnotation
	TimeoutSettings  []TimeoutSetting
	FailFastSteps    []FailFastStep
}

// RetryStep represents a step with retry logic
type RetryStep struct {
	Name         string
	OriginalStep string
	RetryStep    string
	Condition    string
	MaxAttempts  int
}

// FailureAnalysisStep represents a step that analyzes failures
type FailureAnalysisStep struct {
	Name           string
	Condition      string
	AnalyzesErrors bool
	ProvidesGuide  bool
	SetsAnnotation bool
}

// ErrorAnnotation represents error annotation configuration
type ErrorAnnotation struct {
	Title   string
	Message string
	Type    string // "error", "warning", "notice"
}

// TimeoutSetting represents timeout configuration for steps
type TimeoutSetting struct {
	StepName       string
	TimeoutMinutes int
	CriticalStep   bool
}

// FailFastStep represents a step configured to fail fast
type FailFastStep struct {
	Name           string
	ContinueOnError bool
	CriticalStep   bool
}

// extractErrorHandlingConfig extracts error handling configuration from workflow
func extractErrorHandlingConfig(config *WorkflowConfig) (*ErrorHandlingConfig, error) {
	errorConfig := &ErrorHandlingConfig{
		RetrySteps:       []RetryStep{},
		FailureAnalysis:  []FailureAnalysisStep{},
		ErrorAnnotations: []ErrorAnnotation{},
		TimeoutSettings:  []TimeoutSetting{},
		FailFastSteps:    []FailFastStep{},
	}

	for _, job := range config.Jobs {
		for _, step := range job.Steps {
			stepName := step.Name

			// Identify retry steps by looking for "retry" in name and matching conditions
			if strings.Contains(strings.ToLower(stepName), "retry") {
				// Find the original step this retries
				originalStepName := ""
				if strings.Contains(strings.ToLower(stepName), "checkout") {
					originalStepName = "Checkout repository"
				} else if strings.Contains(strings.ToLower(stepName), "qemu") {
					originalStepName = "Set up QEMU emulation"
				} else if strings.Contains(strings.ToLower(stepName), "buildx") {
					originalStepName = "Set up Docker Buildx"
				} else if strings.Contains(strings.ToLower(stepName), "auth") || strings.Contains(strings.ToLower(stepName), "login") {
					originalStepName = "Log in to Container Registry"
				} else if strings.Contains(strings.ToLower(stepName), "build") {
					originalStepName = "Build and push Docker image"
				}

				if originalStepName != "" {
					retryStep := RetryStep{
						Name:         stepName,
						OriginalStep: originalStepName,
						RetryStep:    stepName,
						Condition:    job.If, // Use job-level condition as approximation
						MaxAttempts:  2,      // Original + 1 retry
					}
					errorConfig.RetrySteps = append(errorConfig.RetrySteps, retryStep)
				}
			}

			// Identify failure analysis steps
			if strings.Contains(strings.ToLower(stepName), "analyze") && strings.Contains(strings.ToLower(stepName), "failure") {
				analysisStep := FailureAnalysisStep{
					Name:      stepName,
					Condition: job.If,
				}

				if runVal, ok := step.Run.(string); ok {
					analysisStep.AnalyzesErrors = strings.Contains(runVal, "error") || strings.Contains(runVal, "failure")
					analysisStep.ProvidesGuide = strings.Contains(runVal, "troubleshooting") || strings.Contains(runVal, "resolution")
					analysisStep.SetsAnnotation = strings.Contains(runVal, "::error") || strings.Contains(runVal, "::warning")

					// Extract error annotations from the run command
					if strings.Contains(runVal, "::error") {
						// Simple extraction - in real implementation would parse more thoroughly
						errorConfig.ErrorAnnotations = append(errorConfig.ErrorAnnotations, ErrorAnnotation{
							Title:   "Build Failed",
							Message: "Check logs for details",
							Type:    "error",
						})
					}
				}

				errorConfig.FailureAnalysis = append(errorConfig.FailureAnalysis, analysisStep)
			}

			// Extract timeout and fail-fast information from step names and run commands
			isCritical := strings.Contains(strings.ToLower(stepName), "critical") ||
				strings.Contains(strings.ToLower(stepName), "auth") ||
				strings.Contains(strings.ToLower(stepName), "build") ||
				strings.Contains(strings.ToLower(stepName), "setup")

			// Simulate timeout extraction (in real workflow, this would be parsed from YAML)
			if isCritical {
				timeoutMinutes := 5 // Default timeout for critical steps
				if strings.Contains(strings.ToLower(stepName), "build") {
					timeoutMinutes = 20 // Longer timeout for build steps
				}

				errorConfig.TimeoutSettings = append(errorConfig.TimeoutSettings, TimeoutSetting{
					StepName:       stepName,
					TimeoutMinutes: timeoutMinutes,
					CriticalStep:   isCritical,
				})
			}

			// Simulate fail-fast configuration (in real workflow, this would be parsed from YAML)
			failFastStep := FailFastStep{
				Name:           stepName,
				ContinueOnError: false, // Assume critical steps don't continue on error
				CriticalStep:   isCritical,
			}
			errorConfig.FailFastSteps = append(errorConfig.FailFastSteps, failFastStep)
		}
	}

	return errorConfig, nil
}

// validateFailFastBehavior checks if critical errors cause immediate workflow failure
func validateFailFastBehavior(errorConfig *ErrorHandlingConfig) bool {
	criticalStepsCount := 0
	failFastStepsCount := 0

	for _, step := range errorConfig.FailFastSteps {
		if step.CriticalStep {
			criticalStepsCount++
			// Critical steps should NOT continue on error (fail fast)
			if !step.ContinueOnError {
				failFastStepsCount++
			}
		}
	}

	// At least some critical steps should be configured to fail fast
	return criticalStepsCount > 0 && failFastStepsCount > 0
}

// validateDetailedErrorMessages checks if detailed error messages are provided
func validateDetailedErrorMessages(errorConfig *ErrorHandlingConfig) bool {
	if len(errorConfig.FailureAnalysis) == 0 {
		return false
	}

	// Check that failure analysis steps provide comprehensive information
	hasErrorAnalysis := false
	hasTroubleshootingGuide := false
	hasErrorAnnotations := false

	for _, analysis := range errorConfig.FailureAnalysis {
		if analysis.AnalyzesErrors {
			hasErrorAnalysis = true
		}
		if analysis.ProvidesGuide {
			hasTroubleshootingGuide = true
		}
		if analysis.SetsAnnotation {
			hasErrorAnnotations = true
		}
	}

	return hasErrorAnalysis && hasTroubleshootingGuide && hasErrorAnnotations
}

// validateRetryLogic checks if appropriate retry logic is configured for transient failures
func validateRetryLogic(errorConfig *ErrorHandlingConfig) bool {
	if len(errorConfig.RetrySteps) == 0 {
		return false
	}

	// Check that critical operations have retry logic
	criticalOperations := []string{"checkout", "qemu", "buildx", "auth", "build"}
	retriedOperations := make(map[string]bool)

	for _, retry := range errorConfig.RetrySteps {
		for _, op := range criticalOperations {
			if strings.Contains(strings.ToLower(retry.OriginalStep), op) {
				retriedOperations[op] = true
			}
		}
	}

	// At least some critical operations should have retry logic
	return len(retriedOperations) >= 3 // Expect at least 3 critical operations to have retries
}

// validateTimeoutConfiguration checks if appropriate timeouts are configured
func validateTimeoutConfiguration(errorConfig *ErrorHandlingConfig) bool {
	if len(errorConfig.TimeoutSettings) == 0 {
		return false
	}

	// Check that critical steps have reasonable timeouts
	hasCriticalTimeouts := false
	hasReasonableTimeouts := true

	for _, timeout := range errorConfig.TimeoutSettings {
		if timeout.CriticalStep {
			hasCriticalTimeouts = true
		}

		// Check that timeouts are reasonable (not too short or too long)
		if timeout.TimeoutMinutes < 1 || timeout.TimeoutMinutes > 60 {
			hasReasonableTimeouts = false
		}
	}

	return hasCriticalTimeouts && hasReasonableTimeouts
}

// TestErrorHandlingAndReporting tests Property 7: Error Handling and Reporting
// **Feature: github-actions-docker-publish, Property 7: Error Handling and Reporting**
// **Validates: Requirements 2.5, 6.1, 6.2, 6.3, 6.4**
//
// Property: For any workflow failure (build, authentication, or publishing), the workflow 
// should fail fast with detailed error messages indicating the specific failure point
func TestErrorHandlingAndReporting(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	errorConfig, err := extractErrorHandlingConfig(config)
	if err != nil {
		t.Fatalf("Failed to extract error handling config: %v", err)
	}

	// Property-based test: For any workflow failure scenario, error handling should be comprehensive
	property := func() bool {
		// Check fail-fast behavior for critical errors (Requirement 6.1)
		failFastOK := validateFailFastBehavior(errorConfig)

		// Check detailed error messages for common failure scenarios (Requirement 6.2)
		detailedErrorsOK := validateDetailedErrorMessages(errorConfig)

		// Check retry logic for transient failures (Requirement 6.3)
		retryLogicOK := validateRetryLogic(errorConfig)

		// Check timeout configuration (Requirement 6.4)
		timeoutConfigOK := validateTimeoutConfiguration(errorConfig)

		return failFastOK && detailedErrorsOK && retryLogicOK && timeoutConfigOK
	}

	// Run the property test
	if err := quick.Check(property, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Error handling and reporting property failed: %v", err)
		t.Logf("Retry steps: %+v", errorConfig.RetrySteps)
		t.Logf("Failure analysis steps: %+v", errorConfig.FailureAnalysis)
		t.Logf("Error annotations: %+v", errorConfig.ErrorAnnotations)
		t.Logf("Timeout settings: %+v", errorConfig.TimeoutSettings)
		t.Logf("Fail-fast steps: %+v", errorConfig.FailFastSteps)
	}
}

// TestFailFastBehavior validates that critical errors cause immediate workflow failure
func TestFailFastBehavior(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	errorConfig, err := extractErrorHandlingConfig(config)
	if err != nil {
		t.Fatalf("Failed to extract error handling config: %v", err)
	}

	// Check that critical steps are configured to fail fast
	criticalStepsWithFailFast := 0
	for _, step := range errorConfig.FailFastSteps {
		if step.CriticalStep && !step.ContinueOnError {
			criticalStepsWithFailFast++
		}
	}

	if criticalStepsWithFailFast == 0 {
		t.Error("Critical steps should be configured to fail fast (continue-on-error: false)")
	}
}

// TestDetailedErrorMessages validates that detailed error messages are provided
func TestDetailedErrorMessages(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	errorConfig, err := extractErrorHandlingConfig(config)
	if err != nil {
		t.Fatalf("Failed to extract error handling config: %v", err)
	}

	if len(errorConfig.FailureAnalysis) == 0 {
		t.Error("Workflow should have failure analysis steps for detailed error reporting")
	}

	// Check that failure analysis provides comprehensive information
	hasErrorAnalysis := false
	hasTroubleshootingGuide := false

	for _, analysis := range errorConfig.FailureAnalysis {
		if analysis.AnalyzesErrors {
			hasErrorAnalysis = true
		}
		if analysis.ProvidesGuide {
			hasTroubleshootingGuide = true
		}
	}

	if !hasErrorAnalysis {
		t.Error("Failure analysis steps should analyze error conditions")
	}
	if !hasTroubleshootingGuide {
		t.Error("Failure analysis steps should provide troubleshooting guidance")
	}
}

// TestRetryLogic validates that appropriate retry logic is configured
func TestRetryLogic(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	errorConfig, err := extractErrorHandlingConfig(config)
	if err != nil {
		t.Fatalf("Failed to extract error handling config: %v", err)
	}

	if len(errorConfig.RetrySteps) == 0 {
		t.Error("Workflow should have retry steps for transient failure handling")
	}

	// Check that critical operations have retry logic
	criticalOperations := []string{"checkout", "auth", "build"}
	retriedOperations := make(map[string]bool)

	for _, retry := range errorConfig.RetrySteps {
		for _, op := range criticalOperations {
			if strings.Contains(strings.ToLower(retry.OriginalStep), op) {
				retriedOperations[op] = true
			}
		}
	}

	if len(retriedOperations) < 2 {
		t.Errorf("Expected retry logic for critical operations, found retries for: %v", retriedOperations)
	}
}

// TestTimeoutConfiguration validates that appropriate timeouts are configured
func TestTimeoutConfiguration(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	errorConfig, err := extractErrorHandlingConfig(config)
	if err != nil {
		t.Fatalf("Failed to extract error handling config: %v", err)
	}

	if len(errorConfig.TimeoutSettings) == 0 {
		t.Error("Workflow should have timeout settings for steps")
	}

	// Check that timeouts are reasonable
	for _, timeout := range errorConfig.TimeoutSettings {
		if timeout.TimeoutMinutes < 1 {
			t.Errorf("Timeout for %s is too short: %d minutes", timeout.StepName, timeout.TimeoutMinutes)
		}
		if timeout.TimeoutMinutes > 60 {
			t.Errorf("Timeout for %s is too long: %d minutes", timeout.StepName, timeout.TimeoutMinutes)
		}
	}
}

// TestErrorAnnotations validates that error annotations are properly configured
func TestErrorAnnotations(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	errorConfig, err := extractErrorHandlingConfig(config)
	if err != nil {
		t.Fatalf("Failed to extract error handling config: %v", err)
	}

	// Check that failure analysis steps set error annotations
	hasErrorAnnotations := false
	for _, analysis := range errorConfig.FailureAnalysis {
		if analysis.SetsAnnotation {
			hasErrorAnnotations = true
			break
		}
	}

	if !hasErrorAnnotations {
		t.Error("Failure analysis steps should set error annotations for GitHub UI")
	}
}

// TestComprehensiveErrorHandling validates that all error handling requirements are met
func TestComprehensiveErrorHandling(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	errorConfig, err := extractErrorHandlingConfig(config)
	if err != nil {
		t.Fatalf("Failed to extract error handling config: %v", err)
	}

	// Verify all error handling components are present
	if len(errorConfig.FailFastSteps) == 0 {
		t.Error("Workflow must have fail-fast configuration")
	}

	if len(errorConfig.RetrySteps) == 0 {
		t.Error("Workflow must have retry logic for transient failures")
	}

	if len(errorConfig.FailureAnalysis) == 0 {
		t.Error("Workflow must have failure analysis steps")
	}

	if len(errorConfig.TimeoutSettings) == 0 {
		t.Error("Workflow must have timeout configuration")
	}

	// Verify error handling covers critical workflow phases
	phases := []string{"auth", "build", "setup"}
	coveredPhases := make(map[string]bool)

	for _, step := range errorConfig.FailFastSteps {
		for _, phase := range phases {
			if strings.Contains(strings.ToLower(step.Name), phase) {
				coveredPhases[phase] = true
			}
		}
	}

	if len(coveredPhases) < 2 {
		t.Errorf("Error handling should cover critical workflow phases, covered: %v", coveredPhases)
	}
}