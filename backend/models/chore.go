package models

import (
	"fmt"
	"time"
)

type Chore struct {
	ID            int       `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	FrequencyNum  int       `json:"frequency_num"`
	FrequencyUnit string    `json:"frequency_unit"`
	Completed     bool      `json:"completed"`
	LastCompleted *time.Time `json:"last_completed"`
	NextDue       *time.Time `json:"next_due"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// NextDue computes the next due time from a frequency.
func NextDue(freqNum int, freqUnit string) time.Time {
	var duration time.Duration
	switch freqUnit {
	case "hours":
		duration = time.Duration(freqNum) * time.Hour
	case "days":
		duration = time.Duration(freqNum) * 24 * time.Hour
	case "weeks":
		duration = time.Duration(freqNum*7) * 24 * time.Hour
	default:
		duration = time.Duration(freqNum) * 24 * time.Hour
	}
	return time.Now().Add(duration)
}

// CalculateNextDue computes the next due time relative to a given time.
func CalculateNextDue(from time.Time, freqNum int, freqUnit string) time.Time {
	var duration time.Duration
	switch freqUnit {
	case "hours":
		duration = time.Duration(freqNum) * time.Hour
	case "days":
		duration = time.Duration(freqNum) * 24 * time.Hour
	case "weeks":
		duration = time.Duration(freqNum*7) * 24 * time.Hour
	default:
		duration = time.Duration(freqNum) * 24 * time.Hour
	}
	return from.Add(duration)
}

// MarkCompleted marks the chore as completed, sets lastCompleted to now,
// and computes the new nextDue from its frequency.
func (c *Chore) MarkCompleted() {
	c.Completed = true
	now := time.Now()
	c.LastCompleted = &now
	nextDue := NextDue(c.FrequencyNum, c.FrequencyUnit)
	c.NextDue = &nextDue
}

// ValidateFrequencyUnit returns true if the unit is valid.
func ValidateFrequencyUnit(unit string) bool {
	switch unit {
	case "hours", "days", "weeks":
		return true
	default:
		return false
	}
}

// FormatFrequency returns a human-readable frequency string.
func FormatFrequency(freqNum int, freqUnit string) string {
	suffix := ""
	switch freqUnit {
	case "hours":
		suffix = "hour"
	case "days":
		suffix = "day"
	case "weeks":
		suffix = "week"
	default:
		suffix = "day"
	}
	if freqNum != 1 {
		suffix += "s"
	}
	return fmt.Sprintf("%d %s", freqNum, suffix)
}
