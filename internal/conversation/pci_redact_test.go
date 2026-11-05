package conversation

import (
	"testing"

	pciscrub "github.com/mageaustralia/go-pci-scrub"
)

func TestContainsCardholderData(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{
			name: "payment notice with authorisation expiry date",
			text: "Customer has authorised a payment to you of $120.00 AUD.\nThis authorisation is valid until 28 November 2026.\nAuthorisation expiry date: 28 November 2026 9:30:00 PM AEDT",
			want: false,
		},
		{
			name: "card number",
			text: "please use my card 4111 1111 1111 1111 thanks",
			want: true,
		},
		{
			name: "card number with expiry and cvv",
			text: "card number 4111111111111111 expiry 03/29 cvv 123",
			want: true,
		},
		{
			name: "cvv keyword with payment context",
			text: "payment details: security code 123",
			want: true,
		},
		{
			name: "plain message",
			text: "Where is my order? Thanks",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsCardholderData(pciscrub.ScrubWithSpans(tt.text).Spans)
			if got != tt.want {
				t.Fatalf("containsCardholderData = %v, want %v", got, tt.want)
			}
		})
	}
}
