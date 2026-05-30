package services

import (
	"context"
	"database/sql"
	"log"
	"sync"
	"time"
)

// ReminderService monitors overdue chores and logs their count every 5 minutes
type ReminderService struct {
	db       *sql.DB
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// NewReminderService creates a new reminder service
func NewReminderService(db *sql.DB) *ReminderService {
	return &ReminderService{
		db:       db,
		stopChan: make(chan struct{}),
	}
}

// Start begins the reminder service monitoring
func (s *ReminderService) Start() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		// Initial sweep
		s.sweep()

		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.sweep()
			case <-s.stopChan:
				return
			}
		}
	}()
}

// Stop gracefully shuts down the reminder service
func (s *ReminderService) Stop() {
	close(s.stopChan)
	s.wg.Wait()
}

// sweep checks for overdue chores and logs the count
func (s *ReminderService) sweep() {
	var count int
	err := s.db.QueryRowContext(
		context.Background(),
		"SELECT COUNT(*) FROM chores WHERE next_due <= ? AND completed = 0",
		time.Now(),
	).Scan(&count)
	if err != nil {
		log.Printf("ReminderService: failed to count overdue chores: %v", err)
		return
	}
	log.Printf("ReminderService: %d overdue chores detected", count)
}
