package main

import (
	"fmt"
	"strconv"
	"time"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/search"
	smodels "github.com/abhinavxd/libredesk/internal/search/models"
	"github.com/zerodha/fastglue"
)

const (
	minSearchQueryLength = 3
)

// handleSearchConversations searches conversations based on the query.
func handleSearchConversations(r *fastglue.Request) error {
	app, user, q, err := searchInputs(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	results, err := app.search.Conversations(q)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	uuids := make([]string, len(results))
	for i, c := range results {
		uuids[i] = c.UUID
	}
	allowed, err := app.conversation.FilterAuthorizedListUUIDs(user.ID, uuids)
	if err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}
	set := uuidSet(allowed)
	out := make([]smodels.ConversationResult, 0, len(allowed))
	for _, c := range results {
		if _, ok := set[c.UUID]; ok {
			out = append(out, c)
		}
	}
	return r.SendEnvelope(out)
}

// handleSearchMessages searches messages based on the query.
func handleSearchMessages(r *fastglue.Request) error {
	app, user, q, err := searchInputs(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	results, err := app.search.Messages(q)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	uuids := make([]string, len(results))
	for i, m := range results {
		uuids[i] = m.ConversationUUID
	}
	allowed, err := app.conversation.FilterAuthorizedListUUIDs(user.ID, uuids)
	if err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}
	set := uuidSet(allowed)
	out := make([]smodels.MessageResult, 0, len(allowed))
	for _, m := range results {
		if _, ok := set[m.ConversationUUID]; ok {
			out = append(out, m)
		}
	}
	return r.SendEnvelope(out)
}

// handleSearchContacts searches contacts based on the query.
func handleSearchContacts(r *fastglue.Request) error {
	app, _, q, err := searchInputs(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	results, err := app.search.Contacts(q)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(results)
}

// handleUnifiedSearch performs a single search across conversations, subject,
// reference number, contact email, and message text content. Returns paginated
// results — see FS8.
func handleUnifiedSearch(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		q     = string(r.RequestCtx.QueryArgs().Peek("query"))
		scope = string(r.RequestCtx.QueryArgs().Peek("scope"))
	)

	if len(q) < minSearchQueryLength {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.Ts("search.minQueryLength", "length", fmt.Sprintf("%d", minSearchQueryLength)), nil))
	}

	page, pageSize := getSearchPagination(r)

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

// getSearchPagination returns 1-indexed page and page_size with sane defaults
// (page=1, page_size=30, capped at 100). Kept narrow to the search package
// to avoid colliding with any future generic pagination helper.
func getSearchPagination(r *fastglue.Request) (page, pageSize int) {
	page = r.RequestCtx.QueryArgs().GetUintOrZero("page")
	if page < 1 {
		page = 1
	}
	pageSize = r.RequestCtx.QueryArgs().GetUintOrZero("page_size")
	if pageSize < 1 {
		pageSize = 30
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
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

func searchInputs(r *fastglue.Request) (*App, amodels.User, string, error) {
	app := r.Context.(*App)
	user, _ := r.RequestCtx.UserValue("user").(amodels.User)
	q := string(r.RequestCtx.QueryArgs().Peek("query"))
	if len(q) < minSearchQueryLength {
		return app, user, "", envelope.NewError(envelope.InputError, app.i18n.Ts("search.minQueryLength", "length", fmt.Sprintf("%d", minSearchQueryLength)), nil)
	}
	return app, user, q, nil
}

func uuidSet(uuids []string) map[string]struct{} {
	s := make(map[string]struct{}, len(uuids))
	for _, u := range uuids {
		s[u] = struct{}{}
	}
	return s
}
