package main

import (
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/zerodha/fastglue"
)

// handleGetRecentActivities returns paginated recent activities across all conversations.
func handleGetRecentActivities(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		total = 0
	)
	page, pageSize := getPagination(r)
	if pageSize > 50 {
		pageSize = 50
	}

	activities, total, err := app.conversation.GetRecentActivities(page, pageSize)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(envelope.PageResults{
		Results:    activities,
		Total:      total,
		PerPage:    pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
		Page:       page,
	})
}
