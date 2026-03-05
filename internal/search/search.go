// Package search provides search functionality.
package search

import (
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
