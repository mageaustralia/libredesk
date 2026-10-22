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
func (m *Manager) spamRescue(in models.IncomingMessage) bool {
	s, err := m.settingsStore.GetSpamRescueSettings()
	if err != nil || !s.Enabled {
		return false
	}

	body := in.Message.TextContent
	if body == "" {
		body = stringutil.HTML2Text(in.Message.Content)
	}
	if len(body) > spamRescueMaxBodies {
		body = body[:spamRescueMaxBodies]
	}

	checks := spamrescue.Checks{
		RepliedToOurThread: func(refs []string) (bool, error) {
			var ok bool
			err := m.q.ReferencesOurMessage.Get(&ok, pq.Array(refs))
			return ok, err
		},
		EcommerceCustomer: m.SpamRescueCustomerFunc,
		OrderBelongsTo:    m.SpamRescueOrderFunc,
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
		Subject:    in.Message.Subject,
		Body:       body,
		References: nonEmpty(append([]string{in.Message.InReplyTo}, in.Message.References...)),
	})
	for _, w := range d.Warnings {
		m.lo.Warn("spam rescue check failed, treating as no match", "warning", w, "uuid", in.Message.ConversationUUID)
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
	m.lo.Info("spam rescue match", "mode", s.Mode, "rule", d.Rule, "uuid", in.Message.ConversationUUID)
	if sys, err := m.userStore.GetSystemUser(); err != nil {
		m.lo.Error("could not fetch system user for spam rescue note", "error", err)
	} else if err := m.InsertConversationActivity(models.ActivitySpamRescue, in.Message.ConversationUUID, note, sys); err != nil {
		m.lo.Error("could not record spam rescue note", "error", err, "uuid", in.Message.ConversationUUID)
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
