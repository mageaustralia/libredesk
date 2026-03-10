package models

import "time"

var DefaultStatuses = []string{
	"Open",
	"Snoozed",
	"Resolved",
	"Closed",
}

type Status struct {
	ID         int       `db:"id" json:"id"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	Name       string    `db:"name" json:"name"`
	SortOrder  int       `db:"sort_order" json:"sort_order"`
	ShowOnSend bool      `db:"show_on_send" json:"show_on_send"`
}
