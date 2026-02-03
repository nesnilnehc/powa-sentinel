// Package scheduler provides cron-based job scheduling for analysis runs.
package scheduler

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/powa-team/powa-sentinel/internal/engine"
	"github.com/powa-team/powa-sentinel/internal/notifier"
)

// DefaultAnalysisTimeout is the default timeout for analysis runs.
const DefaultAnalysisTimeout = 5 * time.Minute

// Scheduler manages scheduled analysis jobs.
type Scheduler struct {
	cron            *cron.Cron
	engine          *engine.Engine
	notifier        notifier.Notifier
	analysisTimeout time.Duration

	mu        sync.Mutex
	running   bool
	analyzing int32 // atomic flag to prevent concurrent analysis
}

// New creates a new Scheduler. Cron expressions are interpreted in loc; use time.UTC or time.LoadLocation("Asia/Shanghai") etc.
// If loc is nil, UTC is used.
func New(eng *engine.Engine, notify notifier.Notifier, loc *time.Location) *Scheduler {
	if loc == nil {
		loc = time.UTC
	}
	return &Scheduler{
		cron:            cron.New(cron.WithSeconds(), cron.WithLocation(loc)),
		engine:          eng,
		notifier:        notify,
		analysisTimeout: DefaultAnalysisTimeout,
	}
}

// SetAnalysisTimeout sets the timeout for analysis runs.
func (s *Scheduler) SetAnalysisTimeout(timeout time.Duration) {
	s.analysisTimeout = timeout
}

// Schedule adds a job with the given cron expression.
func (s *Scheduler) Schedule(cronExpr string) error {
	_, err := s.cron.AddFunc(cronExpr, func() {
		s.runAnalysis()
	})
	if err != nil {
		return err
	}
	return nil
}

// Start begins running scheduled jobs.
func (s *Scheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return
	}

	s.cron.Start()
	s.running = true
	log.Println("Scheduler started")
}

// Stop halts all scheduled jobs.
func (s *Scheduler) Stop() context.Context {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return context.Background()
	}

	ctx := s.cron.Stop()
	s.running = false
	log.Println("Scheduler stopped")
	return ctx
}

// RunNow triggers an immediate analysis run (bypassing schedule).
func (s *Scheduler) RunNow() {
	s.runAnalysis()
}

// runAnalysis executes the analysis and sends notifications.
// Uses atomic flag to prevent concurrent analysis runs.
func (s *Scheduler) runAnalysis() {
	// Check if analysis is already running (skip if so)
	if !atomic.CompareAndSwapInt32(&s.analyzing, 0, 1) {
		log.Println("Analysis already in progress, skipping this run")
		return
	}
	defer atomic.StoreInt32(&s.analyzing, 0)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), s.analysisTimeout)
	defer cancel()

	log.Println("Starting scheduled analysis...")

	alert, err := s.engine.Analyze(ctx)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("Analysis timed out after %v", s.analysisTimeout)
		} else {
			log.Printf("Analysis failed: %v", err)
		}
		return
	}

	log.Printf("Analysis complete: %d slow queries, %d regressions, %d suggestions",
		len(alert.TopSlowSQL), len(alert.Regressions), len(alert.Suggestions))

	if err := s.notifier.Send(ctx, alert); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("Notification timed out")
		} else {
			log.Printf("Notification failed: %v", err)
		}
		return
	}

	log.Printf("Notification sent via %s", s.notifier.Name())
}

// IsRunning returns whether the scheduler is currently active.
func (s *Scheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// IsAnalyzing returns whether an analysis is currently in progress.
func (s *Scheduler) IsAnalyzing() bool {
	return atomic.LoadInt32(&s.analyzing) == 1
}
