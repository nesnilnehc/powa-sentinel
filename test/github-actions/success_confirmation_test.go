package github_actions

import (
	"strings"
	"testing"
	"testing/quick"
)

// SuccessConfirmationConfig represents success confirmation and progress reporting configuration
type SuccessConfirmationConfig struct {
	ProgressIndicators   []ProgressIndicator
	ConfirmationMessages []ConfirmationMessage
	StatusReporting      []StatusReport
	OutputDetails        []OutputDetail
	CompletionSummary    *CompletionSummary
}

// ProgressIndicator represents a step that provides progress information
type ProgressIndicator struct {
	Name           string
	Phase          string // "setup", "build", "publish", "validation"
	ProvidesStatus bool
	ShowsProgress  bool
	HasMetrics     bool
}

// ConfirmationMessage represents success confirmation messaging
type ConfirmationMessage struct {
	Name            string
	MessageType     string // "published", "validated", "completed"
	IncludesDetails bool
	IncludesLinks   bool
	IncludesNextSteps bool
}

// StatusReport represents workflow status reporting
type StatusReport struct {
	Name         string
	ReportsPhase string
	ShowsMetrics bool
	ShowsTimeline bool
}

// OutputDetail represents detailed output information
type OutputDetail struct {
	Name           string
	OutputType     string // "image_details", "build_metrics", "validation_results"
	IncludesImages bool
	IncludesTags   bool
	IncludesCommands bool
}

// CompletionSummary represents the final workflow completion summary
type CompletionSummary struct {
	Name              string
	ShowsOverallStatus bool
	ShowsTimeline     bool
	ShowsMetrics      bool
	ShowsFinalMessage bool
}

// extractSuccessConfirmationConfig extracts success confirmation configuration from workflow
func extractSuccessConfirmationConfig(config *WorkflowConfig) (*SuccessConfirmationConfig, error) {
	successConfig := &SuccessConfirmationConfig{
		ProgressIndicators:   []ProgressIndicator{},
		ConfirmationMessages: []ConfirmationMessage{},
		StatusReporting:      []StatusReport{},
		OutputDetails:        []OutputDetail{},
	}

	for _, job := range config.Jobs {
		for _, step := range job.Steps {
			stepName := step.Name

			// Identify progress indicator steps
			if strings.Contains(strings.ToLower(stepName), "validat") ||
				strings.Contains(strings.ToLower(stepName), "progress") ||
				strings.Contains(strings.ToLower(stepName), "status") {

				phase := "setup"
				if strings.Contains(strings.ToLower(stepName), "build") {
					phase = "build"
				} else if strings.Contains(strings.ToLower(stepName), "publish") {
					phase = "publish"
				} else if strings.Contains(strings.ToLower(stepName), "validat") {
					phase = "validation"
				}

				indicator := ProgressIndicator{
					Name:  stepName,
					Phase: phase,
				}

				if runVal, ok := step.Run.(string); ok {
					indicator.ProvidesStatus = strings.Contains(runVal, "GITHUB_STEP_SUMMARY") ||
						strings.Contains(runVal, "status") ||
						strings.Contains(runVal, "Status")
					indicator.ShowsProgress = strings.Contains(runVal, "progress") ||
						strings.Contains(runVal, "Progress") ||
						strings.Contains(runVal, "✅") ||
						strings.Contains(runVal, "❌")
					indicator.HasMetrics = strings.Contains(runVal, "metrics") ||
						strings.Contains(runVal, "Metrics") ||
						strings.Contains(runVal, "| Property | Value |")
				}

				successConfig.ProgressIndicators = append(successConfig.ProgressIndicators, indicator)
			}

			// Identify confirmation message steps
			if strings.Contains(strings.ToLower(stepName), "output") ||
				strings.Contains(strings.ToLower(stepName), "success") ||
				strings.Contains(strings.ToLower(stepName), "published") ||
				strings.Contains(strings.ToLower(stepName), "completed") ||
				strings.Contains(strings.ToLower(stepName), "details") {

				messageType := "completed"
				if strings.Contains(strings.ToLower(stepName), "published") ||
					strings.Contains(strings.ToLower(stepName), "image details") {
					messageType = "published"
				} else if strings.Contains(strings.ToLower(stepName), "validat") ||
					strings.Contains(strings.ToLower(stepName), "build-only") {
					messageType = "validated"
				}

				confirmation := ConfirmationMessage{
					Name:        stepName,
					MessageType: messageType,
				}

				if runVal, ok := step.Run.(string); ok {
					confirmation.IncludesDetails = strings.Contains(runVal, "details") ||
						strings.Contains(runVal, "Details") ||
						strings.Contains(runVal, "metadata")
					confirmation.IncludesLinks = strings.Contains(runVal, "http") ||
						strings.Contains(runVal, "github.com") ||
						strings.Contains(runVal, "Registry URL")
					confirmation.IncludesNextSteps = strings.Contains(runVal, "next steps") ||
						strings.Contains(runVal, "Next Steps") ||
						strings.Contains(runVal, "1. **")
				}

				successConfig.ConfirmationMessages = append(successConfig.ConfirmationMessages, confirmation)
			}

			// Identify status reporting steps
			if strings.Contains(strings.ToLower(stepName), "summary") ||
				strings.Contains(strings.ToLower(stepName), "completion") ||
				strings.Contains(strings.ToLower(stepName), "workflow") {

				report := StatusReport{
					Name: stepName,
				}

				if runVal, ok := step.Run.(string); ok {
					report.ReportsPhase = "completion"
					if strings.Contains(runVal, "build") {
						report.ReportsPhase = "build"
					} else if strings.Contains(runVal, "publish") {
						report.ReportsPhase = "publish"
					}

					report.ShowsMetrics = strings.Contains(runVal, "metrics") ||
						strings.Contains(runVal, "Metrics") ||
						strings.Contains(runVal, "| Metric | Value |")
					report.ShowsTimeline = strings.Contains(runVal, "timeline") ||
						strings.Contains(runVal, "Timeline") ||
						strings.Contains(runVal, "| Phase | Status |")
				}

				successConfig.StatusReporting = append(successConfig.StatusReporting, report)

				// Check if this is the completion summary
				if strings.Contains(strings.ToLower(stepName), "completion") &&
					strings.Contains(strings.ToLower(stepName), "summary") {
					if runVal, ok := step.Run.(string); ok {
						successConfig.CompletionSummary = &CompletionSummary{
							Name:              stepName,
							ShowsOverallStatus: strings.Contains(runVal, "Overall Status") ||
								strings.Contains(runVal, "WORKFLOW_STATUS"),
							ShowsTimeline: strings.Contains(runVal, "Timeline") ||
								strings.Contains(runVal, "| Phase | Status |"),
							ShowsMetrics: strings.Contains(runVal, "Metrics") ||
								strings.Contains(runVal, "| Metric | Value |"),
							ShowsFinalMessage: strings.Contains(runVal, "completed successfully") ||
								strings.Contains(runVal, "ready for use"),
						}
					}
				}
			}

			// Identify detailed output steps
			if strings.Contains(strings.ToLower(stepName), "image") &&
				strings.Contains(strings.ToLower(stepName), "details") {

				detail := OutputDetail{
					Name:       stepName,
					OutputType: "image_details",
				}

				if runVal, ok := step.Run.(string); ok {
					detail.IncludesImages = strings.Contains(runVal, "docker pull") ||
						strings.Contains(runVal, "Published Images")
					detail.IncludesTags = strings.Contains(runVal, "tags") ||
						strings.Contains(runVal, "Tags") ||
						strings.Contains(runVal, "steps.meta.outputs.tags")
					detail.IncludesCommands = strings.Contains(runVal, "docker pull") ||
						strings.Contains(runVal, "docker run") ||
						strings.Contains(runVal, "```bash")
				}

				successConfig.OutputDetails = append(successConfig.OutputDetails, detail)
			}
		}
	}

	return successConfig, nil
}

// validateProgressIndicators checks if progress indicators are provided throughout the workflow
func validateProgressIndicators(successConfig *SuccessConfirmationConfig) bool {
	if len(successConfig.ProgressIndicators) == 0 {
		return false
	}

	// Check that progress indicators cover different phases
	phases := make(map[string]bool)
	hasStatusReporting := false
	hasProgressDisplay := false

	for _, indicator := range successConfig.ProgressIndicators {
		phases[indicator.Phase] = true
		if indicator.ProvidesStatus {
			hasStatusReporting = true
		}
		if indicator.ShowsProgress {
			hasProgressDisplay = true
		}
	}

	// Should have indicators for multiple phases and provide status/progress
	return len(phases) >= 2 && hasStatusReporting && hasProgressDisplay
}

// validateConfirmationMessages checks if confirmation messages are provided for successful operations
func validateConfirmationMessages(successConfig *SuccessConfirmationConfig) bool {
	if len(successConfig.ConfirmationMessages) == 0 {
		return false
	}

	// Check that confirmation messages provide comprehensive information
	hasPublishedConfirmation := false
	hasDetailedInfo := false
	hasNextSteps := false
	hasLinks := false

	for _, confirmation := range successConfig.ConfirmationMessages {
		if confirmation.MessageType == "published" || confirmation.MessageType == "completed" {
			hasPublishedConfirmation = true
		}
		if confirmation.IncludesDetails {
			hasDetailedInfo = true
		}
		if confirmation.IncludesNextSteps {
			hasNextSteps = true
		}
		if confirmation.IncludesLinks {
			hasLinks = true
		}
	}

	// At least one confirmation message should have detailed info, and at least one should have links or next steps
	return hasPublishedConfirmation && hasDetailedInfo && (hasNextSteps || hasLinks)
}

// validateStatusReporting checks if workflow status reporting is comprehensive
func validateStatusReporting(successConfig *SuccessConfirmationConfig) bool {
	if len(successConfig.StatusReporting) == 0 {
		return false
	}

	// Check that status reporting includes metrics and timeline
	hasMetrics := false
	hasTimeline := false
	hasCompletionSummary := successConfig.CompletionSummary != nil

	for _, report := range successConfig.StatusReporting {
		if report.ShowsMetrics {
			hasMetrics = true
		}
		if report.ShowsTimeline {
			hasTimeline = true
		}
	}

	return hasMetrics && hasTimeline && hasCompletionSummary
}

// validateOutputDetails checks if detailed output information is provided
func validateOutputDetails(successConfig *SuccessConfirmationConfig) bool {
	if len(successConfig.OutputDetails) == 0 {
		return false
	}

	// Check that output details include comprehensive information
	hasImageDetails := false
	hasTags := false
	hasCommands := false

	for _, detail := range successConfig.OutputDetails {
		if detail.OutputType == "image_details" {
			hasImageDetails = true
		}
		if detail.IncludesTags {
			hasTags = true
		}
		if detail.IncludesCommands {
			hasCommands = true
		}
	}

	return hasImageDetails && hasTags && hasCommands
}

// TestSuccessConfirmation tests Property 10: Success Confirmation
// **Feature: github-actions-docker-publish, Property 10: Success Confirmation**
// **Validates: Requirements 5.5, 6.5**
//
// Property: For any successful build and publish operation, the workflow should provide 
// clear confirmation messages with published image locations and progress indicators
func TestSuccessConfirmation(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	successConfig, err := extractSuccessConfirmationConfig(config)
	if err != nil {
		t.Fatalf("Failed to extract success confirmation config: %v", err)
	}

	// Property-based test: For any successful workflow execution, confirmation should be comprehensive
	property := func() bool {
		// Check progress indicators throughout workflow (Requirement 5.5)
		progressOK := validateProgressIndicators(successConfig)

		// Check confirmation messages with published image locations (Requirement 6.5)
		confirmationOK := validateConfirmationMessages(successConfig)

		// Check workflow status reporting (Requirement 5.5)
		statusReportingOK := validateStatusReporting(successConfig)

		// Check detailed output information (Requirement 6.5)
		outputDetailsOK := validateOutputDetails(successConfig)

		return progressOK && confirmationOK && statusReportingOK && outputDetailsOK
	}

	// Run the property test
	if err := quick.Check(property, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Success confirmation property failed: %v", err)
		t.Logf("Progress indicators: %+v", successConfig.ProgressIndicators)
		t.Logf("Confirmation messages: %+v", successConfig.ConfirmationMessages)
		t.Logf("Status reporting: %+v", successConfig.StatusReporting)
		t.Logf("Output details: %+v", successConfig.OutputDetails)
		t.Logf("Completion summary: %+v", successConfig.CompletionSummary)
	}
}

// TestProgressIndicators validates that progress indicators are provided throughout the workflow
func TestProgressIndicators(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	successConfig, err := extractSuccessConfirmationConfig(config)
	if err != nil {
		t.Fatalf("Failed to extract success confirmation config: %v", err)
	}

	if len(successConfig.ProgressIndicators) == 0 {
		t.Error("Workflow should have progress indicators")
	}

	// Check that progress indicators cover different phases
	phases := make(map[string]bool)
	for _, indicator := range successConfig.ProgressIndicators {
		phases[indicator.Phase] = true
	}

	if len(phases) < 2 {
		t.Errorf("Progress indicators should cover multiple workflow phases, found: %v", phases)
	}
}

// TestConfirmationMessages validates that confirmation messages are provided for successful operations
func TestConfirmationMessages(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	successConfig, err := extractSuccessConfirmationConfig(config)
	if err != nil {
		t.Fatalf("Failed to extract success confirmation config: %v", err)
	}

	if len(successConfig.ConfirmationMessages) == 0 {
		t.Error("Workflow should have confirmation messages for successful operations")
	}

	// Check that confirmation messages include essential information
	hasDetailedInfo := false
	hasLinks := false

	for _, confirmation := range successConfig.ConfirmationMessages {
		if confirmation.IncludesDetails {
			hasDetailedInfo = true
		}
		if confirmation.IncludesLinks {
			hasLinks = true
		}
	}

	if !hasDetailedInfo {
		t.Error("Confirmation messages should include detailed information")
	}
	if !hasLinks {
		t.Error("Confirmation messages should include relevant links")
	}
}

// TestStatusReporting validates that workflow status reporting is comprehensive
func TestStatusReporting(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	successConfig, err := extractSuccessConfirmationConfig(config)
	if err != nil {
		t.Fatalf("Failed to extract success confirmation config: %v", err)
	}

	if len(successConfig.StatusReporting) == 0 {
		t.Error("Workflow should have status reporting")
	}

	// Check that status reporting includes metrics and timeline
	hasMetrics := false
	hasTimeline := false

	for _, report := range successConfig.StatusReporting {
		if report.ShowsMetrics {
			hasMetrics = true
		}
		if report.ShowsTimeline {
			hasTimeline = true
		}
	}

	if !hasMetrics {
		t.Error("Status reporting should include metrics")
	}
	if !hasTimeline {
		t.Error("Status reporting should include timeline information")
	}
}

// TestCompletionSummary validates that a comprehensive completion summary is provided
func TestCompletionSummary(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	successConfig, err := extractSuccessConfirmationConfig(config)
	if err != nil {
		t.Fatalf("Failed to extract success confirmation config: %v", err)
	}

	if successConfig.CompletionSummary == nil {
		t.Error("Workflow should have a completion summary")
		return
	}

	summary := successConfig.CompletionSummary

	if !summary.ShowsOverallStatus {
		t.Error("Completion summary should show overall workflow status")
	}
	if !summary.ShowsTimeline {
		t.Error("Completion summary should show execution timeline")
	}
	if !summary.ShowsMetrics {
		t.Error("Completion summary should show workflow metrics")
	}
	if !summary.ShowsFinalMessage {
		t.Error("Completion summary should show final status message")
	}
}

// TestOutputDetails validates that detailed output information is provided
func TestOutputDetails(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	successConfig, err := extractSuccessConfirmationConfig(config)
	if err != nil {
		t.Fatalf("Failed to extract success confirmation config: %v", err)
	}

	if len(successConfig.OutputDetails) == 0 {
		t.Error("Workflow should provide detailed output information")
	}

	// Check that output details include comprehensive information
	hasImageDetails := false
	hasTags := false
	hasCommands := false

	for _, detail := range successConfig.OutputDetails {
		if detail.OutputType == "image_details" {
			hasImageDetails = true
		}
		if detail.IncludesTags {
			hasTags = true
		}
		if detail.IncludesCommands {
			hasCommands = true
		}
	}

	if !hasImageDetails {
		t.Error("Output details should include image information")
	}
	if !hasTags {
		t.Error("Output details should include image tags")
	}
	if !hasCommands {
		t.Error("Output details should include usage commands")
	}
}

// TestComprehensiveSuccessReporting validates that all success reporting requirements are met
func TestComprehensiveSuccessReporting(t *testing.T) {
	config, err := parseWorkflowFile()
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	successConfig, err := extractSuccessConfirmationConfig(config)
	if err != nil {
		t.Fatalf("Failed to extract success confirmation config: %v", err)
	}

	// Verify all success reporting components are present
	if len(successConfig.ProgressIndicators) == 0 {
		t.Error("Workflow must have progress indicators")
	}

	if len(successConfig.ConfirmationMessages) == 0 {
		t.Error("Workflow must have confirmation messages")
	}

	if len(successConfig.StatusReporting) == 0 {
		t.Error("Workflow must have status reporting")
	}

	if len(successConfig.OutputDetails) == 0 {
		t.Error("Workflow must have detailed output information")
	}

	if successConfig.CompletionSummary == nil {
		t.Error("Workflow must have a completion summary")
	}

	// Verify success reporting covers critical workflow outcomes
	outcomes := []string{"published", "validated", "completed"}
	coveredOutcomes := make(map[string]bool)

	for _, confirmation := range successConfig.ConfirmationMessages {
		for _, outcome := range outcomes {
			if confirmation.MessageType == outcome {
				coveredOutcomes[outcome] = true
			}
		}
	}

	if len(coveredOutcomes) < 2 {
		t.Errorf("Success reporting should cover critical workflow outcomes, covered: %v", coveredOutcomes)
	}
}