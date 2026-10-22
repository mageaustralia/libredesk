package spamrescue

import (
	"context"
	"errors"
	"testing"

	"github.com/abhinavxd/libredesk/internal/setting/models"
)

func TestEvaluate(t *testing.T) {
	ctx := context.Background()
	classifier := func(reply string) func(string, string) (string, error) {
		return func(string, string) (string, error) { return reply, nil }
	}

	tests := []struct {
		name     string
		s        models.SpamRescueSettings
		c        Checks
		in       Input
		wantRule string
	}{
		{"nothing enabled", models.SpamRescueSettings{}, Checks{}, Input{Email: "a@b.com"}, ""},
		{"trusted email", models.SpamRescueSettings{TrustedEmails: []string{"A@b.com"}}, Checks{}, Input{Email: "a@b.com"}, "trusted_sender"},
		{"trusted domain", models.SpamRescueSettings{TrustedDomains: []string{"@example.org"}}, Checks{}, Input{Email: "x@mail.example.org"}, "trusted_sender"},
		{"domain suffix is not a substring match", models.SpamRescueSettings{TrustedDomains: []string{"example.org"}}, Checks{}, Input{Email: "x@badexample.org"}, ""},
		{"reply to thread", models.SpamRescueSettings{RuleReplyToThread: true},
			Checks{RepliedToOurThread: func([]string) (bool, error) { return true, nil }}, Input{References: []string{"x"}}, "reply_to_thread"},
		{"customer", models.SpamRescueSettings{RuleEcommerceCustomer: true},
			Checks{EcommerceCustomer: func(context.Context, string) (bool, error) { return true, nil }}, Input{Email: "a@b.com"}, "ecommerce_customer"},
		{"order owned by sender", models.SpamRescueSettings{RuleOrderNumber: true, OrderNumberPattern: `\d{5}`},
			Checks{OrderBelongsTo: func(_ context.Context, n, e string) (bool, error) { return n == "12345" && e == "a@b.com", nil }},
			Input{Email: "a@b.com", Body: "where is order 12345?"}, "order_number"},
		{"order owned by someone else", models.SpamRescueSettings{RuleOrderNumber: true, OrderNumberPattern: `\d{5}`},
			Checks{OrderBelongsTo: func(context.Context, string, string) (bool, error) { return false, nil }},
			Input{Email: "a@b.com", Body: "order 12345"}, ""},
		{"bad order pattern", models.SpamRescueSettings{RuleOrderNumber: true, OrderNumberPattern: `(`},
			Checks{OrderBelongsTo: func(context.Context, string, string) (bool, error) { return true, nil }},
			Input{Email: "a@b.com", Body: "12345"}, ""},
		{"ai legitimate above threshold", models.SpamRescueSettings{AIEnabled: true, AIMinConfidence: 0.9},
			Checks{Classify: classifier(`{"verdict":"legitimate","confidence":0.95,"reason":"specific enquiry"}`)}, Input{}, "ai"},
		{"ai legitimate below threshold", models.SpamRescueSettings{AIEnabled: true, AIMinConfidence: 0.9},
			Checks{Classify: classifier(`{"verdict":"legitimate","confidence":0.6,"reason":"maybe"}`)}, Input{}, ""},
		{"ai unsure", models.SpamRescueSettings{AIEnabled: true, AIMinConfidence: 0.5},
			Checks{Classify: classifier(`{"verdict":"unsure","confidence":0.99,"reason":""}`)}, Input{}, ""},
		{"ai fenced json", models.SpamRescueSettings{AIEnabled: true, AIMinConfidence: 0.9},
			Checks{Classify: classifier("```json\n{\"verdict\":\"Legitimate\",\"confidence\":0.92,\"reason\":\"ok\"}\n```")}, Input{}, "ai"},
		{"ai garbage", models.SpamRescueSettings{AIEnabled: true, AIMinConfidence: 0.1},
			Checks{Classify: classifier("legitimate!")}, Input{}, ""},
		{"ai error", models.SpamRescueSettings{AIEnabled: true, AIMinConfidence: 0.1},
			Checks{Classify: func(string, string) (string, error) { return "", errors.New("timeout") }}, Input{}, ""},
		{"ai skipped when a rule already matched", models.SpamRescueSettings{TrustedDomains: []string{"b.com"}, AIEnabled: true, AIMinConfidence: 0.1},
			Checks{Classify: func(string, string) (string, error) { t.Fatal("classifier called"); return "", nil }}, Input{Email: "a@b.com"}, "trusted_sender"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := Evaluate(ctx, tc.s, tc.c, tc.in)
			if d.Rule != tc.wantRule || d.Rescue != (tc.wantRule != "") {
				t.Fatalf("got rule=%q rescue=%v, want rule=%q", d.Rule, d.Rescue, tc.wantRule)
			}
		})
	}
}
