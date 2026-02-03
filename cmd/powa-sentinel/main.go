// powa-sentinel is a lightweight sidecar service for PoWA that
// periodically analyzes performance statistics and pushes actionable
// insights to notification channels.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/powa-team/powa-sentinel/internal/config"
	"github.com/powa-team/powa-sentinel/internal/engine"
	"github.com/powa-team/powa-sentinel/internal/notifier"
	"github.com/powa-team/powa-sentinel/internal/reader"
	"github.com/powa-team/powa-sentinel/internal/scheduler"
	"github.com/powa-team/powa-sentinel/internal/server"
)

var (
	// Version information (set at build time via -ldflags)
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

func main() {
	// Parse command-line flags
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	runOnce := flag.Bool("once", false, "Run analysis once and exit (skip scheduler)")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *showVersion {
		fmt.Printf("powa-sentinel %s (commit: %s, built: %s)\n", version, commit, buildDate)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	log.Printf("powa-sentinel %s starting...", version)

	// Initialize database reader
	dbReader, err := reader.New(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database reader: %v", err)
	}
	defer dbReader.Close()

	// Test database connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := dbReader.Ping(ctx); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	cancel()
	log.Println("Database connection established")

	// Initialize analysis engine
	eng := engine.New(cfg, dbReader)

	// Initialize notifier
	var notify notifier.Notifier
	switch cfg.Notifier.Type {
	case "wecom":
		notify, err = notifier.NewWeComNotifier(&cfg.Notifier)
		if err != nil {
			log.Fatalf("Failed to initialize WeCom notifier: %v", err)
		}
	case "console":
		notify = notifier.NewConsoleNotifier()
	default:
		log.Fatalf("Unknown notifier type: %s", cfg.Notifier.Type)
	}
	log.Printf("Notifier initialized: %s", notify.Name())

	// Run-once mode
	if *runOnce {
		log.Println("Running single analysis (--once mode)")

		// Use same timeout as scheduler would
		analysisCtx, analysisCancel := context.WithTimeout(context.Background(), scheduler.DefaultAnalysisTimeout)
		defer analysisCancel()

		alert, err := eng.Analyze(analysisCtx)
		if err != nil {
			if analysisCtx.Err() == context.DeadlineExceeded {
				log.Fatalf("Analysis timed out after %v", scheduler.DefaultAnalysisTimeout)
			}
			log.Fatalf("Analysis failed: %v", err)
		}

		if err := notify.Send(analysisCtx, alert); err != nil {
			if analysisCtx.Err() == context.DeadlineExceeded {
				log.Fatalf("Notification timed out")
			}
			log.Fatalf("Notification failed: %v", err)
		}

		log.Println("Analysis complete, exiting")
		return
	}

	// Initialize health server
	healthServer := server.New(&cfg.Server, dbReader)
	if err := healthServer.Start(); err != nil {
		log.Fatalf("Failed to start health server: %v", err)
	}

	// Initialize scheduler (cron interpreted in configured timezone; Location set by config.Validate)
	sched := scheduler.New(eng, notify, cfg.Schedule.Location)
	if err := sched.Schedule(cfg.Schedule.Cron); err != nil {
		log.Fatalf("Failed to schedule job: %v", err)
	}
	sched.Start()
	log.Printf("Scheduler started with cron: %s (timezone: %s)", cfg.Schedule.Cron, cfg.Schedule.Timezone)

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan
	log.Printf("Received signal %v, shutting down...", sig)

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Stop scheduler
	schedCtx := sched.Stop()
	select {
	case <-schedCtx.Done():
	case <-shutdownCtx.Done():
	}

	// Stop health server
	if err := healthServer.Stop(shutdownCtx); err != nil {
		log.Printf("Error stopping health server: %v", err)
	}

	log.Println("Shutdown complete")
}
