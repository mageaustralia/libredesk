package conversation

import (
	"context"
	"fmt"
	"time"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/spamrescue"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/lib/pq"
)

const (
	spamRescueTimeout   = 25 * time.Second
	spamRescueMaxBodies = 6000
)

// spamRescue decides whether a message from a spam folder should stay in the inbox,
// according to the admin-configured rules. It returns true only when the message
// should actually be kept (live mode). In shadow mode it records what it would
// have done and returns false, so behaviour is unchanged.
func (m *Manager) spamRescue(in models.IncomingMessage, conversationUUID string) bool {
	s, err := m.settingsStore.GetSpamRescueSettings()
	if err != nil || !s.Enabled {
		return false
	}

	body := stringutil.HTML2Text(in.Content)
	if len(body) > spamRescueMaxBodies {
		body = body[:spamRescueMaxBodies]
	}

	checks := spamrescue.Checks{
		RepliedToOurThread: func(refs []string) (bool, error) {
			var ok bool
			err := m.q.ReferencesOurMessage.Get(&ok, pq.Array(refs))
			return ok, err
		},
	}
	if m.SpamRescueCustomerFunc != nil {
		checks.EcommerceCustomer = func(ctx context.Context, email string) (bool, error) {
			return m.SpamRescueCustomerFunc(ctx, in.InboxID, email)
		}
	}
	if m.SpamRescueOrderFunc != nil {
		checks.OrderBelongsTo = func(ctx context.Context, orderNumber, email string) (bool, error) {
			return m.SpamRescueOrderFunc(ctx, in.InboxID, orderNumber, email)
		}
	}
	if m.SpamRescueClassifyFunc != nil {
		checks.Classify = func(system, user string) (string, error) {
			type result struct {
				out string
				err error
			}
			ch := make(chan result, 1)
			go func() {
				out, err := m.SpamRescueClassifyFunc(system, user)
				ch <- result{out, err}
			}()
			select {
			case r := <-ch:
				return r.out, r.err
			case <-time.After(spamRescueTimeout):
				return "", fmt.Errorf("classifier timed out after %s", spamRescueTimeout)
			}
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), spamRescueTimeout)
	defer cancel()
	d := spamrescue.Evaluate(ctx, s, checks, spamrescue.Input{
		ContactID:  in.Contact.ID,
		Email:      in.Contact.Email.String,
		Subject:    in.Subject,
		Body:       body,
		References: nonEmpty(append([]string{in.InReplyTo}, in.References...)),
	})
	for _, w := range d.Warnings {
		m.lo.Warn("spam rescue check failed, treating as no match", "warning", w, "uuid", conversationUUID)
	}
	if !d.Rescue {
		return false
	}

	live := s.Mode == spamrescue.ModeLive
	var note string
	if live {
		note = fmt.Sprintf("Kept in inbox by spam rescue (%s): %s", d.Rule, d.Reason)
	} else {
		note = fmt.Sprintf("Spam rescue (shadow mode, no change made) would have kept this in the inbox (%s): %s", d.Rule, d.Reason)
	}
	m.lo.Info("spam rescue match", "mode", s.Mode, "rule", d.Rule, "uuid", conversationUUID)
	if sys, err := m.userStore.GetSystemUser(); err != nil {
		m.lo.Error("could not fetch system user for spam rescue note", "error", err)
	} else if err := m.InsertConversationActivity(models.ActivitySpamRescue, conversationUUID, note, sys); err != nil {
		m.lo.Error("could not record spam rescue note", "error", err, "uuid", conversationUUID)
	}
	return live
}

// nonEmpty drops blank entries so an absent header can't match a blank source ID.
func nonEmpty(in []string) []string {
	out := in[:0:0]
	for _, v := range in {
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}
