package models

import (
	"time"

	"github.com/volatiletech/null/v9"
)

type ConversationResult struct {
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	UUID            string    `db:"uuid" json:"uuid"`
	ReferenceNumber string    `db:"reference_number" json:"reference_number"`
	Subject         string    `db:"subject" json:"subject"`
}

type MessageResult struct {
	CreatedAt                   time.Time `db:"created_at" json:"created_at"`
	TextContent                 string    `db:"text_content" json:"text_content"`
	ConversationCreatedAt       time.Time `db:"conversation_created_at" json:"conversation_created_at"`
	ConversationUUID            string    `db:"conversation_uuid" json:"conversation_uuid"`
	ConversationReferenceNumber string    `db:"conversation_reference_number" json:"conversation_reference_number"`
}

type UnifiedResult struct {
	Total             int         `db:"total" json:"-"`
	CreatedAt         time.Time   `db:"created_at" json:"created_at"`
	LastMessageAt     null.Time   `db:"last_message_at" json:"last_message_at"`
	LastMessageSender null.String `db:"last_message_sender" json:"last_message_sender"`
	UUID              string      `db:"uuid" json:"uuid"`
	ReferenceNumber   string      `db:"reference_number" json:"reference_number"`
	Subject           string      `db:"subject" json:"subject"`
	Snippet           string      `db:"snippet" json:"snippet"`
}

type ContactResult struct {
	ID        int       `db:"id" json:"id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	FirstName string    `db:"first_name" json:"first_name"`
	LastName  string    `db:"last_name" json:"last_name"`
	Email     string    `db:"email" json:"email"`
}

type UnifiedContactResult struct {
	Total      int       `db:"total" json:"-"`
	ID         int       `db:"id" json:"id"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	FirstName  string    `db:"first_name" json:"first_name"`
	LastName   string    `db:"last_name" json:"last_name"`
	Email      string    `db:"email" json:"email"`
	Similarity float64   `db:"sim" json:"-"`
}

type UnifiedConversationResult struct {
	Total             int         `db:"total" json:"-"`
	CreatedAt         time.Time   `db:"created_at" json:"created_at"`
	LastMessageAt     null.Time   `db:"last_message_at" json:"last_message_at"`
	LastMessageSender null.String `db:"last_message_sender" json:"last_message_sender"`
	UUID              string      `db:"uuid" json:"uuid"`
	ReferenceNumber   string      `db:"reference_number" json:"reference_number"`
	Subject           string      `db:"subject" json:"subject"`
	ContactName       string      `db:"contact_name" json:"contact_name"`
	ContactEmail      string      `db:"contact_email" json:"contact_email"`
	MatchRank         int         `db:"match_rank" json:"-"`
}

type UnifiedMessageResult struct {
	Total           int       `db:"total" json:"-"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	UUID            string    `db:"uuid" json:"uuid"`
	ReferenceNumber string    `db:"reference_number" json:"reference_number"`
	Subject         string    `db:"subject" json:"subject"`
	Snippet         string    `db:"snippet" json:"snippet"`
	MatchRank       float64   `db:"match_rank" json:"-"`
}
