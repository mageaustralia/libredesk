package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/search"
	"github.com/zerodha/fastglue"
)

const (
	minSearchQueryLength = 3
)

// handleSearchConversations searches conversations based on the query.
func handleSearchConversations(r *fastglue.Request) error {
	app := r.Context.(*App)
	wrapper := func(query string) (interface{}, error) {
		return app.search.Conversations(query)
	}
	return handleSearch(r, wrapper)
}

// handleSearchMessages searches messages based on the query.
func handleSearchMessages(r *fastglue.Request) error {
	app := r.Context.(*App)
	wrapper := func(query string) (interface{}, error) {
		return app.search.Messages(query)
	}
	return handleSearch(r, wrapper)
}

// handleSearchContacts searches contacts based on the query.
func handleSearchContacts(r *fastglue.Request) error {
	app := r.Context.(*App)
	wrapper := func(query string) (interface{}, error) {
		return app.search.Contacts(query)
	}
	return handleSearch(r, wrapper)
}

// handleUnifiedSearch performs a single search across conversations and messages.
func handleUnifiedSearch(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		q     = string(r.RequestCtx.QueryArgs().Peek("query"))
		scope = string(r.RequestCtx.QueryArgs().Peek("scope"))
	)

	if len(q) < minSearchQueryLength {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.Ts("search.minQueryLength", "length", fmt.Sprintf("%d", minSearchQueryLength)), nil))
	}

	page, pageSize := getPagination(r)

	// No scope param: preserve the legacy flat response (mobile-app compat).
	if scope == "" {
		results, err := app.search.Unified(q, page, pageSize)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		return r.SendEnvelope(results)
	}

	switch scope {
	case search.ScopeAll, search.ScopeContacts, search.ScopeConversations, search.ScopeMessages:
	default:
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, "invalid scope", nil))
	}

	var filters search.Filters
	filters.StatusID, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("status_id")))
	filters.InboxID, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("inbox_id")))
	filters.AssignedUserID, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("assigned_user_id")))
	if v := string(r.RequestCtx.QueryArgs().Peek("from_date")); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filters.FromDate = &t
		}
	}
	if v := string(r.RequestCtx.QueryArgs().Peek("to_date")); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filters.ToDate = &t
		}
	}

	results, err := app.search.UnifiedGrouped(q, scope, filters, page, pageSize)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(results)
}

// handleSearch searches for the given query using the provided search function.
func handleSearch(r *fastglue.Request, searchFunc func(string) (interface{}, error)) error {
	var (
		app = r.Context.(*App)
		q   = string(r.RequestCtx.QueryArgs().Peek("query"))
	)

	if len(q) < minSearchQueryLength {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.Ts("search.minQueryLength", "length", fmt.Sprintf("%d", minSearchQueryLength)), nil))
	}

	results, err := searchFunc(q)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(results)
}
