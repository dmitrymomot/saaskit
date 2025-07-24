package queue

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

// SchedulerRepository defines the interface for scheduler operations
type SchedulerRepository interface {
	// CreateTask creates a new task in the storage
	CreateTask(ctx context.Context, task *Task) error

	// GetPendingTaskByName checks if a pending task with given name exists
	GetPendingTaskByName(ctx context.Context, taskName string) (*Task, error)
}

// Scheduler manages periodic task scheduling
type Scheduler struct {
	repo     SchedulerRepository
	tasks    map[string]*scheduledTask
	mu       sync.RWMutex
	ticker   *time.Ticker
	interval time.Duration
	logger   *slog.Logger
}

// scheduledTask holds configuration for a periodic task
type scheduledTask struct {
	name            string
	schedule        Schedule
	queue           string
	priority        Priority
	maxRetries      int8
	lastScheduledAt *time.Time // Track when we last created a task
}

// NewScheduler creates a new task scheduler
func NewScheduler(repo SchedulerRepository, opts ...SchedulerOption) (*Scheduler, error) {
	if repo == nil {
		return nil, ErrRepositoryNil
	}

	// Default options
	options := &schedulerOptions{
		checkInterval: 30 * time.Second,
		logger:        slog.Default(),
	}

	// Apply options
	for _, opt := range opts {
		opt(options)
	}

	return &Scheduler{
		repo:     repo,
		tasks:    make(map[string]*scheduledTask),
		interval: options.checkInterval,
		logger:   options.logger,
	}, nil
}

// AddTask registers a periodic task
func (s *Scheduler) AddTask(name string, schedule Schedule, opts ...SchedulerTaskOption) error {
	// Default task options
	taskOpts := &schedulerTaskOptions{
		queue:      DefaultQueueName,
		priority:   PriorityDefault,
		maxRetries: 3,
	}

	// Apply options
	for _, opt := range opts {
		opt(taskOpts)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if task already registered
	if _, exists := s.tasks[name]; exists {
		return ErrTaskAlreadyRegistered
	}

	// Register the task
	task := &scheduledTask{
		name:       name,
		schedule:   schedule,
		queue:      taskOpts.queue,
		priority:   taskOpts.priority,
		maxRetries: taskOpts.maxRetries,
	}

	s.tasks[name] = task

	// Log registration
	s.logger.Info("registered periodic task",
		slog.String("task_name", name),
		slog.String("schedule", schedule.String()))

	return nil
}

// Start begins the scheduler's periodic task checking
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.RLock()
	taskCount := len(s.tasks)
	s.mu.RUnlock()

	if taskCount == 0 {
		return ErrSchedulerNotConfigured
	}

	// Create ticker
	s.ticker = time.NewTicker(s.interval)
	defer s.ticker.Stop()

	// Check immediately on start
	s.checkTasks(ctx)

	// Then check periodically
	for {
		select {
		case <-ctx.Done():
			s.logger.Info("scheduler shutting down")
			return ctx.Err()
		case <-s.ticker.C:
			s.checkTasks(ctx)
		}
	}
}

// checkTasks checks all registered tasks and creates any that are due
func (s *Scheduler) checkTasks(ctx context.Context) {
	// Get a snapshot of tasks
	s.mu.RLock()
	tasks := make([]*scheduledTask, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}
	s.mu.RUnlock()

	now := time.Now()

	// Check each task
	for _, task := range tasks {
		// 1. Always calculate next run time first
		var nextRun time.Time
		if task.lastScheduledAt == nil {
			// First run: next run from now
			nextRun = task.schedule.Next(now)
		} else {
			// Subsequent runs: next run from last scheduled
			nextRun = task.schedule.Next(*task.lastScheduledAt)
		}

		// 2. Skip if not due (only for subsequent runs)
		if task.lastScheduledAt != nil && nextRun.After(now) {
			s.logger.Debug("periodic task not due yet",
				slog.String("task_name", task.name),
				slog.Time("next_run", nextRun))
			continue
		}

		// 3. Always check DB when we reach here
		existing, err := s.repo.GetPendingTaskByName(ctx, task.name)
		if err == nil && existing != nil {
			// Task exists - just update our state
			s.mu.Lock()
			if t, ok := s.tasks[task.name]; ok {
				t.lastScheduledAt = &existing.ScheduledAt
			}
			s.mu.Unlock()

			s.logger.Debug("periodic task already pending",
				slog.String("task_name", task.name),
				slog.Time("scheduled_for", existing.ScheduledAt))
			continue
		}

		// 4. Create task (we only reach here if no existing task)
		if err := s.createTask(ctx, task, nextRun); err != nil {
			s.logger.Error("failed to create periodic task",
				slog.String("task_name", task.name),
				slog.String("error", err.Error()))
			continue
		}

		// 5. Update state
		s.mu.Lock()
		if t, ok := s.tasks[task.name]; ok {
			t.lastScheduledAt = &nextRun
		}
		s.mu.Unlock()

		// Log with appropriate message
		if task.lastScheduledAt == nil {
			s.logger.Info("created periodic task (first run)",
				slog.String("task_name", task.name),
				slog.Time("scheduled_for", nextRun))
		} else {
			s.logger.Info("created periodic task",
				slog.String("task_name", task.name),
				slog.Time("scheduled_for", nextRun))
		}
	}
}

// createTask creates a new task instance in the database
func (s *Scheduler) createTask(ctx context.Context, task *scheduledTask, scheduledAt time.Time) error {
	newTask := &Task{
		ID:          uuid.New(),
		Queue:       task.queue,
		TaskType:    TaskTypePeriodic,
		TaskName:    task.name,
		Payload:     nil, // Periodic tasks have no payload
		Status:      TaskStatusPending,
		Priority:    task.priority,
		RetryCount:  0,
		MaxRetries:  task.maxRetries,
		ScheduledAt: scheduledAt,
		CreatedAt:   time.Now(),
	}

	return s.repo.CreateTask(ctx, newTask)
}

// RemoveTask removes a periodic task from the scheduler
func (s *Scheduler) RemoveTask(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.tasks, name)

	s.logger.Info("removed periodic task",
		slog.String("task_name", name))
}

// ListTasks returns all registered periodic tasks
func (s *Scheduler) ListTasks() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, 0, len(s.tasks))
	for name := range s.tasks {
		names = append(names, name)
	}
	return names
}
