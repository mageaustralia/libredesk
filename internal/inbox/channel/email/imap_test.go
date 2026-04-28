package email

import (
	"strings"
	"testing"

	"github.com/emersion/go-message/mail"
	"github.com/jhillyerd/enmime"
)

func TestEmail_extractUUIDFromReplyAddress(t *testing.T) {
	e := &Email{}

	testCases := []struct {
		name     string
		address  string
		expected string
	}{
		{
			name:     "Valid reply address with UUID",
			address:  "support+550e8400-e29b-41d4-a716-446655440000@example.com",
			expected: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:     "Reply address with angle brackets",
			address:  "<support+123e4567-e89b-42d3-a456-426614174000@example.com>",
			expected: "123e4567-e89b-42d3-a456-426614174000",
		},
		{
			name:     "No plus sign in address",
			address:  "support@example.com",
			expected: "",
		},
		{
			name:     "Plus sign but no UUID",
			address:  "support+test@example.com",
			expected: "",
		},
		{
			name:     "Invalid UUID format",
			address:  "support+550e8400-e29b-41d4-a716-44665544000X@example.com",
			expected: "550e8400-e29b-41d4-a716-44665544000X", // extractUUIDFromReplyAddress uses simple format check
		},
		{
			name:     "Empty address",
			address:  "",
			expected: "",
		},
		{
			name:     "UUID too short",
			address:  "support+550e8400-e29b-41d4-a716-4466554400@example.com",
			expected: "",
		},
		{
			name:     "UUID too long",
			address:  "support+550e8400-e29b-41d4-a716-4466554400000@example.com",
			expected: "",
		},
		{
			name:     "Multiple plus signs",
			address:  "support+test+550e8400-e29b-41d4-a716-446655440000@example.com",
			expected: "", // "test+550e8400-e29b-41d4-a716-446655440000" is not 36 chars, so validation fails
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := e.extractUUIDFromReplyAddress(tc.address)
			if result != tc.expected {
				t.Errorf("extractUUIDFromReplyAddress(%q) = %q; expected %q", tc.address, result, tc.expected)
			}
		})
	}
}

// TestGoIMAPMessageIDParsing shows how go-imap fails to parse malformed Message-IDs
// and demonstrates the fallback solution.
// go-imap uses mail.Header.MessageID() which strictly follows RFC 5322 and returns
// empty strings for Message-IDs with multiple @ symbols.
//
// This caused emails to be dropped since we require Message-IDs for deduplication.
// References:
// - https://community.mailcow.email/d/701-multiple-at-in-message-id/5
// - https://github.com/emersion/go-message/issues/154#issuecomment-1425634946
func TestGoIMAPMessageIDParsing(t *testing.T) {
	testCases := []struct {
		input            string
		expectedIMAP     string
		expectedFallback string
		name             string
	}{
		{"<normal@example.com>", "normal@example.com", "normal@example.com", "normal message ID"},
		{"<malformed@@example.com>", "", "malformed@@example.com", "double @ - IMAP fails, fallback works"},
		{"<001c01d710db$a8137a50$f83a6ef0$@jones.smith@example.com>", "", "001c01d710db$a8137a50$f83a6ef0$@jones.smith@example.com", "mailcow-style - IMAP fails, fallback works"},
		{"<test@@@domain.com>", "", "test@@@domain.com", "triple @ - IMAP fails, fallback works"},
		{"  <abc123@example.com>  ", "abc123@example.com", "abc123@example.com", "with whitespace - both handle correctly"},
		{"abc123@example.com", "", "abc123@example.com", "no angle brackets - IMAP fails, fallback works"},
		{"", "", "", "empty input"},
		{"<>", "", "", "empty brackets"},
		{"<CAFnQjQFhY8z@mail.example.com@gateway.company.com>", "", "CAFnQjQFhY8z@mail.example.com@gateway.company.com", "gateway-style - IMAP fails, fallback works"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test go-imap parsing behavior
			var h mail.Header
			h.Set("Message-Id", tc.input)
			imapResult, _ := h.MessageID()

			if imapResult != tc.expectedIMAP {
				t.Errorf("IMAP parsing of %q: expected %q, got %q", tc.input, tc.expectedIMAP, imapResult)
			}

			// Test fallback solution
			if tc.input != "" {
				rawEmail := "From: test@example.com\nMessage-ID: " + tc.input + "\n\nBody"
				envelope, err := enmime.ReadEnvelope(strings.NewReader(rawEmail))
				if err != nil {
					t.Fatal(err)
				}

				fallbackResult := extractMessageIDFromHeaders(envelope)
				if fallbackResult != tc.expectedFallback {
					t.Errorf("Fallback extraction of %q: expected %q, got %q", tc.input, tc.expectedFallback, fallbackResult)
				}

				// Critical check: ensure fallback works when IMAP fails
				if imapResult == "" && tc.expectedFallback != "" && fallbackResult == "" {
					t.Errorf("CRITICAL: Both IMAP and fallback failed for %q - would drop email!", tc.input)
				}
			}
		})
	}
}

// TestEdgeCasesMessageID tests additional edge cases for Message-ID extraction.
func TestEdgeCasesMessageID(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected string
	}{
		{
			name: "no Message-ID header",
			email: `From: test@example.com
To: inbox@test.com
Subject: Test

Body`,
			expected: "",
		},
		{
			name: "malformed header syntax",
			email: `From: test@example.com
Message-ID: malformed-no-brackets@@domain.com
To: inbox@test.com

Body`,
			expected: "malformed-no-brackets@@domain.com",
		},
		{
			name: "multiple Message-ID headers (first wins)",
			email: `From: test@example.com
Message-ID: <first@example.com>
Message-ID: <second@@example.com>
To: inbox@test.com

Body`,
			expected: "first@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envelope, err := enmime.ReadEnvelope(strings.NewReader(tt.email))
			if err != nil {
				t.Fatal(err)
			}

			result := extractMessageIDFromHeaders(envelope)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestExtractDSNDiagnostic verifies that the diagnostic and original-message
// parts of an RFC 3464 multipart/report bounce are surfaced into the displayed
// body. Without this, a bounce only shows the friendly summary and the agent
// has no way to see the SMTP error code or which original message bounced.
func TestExtractDSNDiagnostic(t *testing.T) {
	// Realistic Amazon SES-shaped DSN. CRLF line endings are required for
	// enmime to parse the multipart structure correctly.
	dsn := "From: mailer-daemon@amazonses.com\r\n" +
		"To: orders@tenniswarehouse.com.au\r\n" +
		"Subject: Delivery Status Notification (Failure)\r\n" +
		"Message-ID: <bounce-test@amazonses.com>\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: multipart/report; report-type=delivery-status; boundary=\"bound42\"\r\n" +
		"\r\n" +
		"--bound42\r\n" +
		"Content-Type: text/plain; charset=us-ascii\r\n" +
		"\r\n" +
		"An error occurred while trying to deliver the mail to the following recipients:\r\n" +
		"info@meristpor.com.tr\r\n" +
		"\r\n" +
		"--bound42\r\n" +
		"Content-Type: message/delivery-status\r\n" +
		"\r\n" +
		"Reporting-MTA: dns; a8-23.smtp-out.amazonses.com\r\n" +
		"\r\n" +
		"Final-Recipient: rfc822; info@meristpor.com.tr\r\n" +
		"Action: failed\r\n" +
		"Status: 5.1.1\r\n" +
		"Diagnostic-Code: smtp; 550 5.1.1 <info@meristpor.com.tr>: User unknown\r\n" +
		"\r\n" +
		"--bound42\r\n" +
		"Content-Type: message/rfc822-headers\r\n" +
		"\r\n" +
		"From: orders@tenniswarehouse.com.au\r\n" +
		"To: info@meristpor.com.tr\r\n" +
		"Subject: Order Confirmation #12345\r\n" +
		"Message-ID: <original-12345@tenniswarehouse.com.au>\r\n" +
		"\r\n" +
		"--bound42--\r\n"

	envelope, err := enmime.ReadEnvelope(strings.NewReader(dsn))
	if err != nil {
		t.Fatal(err)
	}

	got := extractDSNDiagnostic(envelope)

	wantSubstrings := []string{
		"550 5.1.1",                        // SMTP diagnostic code
		"info@meristpor.com.tr",            // failed recipient
		"Status: 5.1.1",                    // RFC 3464 status code
		"Order Confirmation #12345",        // original message subject
		"original-12345@tenniswarehouse.com.au", // original Message-ID
	}
	for _, want := range wantSubstrings {
		if !strings.Contains(got, want) {
			t.Errorf("extractDSNDiagnostic missing %q\nfull output:\n%s", want, got)
		}
	}
}

// TestExtractDSNDiagnostic_NotADSN verifies that ordinary emails return an
// empty diagnostic so we don't pollute regular message bodies.
func TestExtractDSNDiagnostic_NotADSN(t *testing.T) {
	plain := "From: alice@example.com\r\n" +
		"To: bob@example.com\r\n" +
		"Subject: Hello\r\n" +
		"Content-Type: text/plain; charset=us-ascii\r\n" +
		"\r\n" +
		"Just a regular email.\r\n"

	envelope, err := enmime.ReadEnvelope(strings.NewReader(plain))
	if err != nil {
		t.Fatal(err)
	}

	if got := extractDSNDiagnostic(envelope); got != "" {
		t.Errorf("expected empty diagnostic for non-DSN email, got %q", got)
	}
}

// TestIsAutoReply_KeepsDSN verifies that bounces (which set
// Auto-Submitted: auto-replied per RFC 3464) are NOT classified as auto-replies
// and so will be ingested. Without this, every bounce gets silently dropped
// and agents have no record of failed sends.
func TestIsAutoReply_KeepsDSN(t *testing.T) {
	dsn := "From: mailer-daemon@example.net\r\n" +
		"To: support@example.com\r\n" +
		"Subject: Delivery Status Notification (Failure)\r\n" +
		"Auto-Submitted: auto-replied\r\n" +
		"Content-Type: multipart/report; report-type=delivery-status; boundary=\"b\"\r\n" +
		"\r\n--b\r\nContent-Type: text/plain\r\n\r\nAddress not found\r\n--b--\r\n"
	envelope, err := enmime.ReadEnvelope(strings.NewReader(dsn))
	if err != nil {
		t.Fatal(err)
	}
	if isAutoReply(envelope) {
		t.Error("DSN bounce was classified as auto-reply; bounces must be ingested for diagnostic visibility")
	}
}

// TestIsAutoReply_SkipsVacationResponder verifies that genuine auto-replies
// (out-of-office, vacation responders) are still skipped — the fix should
// only carve out DSNs, not weaken the filter.
func TestIsAutoReply_SkipsVacationResponder(t *testing.T) {
	vacation := "From: ooo@example.net\r\n" +
		"To: support@example.com\r\n" +
		"Subject: Out of office\r\n" +
		"Auto-Submitted: auto-replied\r\n" +
		"Content-Type: text/plain\r\n" +
		"\r\nI'm on holiday until next week.\r\n"
	envelope, err := enmime.ReadEnvelope(strings.NewReader(vacation))
	if err != nil {
		t.Fatal(err)
	}
	if !isAutoReply(envelope) {
		t.Error("vacation responder was NOT classified as auto-reply; the filter is too lax")
	}
}
