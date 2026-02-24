package conversation

import (
	"context"
	"fmt"
	"time"
)

// TrashSettingsFunc is a function that returns trash cleanup settings.
// Returns (autoTrashResolvedDays, autoTrashSpamDays, purgeTrashDays).
type TrashSettingsFunc func() (int, int, int)

// RunTrashManager runs the trash management routine every hour.
// It reads settings each cycle via the provided function so changes take effect without restart.
func (c *Manager) RunTrashManager(ctx context.Context, getSettings TrashSettingsFunc) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			autoTrashResolvedDays, autoTrashSpamDays, purgeTrashDays := getSettings()
			c.runTrashCycle(ctx, autoTrashResolvedDays, autoTrashSpamDays, purgeTrashDays)
		}
	}
}

func (c *Manager) runTrashCycle(ctx context.Context, autoTrashResolvedDays, autoTrashSpamDays, purgeTrashDays int) {
	// Auto-trash old resolved/closed conversations
	if autoTrashResolvedDays > 0 {
		res, err := c.q.AutoTrashResolved.ExecContext(ctx, autoTrashResolvedDays)
		if err != nil {
			c.lo.Error("error auto-trashing resolved conversations", "error", err)
		} else if rows, _ := res.RowsAffected(); rows > 0 {
			c.lo.Info(fmt.Sprintf("auto-trashed %d resolved/closed conversations", rows))
		}
	}

	// Auto-trash old spam conversations
	if autoTrashSpamDays > 0 {
		res, err := c.q.AutoTrashSpam.ExecContext(ctx, autoTrashSpamDays)
		if err != nil {
			c.lo.Error("error auto-trashing spam conversations", "error", err)
		} else if rows, _ := res.RowsAffected(); rows > 0 {
			c.lo.Info(fmt.Sprintf("auto-trashed %d spam conversations", rows))
		}
	}

	// Purge old trashed conversations (permanent delete)
	if purgeTrashDays > 0 {
		res, err := c.q.PurgeOldTrash.ExecContext(ctx, purgeTrashDays)
		if err != nil {
			c.lo.Error("error purging old trash", "error", err)
		} else if rows, _ := res.RowsAffected(); rows > 0 {
			c.lo.Info(fmt.Sprintf("permanently deleted %d trashed conversations", rows))
		}
	}
}
