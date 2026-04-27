package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/mail"
	smtplib "net/smtp"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/envelope"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/abhinavxd/libredesk/internal/setting/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// testInboxRequest is the body for POST /api/v1/inboxes/test-connection.
type testInboxRequest struct {
	IMAP      *imodels.IMAPConfig `json:"imap"`
	SMTP      *imodels.SMTPConfig `json:"smtp"`
	AuthType  string              `json:"auth_type"`
	TestEmail string              `json:"test_email"`
}

// testInboxResponse is returned from POST /api/v1/inboxes/test-connection.
type testInboxResponse struct {
	Success  bool     `json:"success"`
	IMAPLogs []string `json:"imap_logs"`
	SMTPLogs []string `json:"smtp_logs"`
}

// testEmailRequest is the body for POST /api/v1/settings/notifications/email/test.
type testEmailRequest struct {
	models.EmailNotification
	TestEmail string `json:"test_email"`
}

// testEmailResponse is returned from POST /api/v1/settings/notifications/email/test.
type testEmailResponse struct {
	Success bool     `json:"success"`
	Logs    []string `json:"logs"`
}

// handleTestInboxConnection tests IMAP and/or SMTP connection with the provided config.
func handleTestInboxConnection(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = testInboxRequest{}
	)

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}

	resp := testInboxResponse{Success: true}

	// Test IMAP if config provided.
	if req.IMAP != nil && req.IMAP.Host != "" {
		imapLogs, imapOK := testIMAPConnection(req.IMAP)
		resp.IMAPLogs = imapLogs
		if !imapOK {
			resp.Success = false
		}
	}

	// Test SMTP if config provided.
	if req.SMTP != nil && req.SMTP.Host != "" {
		smtpLogs, smtpOK := testSMTPConn(req.SMTP, req.TestEmail)
		resp.SMTPLogs = smtpLogs
		if !smtpOK {
			resp.Success = false
		}
	}

	return r.SendEnvelope(resp)
}

// handleTestEmailNotificationSettings tests the notification email SMTP settings.
func handleTestEmailNotificationSettings(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		req  = testEmailRequest{}
		cur  = models.EmailNotification{}
		logs = []string{}
	)

	addLog := func(msg string) {
		logs = append(logs, fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg))
	}

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}

	// Validate test email.
	if req.TestEmail == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Test email address is required", nil, envelope.InputError)
	}
	if _, err := mail.ParseAddress(req.TestEmail); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid test email address", nil, envelope.InputError)
	}

	addLog(fmt.Sprintf("Starting SMTP test to %s", req.TestEmail))

	// Fetch current stored settings to retrieve password when dummy is passed.
	out, err := app.setting.GetByPrefix("notification.email")
	if err != nil {
		addLog(fmt.Sprintf("Error fetching current settings: %v", err))
		return r.SendEnvelope(testEmailResponse{Success: false, Logs: logs})
	}
	if err := json.Unmarshal(out, &cur); err != nil {
		addLog(fmt.Sprintf("Error parsing current settings: %v", err))
		return r.SendEnvelope(testEmailResponse{Success: false, Logs: logs})
	}

	// Use the stored password when the request carries a dummy placeholder.
	password := req.Password
	if password == "" || strings.Contains(password, stringutil.PasswordDummy) {
		password = cur.Password
	}

	serverAddr := fmt.Sprintf("%s:%d", req.Host, req.Port)
	addLog(fmt.Sprintf("Connecting to SMTP server: %s", serverAddr))

	tlsConfig := &tls.Config{
		ServerName:         req.Host,
		InsecureSkipVerify: req.TLSSkipVerify,
	}

	var client *smtplib.Client

	switch req.TLSType {
	case "tls":
		addLog("Using SSL/TLS connection")
		conn, err := tls.Dial("tcp", serverAddr, tlsConfig)
		if err != nil {
			addLog(fmt.Sprintf("TLS connection failed: %v", err))
			return r.SendEnvelope(testEmailResponse{Success: false, Logs: logs})
		}
		defer conn.Close()
		client, err = smtplib.NewClient(conn, req.Host)
		if err != nil {
			addLog(fmt.Sprintf("Failed to create SMTP client: %v", err))
			return r.SendEnvelope(testEmailResponse{Success: false, Logs: logs})
		}
	default:
		addLog("Using plain/STARTTLS connection")
		client, err = smtplib.Dial(serverAddr)
		if err != nil {
			addLog(fmt.Sprintf("Connection failed: %v", err))
			return r.SendEnvelope(testEmailResponse{Success: false, Logs: logs})
		}
	}
	defer client.Close()
	addLog("Connected successfully")

	// EHLO.
	hostname := req.HelloHostname
	if hostname == "" {
		hostname = "localhost"
	}
	addLog(fmt.Sprintf("Sending EHLO %s", hostname))
	if err := client.Hello(hostname); err != nil {
		addLog(fmt.Sprintf("EHLO failed: %v", err))
		return r.SendEnvelope(testEmailResponse{Success: false, Logs: logs})
	}

	// STARTTLS if required.
	if req.TLSType == "starttls" {
		addLog("Starting TLS (STARTTLS)")
		if err := client.StartTLS(tlsConfig); err != nil {
			addLog(fmt.Sprintf("STARTTLS failed: %v", err))
			return r.SendEnvelope(testEmailResponse{Success: false, Logs: logs})
		}
		addLog("TLS connection established")
	}

	// Authenticate if credentials provided.
	if req.Username != "" && password != "" {
		addLog(fmt.Sprintf("Authenticating as %s using %s", req.Username, req.AuthProtocol))
		var auth smtplib.Auth
		switch req.AuthProtocol {
		case "plain":
			auth = smtplib.PlainAuth("", req.Username, password, req.Host)
		case "login":
			auth = &loginAuth{username: req.Username, password: password}
		case "cram":
			auth = smtplib.CRAMMD5Auth(req.Username, password)
		case "none", "":
			addLog("No authentication required")
		default:
			auth = smtplib.PlainAuth("", req.Username, password, req.Host)
		}
		if auth != nil {
			if err := client.Auth(auth); err != nil {
				addLog(fmt.Sprintf("Authentication failed: %v", err))
				return r.SendEnvelope(testEmailResponse{Success: false, Logs: logs})
			}
			addLog("Authentication successful")
		}
	}

	// Send test email.
	fromAddr := req.EmailAddress
	if fromAddr == "" {
		fromAddr = req.Username
	}
	addLog(fmt.Sprintf("Setting sender: %s", fromAddr))
	if err := client.Mail(fromAddr); err != nil {
		addLog(fmt.Sprintf("MAIL FROM failed: %v", err))
		return r.SendEnvelope(testEmailResponse{Success: false, Logs: logs})
	}
	addLog(fmt.Sprintf("Setting recipient: %s", req.TestEmail))
	if err := client.Rcpt(req.TestEmail); err != nil {
		addLog(fmt.Sprintf("RCPT TO failed: %v", err))
		return r.SendEnvelope(testEmailResponse{Success: false, Logs: logs})
	}
	addLog("Sending test message")
	w, err := client.Data()
	if err != nil {
		addLog(fmt.Sprintf("DATA command failed: %v", err))
		return r.SendEnvelope(testEmailResponse{Success: false, Logs: logs})
	}
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: LibreDesk SMTP Test\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nThis is a test email from LibreDesk to verify your SMTP notification settings are working correctly.\r\n\r\nSent at: %s",
		fromAddr, req.TestEmail, time.Now().Format(time.RFC1123))
	if _, err := w.Write([]byte(msg)); err != nil {
		addLog(fmt.Sprintf("Failed to write message: %v", err))
		return r.SendEnvelope(testEmailResponse{Success: false, Logs: logs})
	}
	if err := w.Close(); err != nil {
		addLog(fmt.Sprintf("Failed to close message: %v", err))
		return r.SendEnvelope(testEmailResponse{Success: false, Logs: logs})
	}
	addLog("Test email sent successfully!")
	client.Quit()
	return r.SendEnvelope(testEmailResponse{Success: true, Logs: logs})
}

// loginAuth implements smtp.Auth for LOGIN authentication.
type loginAuth struct {
	username, password string
}

func (a *loginAuth) Start(server *smtplib.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte{}, nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch string(fromServer) {
		case "Username:":
			return []byte(a.username), nil
		case "Password:":
			return []byte(a.password), nil
		}
	}
	return nil, nil
}

// testIMAPConnection dials the IMAP server and tries to authenticate.
// Credentials are never echoed in logs.
func testIMAPConnection(cfg *imodels.IMAPConfig) ([]string, bool) {
	logs := []string{}
	addLog := func(msg string) {
		logs = append(logs, fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg))
	}

	address := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	addLog(fmt.Sprintf("Connecting to IMAP server: %s", address))

	imapOptions := &imapclient.Options{
		TLSConfig: &tls.Config{
			InsecureSkipVerify: cfg.TLSSkipVerify,
		},
	}

	var (
		client *imapclient.Client
		err    error
	)

	switch cfg.TLSType {
	case "none":
		addLog("Using plain connection (no TLS)")
		client, err = imapclient.DialInsecure(address, imapOptions)
	case "starttls":
		addLog("Using STARTTLS connection")
		client, err = imapclient.DialStartTLS(address, imapOptions)
	case "tls":
		addLog("Using SSL/TLS connection")
		client, err = imapclient.DialTLS(address, imapOptions)
	default:
		addLog(fmt.Sprintf("Unknown TLS type: %s", cfg.TLSType))
		return logs, false
	}

	if err != nil {
		addLog(fmt.Sprintf("Connection failed: %v", err))
		return logs, false
	}
	defer client.Logout()
	addLog("Connected successfully")

	// Authenticate.
	addLog(fmt.Sprintf("Authenticating as: %s", cfg.Username))
	if err := client.Login(cfg.Username, cfg.Password).Wait(); err != nil {
		addLog(fmt.Sprintf("Authentication failed: %v", err))
		return logs, false
	}
	addLog("Authentication successful")

	// Select mailbox.
	addLog(fmt.Sprintf("Selecting mailbox: %s", cfg.Mailbox))
	mbox, err := client.Select(cfg.Mailbox, &imap.SelectOptions{ReadOnly: true}).Wait()
	if err != nil {
		addLog(fmt.Sprintf("Failed to select mailbox: %v", err))
		return logs, false
	}
	addLog(fmt.Sprintf("Mailbox selected — %d messages", mbox.NumMessages))
	addLog("IMAP test completed successfully!")
	return logs, true
}

// testSMTPConn dials the SMTP server and tries to authenticate.
// A test email is sent only when testEmail is non-empty.
// Credentials are never echoed in logs.
func testSMTPConn(cfg *imodels.SMTPConfig, testEmail string) ([]string, bool) {
	logs := []string{}
	addLog := func(msg string) {
		logs = append(logs, fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg))
	}

	serverAddr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	addLog(fmt.Sprintf("Connecting to SMTP server: %s", serverAddr))

	tlsConfig := &tls.Config{
		ServerName:         cfg.Host,
		InsecureSkipVerify: cfg.TLSSkipVerify,
	}

	var (
		client *smtplib.Client
		err    error
	)

	switch cfg.TLSType {
	case "tls":
		addLog("Using SSL/TLS connection")
		conn, connErr := tls.Dial("tcp", serverAddr, tlsConfig)
		if connErr != nil {
			addLog(fmt.Sprintf("TLS connection failed: %v", connErr))
			return logs, false
		}
		defer conn.Close()
		client, err = smtplib.NewClient(conn, cfg.Host)
		if err != nil {
			addLog(fmt.Sprintf("Failed to create SMTP client: %v", err))
			return logs, false
		}
	default:
		addLog("Using plain/STARTTLS connection")
		client, err = smtplib.Dial(serverAddr)
		if err != nil {
			addLog(fmt.Sprintf("Connection failed: %v", err))
			return logs, false
		}
	}
	defer client.Close()
	addLog("Connected successfully")

	// EHLO.
	hostname := cfg.HelloHostname
	if hostname == "" {
		hostname = "localhost"
	}
	addLog(fmt.Sprintf("Sending EHLO %s", hostname))
	if err := client.Hello(hostname); err != nil {
		addLog(fmt.Sprintf("EHLO failed: %v", err))
		return logs, false
	}

	// STARTTLS if required.
	if cfg.TLSType == "starttls" {
		addLog("Starting TLS (STARTTLS)")
		if err := client.StartTLS(tlsConfig); err != nil {
			addLog(fmt.Sprintf("STARTTLS failed: %v", err))
			return logs, false
		}
		addLog("TLS connection established")
	}

	// Authenticate if credentials provided.
	if cfg.Username != "" && cfg.Password != "" {
		addLog(fmt.Sprintf("Authenticating as %s using %s", cfg.Username, cfg.AuthProtocol))
		var auth smtplib.Auth
		switch cfg.AuthProtocol {
		case "plain":
			auth = smtplib.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
		case "login":
			auth = &loginAuth{username: cfg.Username, password: cfg.Password}
		case "cram":
			auth = smtplib.CRAMMD5Auth(cfg.Username, cfg.Password)
		case "none", "":
			addLog("No authentication required")
		}
		if auth != nil {
			if err := client.Auth(auth); err != nil {
				addLog(fmt.Sprintf("Authentication failed: %v", err))
				return logs, false
			}
			addLog("Authentication successful")
		}
	}

	// Optionally send a test email.
	if testEmail != "" {
		fromAddr := cfg.Username
		addLog(fmt.Sprintf("Sending test email to %s", testEmail))
		if err := client.Mail(fromAddr); err != nil {
			addLog(fmt.Sprintf("MAIL FROM failed: %v", err))
			return logs, false
		}
		if err := client.Rcpt(testEmail); err != nil {
			addLog(fmt.Sprintf("RCPT TO failed: %v", err))
			return logs, false
		}
		w, err := client.Data()
		if err != nil {
			addLog(fmt.Sprintf("DATA command failed: %v", err))
			return logs, false
		}
		msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: LibreDesk Inbox SMTP Test\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nThis is a test email from LibreDesk inbox SMTP configuration.\r\nSent at: %s",
			fromAddr, testEmail, time.Now().Format(time.RFC1123))
		if _, err := w.Write([]byte(msg)); err != nil {
			addLog(fmt.Sprintf("Failed to write message: %v", err))
			return logs, false
		}
		if err := w.Close(); err != nil {
			addLog(fmt.Sprintf("Failed to close message: %v", err))
			return logs, false
		}
		addLog("Test email sent successfully!")
	}

	addLog("SMTP test completed successfully!")
	client.Quit()
	return logs, true
}
