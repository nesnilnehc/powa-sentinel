package notifier

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/powa-team/powa-sentinel/internal/model"
)

// ConsoleNotifier prints alerts to the console (useful for testing).
type ConsoleNotifier struct{}

// NewConsoleNotifier creates a new console notifier.
func NewConsoleNotifier() *ConsoleNotifier {
	return &ConsoleNotifier{}
}

// Name returns the notifier name.
func (c *ConsoleNotifier) Name() string {
	return "console"
}

// Send prints the alert to the console.
func (c *ConsoleNotifier) Send(ctx context.Context, alert *model.AlertContext) error {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString("═══════════════════════════════════════════════════════════════\n")
	sb.WriteString("                    POWA SENTINEL REPORT                       \n")
	sb.WriteString("═══════════════════════════════════════════════════════════════\n")
	sb.WriteString(fmt.Sprintf("Report ID:    %s\n", alert.ReqID))
	sb.WriteString(fmt.Sprintf("Timestamp:    %s\n", alert.Timestamp.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("Health Score: %d/100 (%s)\n", alert.Summary.HealthScore, alert.Summary.HealthStatus))
	sb.WriteString("───────────────────────────────────────────────────────────────\n")

	sb.WriteString(fmt.Sprintf("Analysis Window:  %s ~ %s\n",
		alert.AnalysisWindow.Start.Format("2006-01-02 15:04"),
		alert.AnalysisWindow.End.Format("2006-01-02 15:04")))
	sb.WriteString(fmt.Sprintf("Baseline Window:  %s ~ %s\n",
		alert.BaselineWindow.Start.Format("2006-01-02 15:04"),
		alert.BaselineWindow.End.Format("2006-01-02 15:04")))
	sb.WriteString("───────────────────────────────────────────────────────────────\n")

	sb.WriteString("\n📊 SUMMARY\n")
	sb.WriteString(fmt.Sprintf("  • Queries Analyzed: %d\n", alert.Summary.TotalQueriesAnalyzed))
	sb.WriteString(fmt.Sprintf("  • Slow Queries:     %d\n", alert.Summary.SlowQueryCount))
	sb.WriteString(fmt.Sprintf("  • Regressions:      %d\n", alert.Summary.RegressionCount))
	sb.WriteString(fmt.Sprintf("  • Index Suggestions: %d\n", alert.Summary.SuggestionCount))

	if len(alert.TopSlowSQL) > 0 {
		sb.WriteString("\n⏱ TOP SLOW QUERIES\n")
		for i, q := range alert.TopSlowSQL {
			sb.WriteString(fmt.Sprintf("  %d. [%d] %.2fms (×%d calls)\n",
				i+1, q.QueryID, q.TotalTime, q.Calls))
			query := strings.Join(strings.Fields(q.Query), " ")
			if len(query) > 60 {
				query = query[:57] + "..."
			}
			sb.WriteString(fmt.Sprintf("      %s\n", query))
		}
	}

	if len(alert.Regressions) > 0 {
		sb.WriteString("\n📈 REGRESSIONS\n")
		count := 0
		for i, r := range alert.Regressions {
			if count >= 20 {
				sb.WriteString(fmt.Sprintf("  ... and %d more\n", len(alert.Regressions)-20))
				break
			}
			count++
			serverInfo := r.DatabaseName
			if r.ServerName != "" && r.ServerName != "local" {
				serverInfo = fmt.Sprintf("%s/%s", r.ServerName, r.DatabaseName)
			}
			sb.WriteString(fmt.Sprintf("  %d. [%d] [%s] %.2fms → %.2fms (+%.1f%%) [%s]\n",
				i+1, r.QueryID, serverInfo, r.BaselineMeanTime, r.CurrentMeanTime, r.ChangePercent, r.Severity))
			query := strings.Join(strings.Fields(r.Query), " ")
			if len(query) > 60 {
				query = query[:57] + "..."
			}
			sb.WriteString(fmt.Sprintf("      %s\n", query))
		}
	}

	if len(alert.Suggestions) > 0 {
		sb.WriteString("\n💡 INDEX SUGGESTIONS\n")
		for i, s := range alert.Suggestions {
			sb.WriteString(fmt.Sprintf("  %d. %s (%s) - Est. +%.0f%%\n",
				i+1, s.FullTableName(), strings.Join(s.Columns, ", "), s.EstImprovementPercent))
		}
	}

	sb.WriteString("\n═══════════════════════════════════════════════════════════════\n")

	log.Print(sb.String())
	return nil
}
