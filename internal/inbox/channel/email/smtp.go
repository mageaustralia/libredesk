package email

import (
	"crypto/tls"
	"fmt"
	"math/rand"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/knadh/smtppool"
)

const (
	headerReturnPath              = "Return-Path"
	headerMessageID               = "Message-ID"
	headerReferences              = "References"
	headerInReplyTo               = "In-Reply-To"
	headerLibredeskLoopPrevention = "X-Libredesk-Loop-Prevention"
	headerLibredeskConversationID = "X-Libredesk-Conversation-UUID"
	headerAutoreply               = "X-Autoreply"
	headerAutoSubmitted           = "Auto-Submitted"

	dispositionInline = "inline"
)

// NewSmtpPool returns a smtppool
func NewSmtpPool(configs []imodels.SMTPConfig, oauth *imodels.OAuthConfig) ([]*smtppool.Pool, error) {
	pools := make([]*smtppool.Pool, 0, len(configs))

	for _, cfg := range configs {
		var auth smtp.Auth

		// Check if OAuth authentication should be used
		if oauth != nil && oauth.AccessToken != "" {
			auth = &XOAuth2SMTPAuth{
				Username: cfg.Username,
				Token:    oauth.AccessToken,
			}
		} else {
			// Use traditional authentication methods
			switch cfg.AuthProtocol {
			case "cram":
				auth = smtp.CRAMMD5Auth(cfg.Username, cfg.Password)
			case "plain":
				auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
			case "login":
				auth = &smtppool.LoginAuth{Username: cfg.Username, Password: cfg.Password}
			case "", "none":
				// No authentication
			default:
				return nil, fmt.Errorf("unknown SMTP auth type '%s'", cfg.AuthProtocol)
			}
		}
		cfg.Auth = auth

		// TLS config
		if cfg.TLSType != "none" {
			cfg.TLSConfig = &tls.Config{}
			if cfg.TLSSkipVerify {
				cfg.TLSConfig.InsecureSkipVerify = cfg.TLSSkipVerify
			} else {
				cfg.TLSConfig.ServerName = cfg.Host
			}

			// SSL/TLS, not STARTTLS
			if cfg.TLSType == "tls" {
				cfg.SSL = true
			}
		}

		// Parse timeouts.
		idleTimeout, err := time.ParseDuration(cfg.IdleTimeout)
		if err != nil {
			idleTimeout = 30 * time.Second
		}
		poolWaitTimeout, err := time.ParseDuration(cfg.PoolWaitTimeout)
		if err != nil {
			poolWaitTimeout = 40 * time.Second
		}

		pool, err := smtppool.New(smtppool.Opt{
			Host:              cfg.Host,
			Port:              cfg.Port,
			HelloHostname:     cfg.HelloHostname,
			MaxConns:          cfg.MaxConns,
			MaxMessageRetries: cfg.MaxMessageRetries,
			IdleTimeout:       idleTimeout,
			PoolWaitTimeout:   poolWaitTimeout,
			SSL:               cfg.SSL,
			Auth:              cfg.Auth,
			TLSConfig:         cfg.TLSConfig,
		})
		if err != nil {
			return nil, err
		}
		pools = append(pools, pool)
	}

	return pools, nil
}

// Send sends an email using one of the configured SMTP servers.
func (e *Email) Send(m models.Message) error {
	// Refresh OAuth token if needed
	oauthConfig, _, err := e.refreshOAuthIfNeeded()
	if err != nil {
		return err
	}

	// Recreate SMTP pools if token changed (handles both: we refreshed or IMAP refreshed)
	if e.authType == imodels.AuthTypeOAuth2 && oauthConfig != nil {
		e.smtpPoolsMu.Lock()
		if e.smtpPoolsToken != oauthConfig.AccessToken {
			// Close existing pools
			for _, p := range e.smtpPools {
				p.Close()
			}

			// Create new pools with current token
			newPools, err := NewSmtpPool(e.smtpCfg, oauthConfig)
			if err != nil {
				e.smtpPoolsMu.Unlock()
				e.lo.Error("Failed to recreate SMTP pools after token refresh", "inbox_id", e.Identifier(), "error", err)
				return fmt.Errorf("failed to recreate SMTP pools: %w", err)
			}
			e.smtpPools = newPools
			e.smtpPoolsToken = oauthConfig.AccessToken
		}
		e.smtpPoolsMu.Unlock()
	}

	// Select a random SMTP server if there are multiple
	e.smtpPoolsMu.RLock()
	var (
		serverCount = len(e.smtpPools)
		server      *smtppool.Pool
	)
	if serverCount > 1 {
		server = e.smtpPools[rand.Intn(serverCount)]
	} else {
		server = e.smtpPools[0]
	}
	e.smtpPoolsMu.RUnlock()

	// Prepare attachments if there are any
	var attachments []smtppool.Attachment
	if m.Attachments != nil {
		attachments = make([]smtppool.Attachment, 0, len(m.Attachments))
		for _, file := range m.Attachments {
			attachment := smtppool.Attachment{
				Filename: file.Name,
				Header:   file.Header,
				Content:  make([]byte, len(file.Content)),
			}
			copy(attachment.Content, file.Content)
			attachments = append(attachments, attachment)
		}
	}

	email := smtppool.Email{
		From:        m.From,
		To:          m.To,
		Cc:          m.CC,
		Bcc:         m.BCC,
		Subject:     m.Subject,
		Attachments: attachments,
		Headers:     textproto.MIMEHeader{},
	}

	// Set libredesk loop prevention header to from address.
	emailAddress, err := stringutil.ExtractEmail(m.From)
	if err != nil {
		e.lo.Error("Failed to extract email address from the 'From' header", "error", err)
		return fmt.Errorf("failed to extract email address from 'From' header: %w", err)
	}
	email.Headers.Set(headerLibredeskLoopPrevention, emailAddress)

	// Set Reply-To with plus-addressing for conversation matching (if enabled)
	// e.g., support@company.com → support+conv-{uuid}@company.com
	if e.enablePlusAddressing && m.ConversationUUID != "" {
		replyToAddr := buildPlusAddress(emailAddress, m.ConversationUUID)
		email.Headers.Set("Reply-To", replyToAddr)
		e.lo.Debug("Reply-To header set with plus-addressing", "reply_to", replyToAddr)
	}

	// Attach SMTP level headers
	for key, value := range e.headers {
		email.Headers.Set(key, value)
	}

	// Attach email level headers
	for key, value := range m.Headers {
		email.Headers.Set(key, value[0])
	}

	// Set In-Reply-To header
	if m.InReplyTo != "" {
		email.Headers.Set(headerInReplyTo, "<"+m.InReplyTo+">")
		e.lo.Debug("In-Reply-To header set", "message_id", m.InReplyTo)
	}

	// Set message id header
	if m.SourceID.String != "" {
		email.Headers.Set(headerMessageID, fmt.Sprintf("<%s>", m.SourceID.String))
		e.lo.Debug("Message-ID header set", "message_id", m.SourceID.String)
	}

	// Set references header
	var references string
	for _, ref := range m.References {
		references += "<" + ref + "> "
	}
	email.Headers.Set(headerReferences, references)

	e.lo.Debug("References header set", "references", references)

	// Set conversation uuid header
	if m.ConversationUUID != "" {
		email.Headers.Set(headerLibredeskConversationID, m.ConversationUUID)
		e.lo.Debug("Conversation UUID header set", "conversation_uuid", m.ConversationUUID)
	}

	// Set email content
	switch m.ContentType {
	case "plain":
		email.Text = []byte(m.Content)
	default:
		// Process HTML for email clients.
		// TipTap wraps each line in <p> tags. In email clients, <p> tags have default
		// margins (~1em) which makes single-Enter look double-spaced.
		// Strategy:
		// - Body text: flatten <p> to <br> so single-Enter = next line,
		//   empty <p> (double-Enter) = blank line (paragraph gap).
		// - Signature: leave <p> tags untouched so they keep default email
		//   client margins (proper paragraph spacing between name, company, etc.)
		htmlContent := m.Content
		htmlContent = processEmailHTML(htmlContent)
		email.HTML = []byte(htmlContent)
		if len(m.AltContent) > 0 {
			email.Text = []byte(m.AltContent)
		}
	}
	return server.Send(email)
}


// processEmailHTML converts TipTap paragraph HTML to email-friendly HTML.
// Body <p> tags are flattened to <br> (single-Enter = next line).
// Empty <p> tags (double-Enter) become a blank line gap.
// Signature <p> tags are left intact so email clients render paragraph spacing.
func processEmailHTML(html string) string {
	const sigMarker = `<div class="email-signature"`

	// Split body from signature
	sigIdx := strings.Index(html, sigMarker)
	var body, signature string
	if sigIdx >= 0 {
		body = html[:sigIdx]
		signature = html[sigIdx:]
	} else {
		body = html
		signature = ""
	}

	// Process body: flatten <p> tags into <br> line breaks.
	// 1. Mark empty paragraphs (double-Enter) - these become blank line gaps
	body = strings.ReplaceAll(body, "<p></p>", "<!--BLANK-->")
	body = strings.ReplaceAll(body, "<p><br></p>", "<!--BLANK-->")
	body = strings.ReplaceAll(body, "<p><br/></p>", "<!--BLANK-->")

	// 2. Remove opening <p> and convert closing </p> to <br>
	//    This turns <p>line1</p><p>line2</p> into line1<br>line2<br>
	body = strings.ReplaceAll(body, "</p><p>", "<br>")
	body = strings.ReplaceAll(body, "<p>", "")
	body = strings.ReplaceAll(body, "</p>", "")

	// 3. Restore blank line markers as <br><br> (visible gap)
	body = strings.ReplaceAll(body, "<!--BLANK-->", "<br><br>")

	// 4. Clean up trailing whitespace
	body = strings.TrimRight(body, " \n\t")

	// 5. If there's a signature, add inline styles for email clients
	//    (CSS classes don't work in email - must use inline styles)
	if signature != "" {
		// Add margin-top to signature div for visual separation from body
		signature = strings.Replace(signature,
			`<div class="email-signature"`,
			`<div class="email-signature" style="margin-top:1em;padding-top:0.75em;border-top:1px solid #e5e7eb"`,
			1)
	}

	return body + signature
}

// buildPlusAddress creates a plus-addressed email for conversation matching.
// e.g., support@company.com + uuid → support+conv-{uuid}@company.com
func buildPlusAddress(email, conversationUUID string) string {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 {
		return email // fallback to original if invalid format
	}
	return fmt.Sprintf("%s+conv-%s@%s", parts[0], conversationUUID, parts[1])
}
