package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/powa-team/powa-sentinel/internal/config"
	"github.com/powa-team/powa-sentinel/internal/model"
)

// WeComNotifier sends alerts to WeCom (WeChat Work) via webhook.
type WeComNotifier struct {
	webhookURL string
	retries    int
	retryDelay time.Duration
	client     *http.Client
}

// wecomMessage represents the WeCom webhook message format.
type wecomMessage struct {
	MsgType  string           `json:"msgtype"`
	Markdown *markdownContent `json:"markdown,omitempty"`
	Text     *textContent     `json:"text,omitempty"`
}

type markdownContent struct {
	Content string `json:"content"`
}

type textContent struct {
	Content string `json:"content"`
}

// wecomResponse represents the WeCom API response.
type wecomResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

// NewWeComNotifier creates a new WeCom notifier.
func NewWeComNotifier(cfg *config.NotifierConfig) (*WeComNotifier, error) {
	retryDelay, err := cfg.RetryDelayParsed()
	if err != nil {
		retryDelay = time.Second
	}

	return &WeComNotifier{
		webhookURL: cfg.WebhookURL,
		retries:    cfg.Retries,
		retryDelay: retryDelay,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// Name returns the notifier name.
func (w *WeComNotifier) Name() string {
	return "wecom"
}

// Send sends the alert to WeCom.
func (w *WeComNotifier) Send(ctx context.Context, alert *model.AlertContext) error {
	content := w.formatMessage(alert)

	// Split message if it exceeds WeCom limit (4096 bytes)
	chunks := splitMessage(content, 4096)

	for i, chunk := range chunks {
		// Add pagination indicator if multiple chunks
		if len(chunks) > 1 {
			chunk += fmt.Sprintf("\n\n*(Part %d/%d)*", i+1, len(chunks))
		}

		msg := wecomMessage{
			MsgType: "markdown",
			Markdown: &markdownContent{
				Content: chunk,
			},
		}

		if err := w.sendWithRetry(ctx, msg); err != nil {
			return fmt.Errorf("failed to send chunk %d: %w", i+1, err)
		}
	}

	return nil
}

// formatMessage creates a markdown message from the alert context.
func (w *WeComNotifier) formatMessage(alert *model.AlertContext) string {
	var sb strings.Builder

	// Header with health status
	statusEmoji := getStatusEmoji(alert.Summary.HealthStatus)
	sb.WriteString(fmt.Sprintf("## %s PoWA Sentinel Report\n\n", statusEmoji))

	// Summary section (L1 - Management level)
	sb.WriteString("### 📊 Summary\n")
	sb.WriteString(fmt.Sprintf("> **Health Score**: %d/100 (%s)\n",
		alert.Summary.HealthScore, alert.Summary.HealthStatus))
	sb.WriteString(fmt.Sprintf("> **Analysis Period**: %s ~ %s\n",
		alert.AnalysisWindow.Start.Format("2006-01-02 15:04"),
		alert.AnalysisWindow.End.Format("2006-01-02 15:04")))
	sb.WriteString(fmt.Sprintf("> **Queries Analyzed**: %d\n\n",
		alert.Summary.TotalQueriesAnalyzed))

	// Issues summary
	if alert.Summary.RegressionCount > 0 || alert.Summary.SuggestionCount > 0 {
		sb.WriteString("**Issues Found**:\n")
		if alert.Summary.RegressionCount > 0 {
			sb.WriteString(fmt.Sprintf("- 🔴 %d Performance Regressions\n", alert.Summary.RegressionCount))
		}
		if alert.Summary.SuggestionCount > 0 {
			sb.WriteString(fmt.Sprintf("- 💡 %d Index Suggestions\n", alert.Summary.SuggestionCount))
		}
		sb.WriteString("\n")
	}

	// Slow SQL section (L2 - Tech Lead level)
	if len(alert.TopSlowSQL) > 0 {
		sb.WriteString("### ⏱ Top Slow Queries\n")
		for i, q := range alert.TopSlowSQL {
			if i >= 5 { // Limit to top 5 in message
				sb.WriteString(fmt.Sprintf("... and %d more\n", len(alert.TopSlowSQL)-5))
				break
			}
			serverInfo := q.DatabaseName
			if q.ServerName != "" && q.ServerName != "local" {
				serverInfo = fmt.Sprintf("%s/%s", q.ServerName, q.DatabaseName)
			}
			sb.WriteString(fmt.Sprintf("**%d. [%s] Query ID**: `%d`\n", i+1, serverInfo, q.QueryID))
			sb.WriteString(fmt.Sprintf("   - Total Time: %.2fms | Calls: %d\n", q.TotalTime, q.Calls))
			// Truncate query for readability and use code block
			queryPreview := truncateQuery(q.Query, 300)
			sb.WriteString(fmt.Sprintf("```sql\n%s\n```\n", queryPreview))
		}
		sb.WriteString("\n")
	}

	// Regressions section (L2/L3 level)
	if len(alert.Regressions) > 0 {
		sb.WriteString("### 📈 Performance Regressions\n")
		for i, r := range alert.Regressions {
			if i >= 10 { // Limit to top 10 in message (increased from 3 as requested)
				sb.WriteString(fmt.Sprintf("... and %d more\n", len(alert.Regressions)-10))
				break
			}
			serverInfo := r.DatabaseName
			if r.ServerName != "" && r.ServerName != "local" {
				serverInfo = fmt.Sprintf("%s/%s", r.ServerName, r.DatabaseName)
			}
			severityIcon := getSeverityIcon(r.Severity)
			sb.WriteString(fmt.Sprintf("%s **[%s] Query ID**: `%d` (%s)\n", severityIcon, serverInfo, r.QueryID, r.Severity))
			sb.WriteString(fmt.Sprintf("   - Mean Time: %.2fms → %.2fms (**+%.1f%%**)\n",
				r.BaselineMeanTime, r.CurrentMeanTime, r.ChangePercent))
			// Truncate query for readability and use code block
			queryPreview := truncateQuery(r.Query, 300)
			sb.WriteString(fmt.Sprintf("```sql\n%s\n```\n", queryPreview))
		}
		sb.WriteString("\n")
	}

	// Index suggestions section (L3 - DBA level)
	if len(alert.Suggestions) > 0 {
		sb.WriteString("### 💡 Index Suggestions\n")
		for i, s := range alert.Suggestions {
			if i >= 3 { // Limit to top 3 in message
				sb.WriteString(fmt.Sprintf("... and %d more\n", len(alert.Suggestions)-3))
				break
			}
			sb.WriteString(fmt.Sprintf("**%d. %s** (Est. +%.0f%%)\n",
				i+1, s.FullTableName(), s.EstImprovementPercent))
			sb.WriteString(fmt.Sprintf("   - Columns: `%s`\n", strings.Join(s.Columns, ", ")))
		}
		sb.WriteString("\n")
	}

	// Footer
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("*Report ID: %s*\n", alert.ReqID))

	return sb.String()
}

// sendWithRetry sends the message with exponential backoff retry.
func (w *WeComNotifier) sendWithRetry(ctx context.Context, msg wecomMessage) error {
	var lastErr error
	delay := w.retryDelay

	for attempt := 0; attempt <= w.retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				delay *= 2 // Exponential backoff
			}
		}

		err := w.send(ctx, msg)
		if err == nil {
			return nil
		}
		lastErr = err
	}

	return fmt.Errorf("failed after %d retries: %w", w.retries, lastErr)
}

// send performs the actual HTTP request to WeCom.
func (w *WeComNotifier) send(ctx context.Context, msg wecomMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshaling message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", w.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result wecomResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	if result.ErrCode != 0 {
		return fmt.Errorf("wecom error: %d - %s", result.ErrCode, result.ErrMsg)
	}

	return nil
}

// Helper functions

func getStatusEmoji(status string) string {
	switch status {
	case "healthy":
		return "✅"
	case "warning":
		return "⚠️"
	case "degraded":
		return "🟠"
	default:
		return "🔴"
	}
}

func getSeverityIcon(severity string) string {
	switch severity {
	case "critical":
		return "🔴"
	case "high":
		return "🟠"
	case "medium":
		return "🟡"
	default:
		return "🔵"
	}
}

func truncateQuery(query string, maxLen int) string {
	// Clean up whitespace
	query = strings.Join(strings.Fields(query), " ")
	if len(query) <= maxLen {
		return query
	}
	return query[:maxLen-3] + "..."
}

// splitMessage splits the markdown message into chunks that fit within WeCom limits.
func splitMessage(msg string, maxLen int) []string {
	// Account for the suffix overhead: "\n\n*(Part X/Y)*" which is roughly 20 bytes.
	// We also leave some safety buffer for JSON escaping overhead (though Go's json.Marshal handles it,
	// byte length can grow if there are many characters needing escape).
	// A safe chunk size is slightly smaller than the hard limit.
	const safeLimit = 4000

	if len(msg) <= safeLimit {
		return []string{msg}
	}

	var chunks []string
	var currentChunk strings.Builder
	lines := strings.Split(msg, "\n")

	for _, line := range lines {
		// Calculate potential new length: current + newline + line
		newLen := currentChunk.Len() + len(line) + 1

		if newLen > safeLimit {
			// If current chunk is not empty, flush it
			if currentChunk.Len() > 0 {
				chunks = append(chunks, currentChunk.String())
				currentChunk.Reset()
			}

			// Handle case where a single line is massive (larger than safeLimit)
			// This shouldn't happen often with our SQL truncation, but good to be safe.
			// We force truncate/split it to avoid infinite loop or oversize payload.
			if len(line) > safeLimit {
				// Simple split for massive lines
				for len(line) > safeLimit {
					chunks = append(chunks, line[:safeLimit])
					line = line[safeLimit:]
				}
				currentChunk.WriteString(line)
				continue
			}
		}

		if currentChunk.Len() > 0 {
			currentChunk.WriteString("\n")
		}
		currentChunk.WriteString(line)
	}

	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	return chunks
}
