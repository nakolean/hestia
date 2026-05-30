package models

import "time"

type APIKey struct {
	ID            int        `json:"id"`
	KeyHash       string     `json:"-"`
	Name          string     `json:"name"`
	Permissions   string     `json:"permissions"`
	IPWhitelist   string     `json:"ip_whitelist"`
	ExpiresAt     *time.Time `json:"expires_at"`
	LastUsedAt    *time.Time `json:"last_used_at"`
	CreatedAt     time.Time  `json:"created_at"`
}
