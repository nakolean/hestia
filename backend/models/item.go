package models

import "time"

type ShoppingItem struct {
	ID          int        `json:"id"`
	Text        string     `json:"text"`
	Purchased   bool       `json:"purchased"`
	CreatedAt   time.Time  `json:"created_at"`
	PurchasedAt *time.Time `json:"purchased_at"`
}
