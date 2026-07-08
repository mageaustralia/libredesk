// Package search provides search functionality.
package search

import (
	"time"
	"embed"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	models "github.com/abhinavxd/libredesk/internal/search/models"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

// Manager is the search manager
type Manager struct {
	q    queries
	lo   *logf.Logger
	i18n *i18n.I18n
}

// Opts contains the options for creating a new search manager
type Opts struct {
	DB   *sqlx.DB
	Lo   *logf.Logger
	I18n *i18n.I18n
}

// queries contains all the prepared queries
type queries struct {
	SearchConversationsByRefNum       *sqlx.Stmt `query:"search-conversations-by-reference-number"`
	SearchConversationsByContactEmail *sqlx.Stmt `query:"search-conversations-by-contact-email"`
	SearchConversationsBySubject      *sqlx.Stmt `query:"search-conversations-by-subject"`
	SearchMessages                    *sqlx.Stmt `query:"search-messages"`
	SearchContacts                    *sqlx.Stmt `query:"search-contacts"`
	SearchUnified                     *sqlx.Stmt `query:"search-unified"`
	SearchUnifiedContacts             *sqlx.Stmt `query:"search-unified-contacts"`
	SearchUnifiedConversations        *sqlx.Stmt `query:"search-unified-conversations"`
	SearchUnifiedMessages             *sqlx.Stmt `query:"search-unified-messages"`
}

// New creates a new search manager
func New(opts Opts) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}
	return &Manager{q: q, lo: opts.Lo, i18n: opts.I18n}, nil
}

// Conversations searches conversations based on the query
func (s *Manager) Conversations(query string) ([]models.ConversationResult, error) {
	var refNumResults = make([]models.ConversationResult, 0)
	if err := s.q.SearchConversationsByRefNum.Select(&refNumResults, query); err != nil {
		s.lo.Error("error searching conversations", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, s.i18n.Ts("globals.messages.errorSearching", "name", s.i18n.Ts("globals.terms.conversation")), nil)
	}

	var emailResults = make([]models.ConversationResult, 0)
	if err := s.q.SearchConversationsByContactEmail.Select(&emailResults, query); err != nil {
		s.lo.Error("error searching conversations", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, s.i18n.Ts("globals.messages.errorSearching", "name", s.i18n.Ts("globals.terms.conversation")), nil)
	}

	var subjectResults = make([]models.ConversationResult, 0)
	if err := s.q.SearchConversationsBySubject.Select(&subjectResults, query); err != nil {
		s.lo.Error("error searching conversations by subject", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, s.i18n.Ts("globals.messages.errorSearching", "name", s.i18n.Ts("globals.terms.conversation")), nil)
	}

	// Combine results, deduplicating by UUID.
	seen := make(map[string]bool)
	var combined []models.ConversationResult
	for _, r := range append(append(refNumResults, emailResults...), subjectResults...) {
		if !seen[r.UUID] {
			seen[r.UUID] = true
			combined = append(combined, r)
		}
	}
	if combined == nil {
		combined = make([]models.ConversationResult, 0)
	}
	return combined, nil
}

// Messages searches messages based on the query
func (s *Manager) Messages(query string) ([]models.MessageResult, error) {
	var results = make([]models.MessageResult, 0)
	if err := s.q.SearchMessages.Select(&results, query); err != nil {
		s.lo.Error("error searching messages", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, s.i18n.Ts("globals.messages.errorSearching", "name", s.i18n.Ts("globals.terms.message")), nil)
	}
	return results, nil
}

// UnifiedResponse wraps search results with total count.
type UnifiedResponse struct {
	Results []models.UnifiedResult `json:"results"`
	Total   int                    `json:"total"`
	Page    int                    `json:"page"`
}

// Unified performs a single search across conversations and messages.
func (s *Manager) Unified(query string, page, pageSize int) (*UnifiedResponse, error) {
	var results = make([]models.UnifiedResult, 0)
	offset := (page - 1) * pageSize
	if err := s.q.SearchUnified.Select(&results, query, pageSize, offset); err != nil {
		s.lo.Error("error in unified search", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, s.i18n.Ts("globals.messages.errorSearching", "name", s.i18n.Ts("globals.terms.conversation")), nil)
	}
	total := 0
	if len(results) > 0 {
		total = results[0].Total
	}
	return &UnifiedResponse{Results: results, Total: total, Page: page}, nil
}

// Contacts searches contacts based on the query
func (s *Manager) Contacts(query string) ([]models.ContactResult, error) {
	var results = make([]models.ContactResult, 0)
	if err := s.q.SearchContacts.Select(&results, query); err != nil {
		s.lo.Error("error searching contacts", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, s.i18n.Ts("globals.messages.errorSearching", "name", s.i18n.Ts("globals.terms.contact")), nil)
	}
	return results, nil
}

// Scopes for grouped unified search.
const (
	ScopeAll           = "all"
	ScopeContacts      = "contacts"
	ScopeConversations = "conversations"
	ScopeMessages      = "messages"

	// Per-group limits when scope=all (single overview page, no pagination).
	allScopeContactsLimit = 5
	allScopeGroupLimit    = 10
)

// Filters are optional constraints on the conversation/message groups.
// Zero/nil values mean "no filter". AssignedUserID -1 means unassigned.
type Filters struct {
	StatusID       int
	InboxID        int
	FromDate       *time.Time
	ToDate         *time.Time
	AssignedUserID int
}

type ContactGroup struct {
	Results []models.UnifiedContactResult `json:"results"`
	Total   int                           `json:"total"`
}

type ConversationGroup struct {
	Results []models.UnifiedConversationResult `json:"results"`
	Total   int                                `json:"total"`
}

type MessageGroup struct {
	Results []models.UnifiedMessageResult `json:"results"`
	Total   int                           `json:"total"`
}

// GroupedResponse is the scope-aware unified search response.
type GroupedResponse struct {
	Contacts      ContactGroup      `json:"contacts"`
	Conversations ConversationGroup `json:"conversations"`
	Messages      MessageGroup      `json:"messages"`
	Page          int               `json:"page"`
}

// UnifiedGrouped searches contacts, conversations and messages as separate
// ranked groups. scope=all returns a capped overview of all three; a specific
// scope paginates within that group only.
func (s *Manager) UnifiedGrouped(query, scope string, f Filters, page, pageSize int) (*GroupedResponse, error) {
	resp := &GroupedResponse{
		Page:          page,
		Contacts:      ContactGroup{Results: make([]models.UnifiedContactResult, 0)},
		Conversations: ConversationGroup{Results: make([]models.UnifiedConversationResult, 0)},
		Messages:      MessageGroup{Results: make([]models.UnifiedMessageResult, 0)},
	}
	offset := (page - 1) * pageSize

	limitFor := func(allLimit int) (int, int) {
		if scope == ScopeAll {
			return allLimit, 0
		}
		return pageSize, offset
	}

	if scope == ScopeAll || scope == ScopeContacts {
		limit, off := limitFor(allScopeContactsLimit)
		if err := s.q.SearchUnifiedContacts.Select(&resp.Contacts.Results, query, limit, off); err != nil {
			s.lo.Error("error in unified contact search", "error", err)
			return nil, envelope.NewError(envelope.GeneralError, s.i18n.Ts("globals.messages.errorSearching", "name", s.i18n.Ts("globals.terms.contact")), nil)
		}
		if len(resp.Contacts.Results) > 0 {
			resp.Contacts.Total = resp.Contacts.Results[0].Total
		}
	}

	if scope == ScopeAll || scope == ScopeConversations {
		limit, off := limitFor(allScopeGroupLimit)
		if err := s.q.SearchUnifiedConversations.Select(&resp.Conversations.Results, query, f.StatusID, f.InboxID, f.FromDate, f.ToDate, f.AssignedUserID, limit, off); err != nil {
			s.lo.Error("error in unified conversation search", "error", err)
			return nil, envelope.NewError(envelope.GeneralError, s.i18n.Ts("globals.messages.errorSearching", "name", s.i18n.Ts("globals.terms.conversation")), nil)
		}
		if len(resp.Conversations.Results) > 0 {
			resp.Conversations.Total = resp.Conversations.Results[0].Total
		}
	}

	if scope == ScopeAll || scope == ScopeMessages {
		limit, off := limitFor(allScopeGroupLimit)
		if err := s.q.SearchUnifiedMessages.Select(&resp.Messages.Results, query, f.StatusID, f.InboxID, f.FromDate, f.ToDate, f.AssignedUserID, limit, off); err != nil {
			s.lo.Error("error in unified message search", "error", err)
			return nil, envelope.NewError(envelope.GeneralError, s.i18n.Ts("globals.messages.errorSearching", "name", s.i18n.Ts("globals.terms.message")), nil)
		}
		if len(resp.Messages.Results) > 0 {
			resp.Messages.Total = resp.Messages.Results[0].Total
		}
	}

	return resp, nil
}
