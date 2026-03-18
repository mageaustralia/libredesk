package inbox

import (
	"crypto/tls"
	"encoding/json"
	"fmt"

	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

// DeleteIMAPMessage connects to the IMAP server for the given inbox,
// searches for a message by Message-ID header, flags it as \Deleted, and expunges.
// Gmail moves IMAP-deleted messages to Trash (auto-purges after 30 days).
func (m *Manager) DeleteIMAPMessage(inboxID int, messageID string) error {
	// Get inbox config.
	dbInbox, err := m.GetDBRecord(inboxID)
	if err != nil {
		return fmt.Errorf("failed to get inbox config: %w", err)
	}

	var cfg imodels.Config
	if err := json.Unmarshal(dbInbox.Config, &cfg); err != nil {
		return fmt.Errorf("failed to parse inbox config: %w", err)
	}

	if len(cfg.IMAP) == 0 {
		return fmt.Errorf("no IMAP config for inbox %d", inboxID)
	}

	imapCfg := cfg.IMAP[0]

	// Connect to IMAP.
	address := fmt.Sprintf("%s:%d", imapCfg.Host, imapCfg.Port)
	imapOptions := &imapclient.Options{
		TLSConfig: &tls.Config{
			InsecureSkipVerify: imapCfg.TLSSkipVerify,
		},
	}

	var client *imapclient.Client
	switch imapCfg.TLSType {
	case "none":
		client, err = imapclient.DialInsecure(address, imapOptions)
	case "starttls":
		client, err = imapclient.DialStartTLS(address, imapOptions)
	case "tls":
		client, err = imapclient.DialTLS(address, imapOptions)
	default:
		return fmt.Errorf("unknown IMAP TLS type: %q", imapCfg.TLSType)
	}
	if err != nil {
		return fmt.Errorf("failed to connect to IMAP: %w", err)
	}
	defer client.Logout()

	// Authenticate.
	if cfg.AuthType == imodels.AuthTypeOAuth2 && cfg.OAuth != nil {
		// For OAuth, we'd need the token refresh logic.
		// For now, fall back to password if available.
		if imapCfg.Password != "" {
			if err := client.Login(imapCfg.Username, imapCfg.Password).Wait(); err != nil {
				return fmt.Errorf("IMAP login failed: %w", err)
			}
		} else {
			return fmt.Errorf("OAuth IMAP delete not yet supported")
		}
	} else {
		if err := client.Login(imapCfg.Username, imapCfg.Password).Wait(); err != nil {
			return fmt.Errorf("IMAP login failed: %w", err)
		}
	}

	// Select mailbox in read-write mode (ReadOnly: false).
	mailbox := imapCfg.Mailbox
	if mailbox == "" {
		mailbox = "INBOX"
	}
	if _, err := client.Select(mailbox, &imap.SelectOptions{ReadOnly: false}).Wait(); err != nil {
		return fmt.Errorf("failed to select mailbox: %w", err)
	}

	// Search for the message by Message-ID header.
	criteria := &imap.SearchCriteria{
		Header: []imap.SearchCriteriaHeaderField{
			{Key: "Message-ID", Value: messageID},
		},
	}

	searchResult, err := client.Search(criteria, nil).Wait()
	if err != nil {
		return fmt.Errorf("IMAP search failed: %w", err)
	}

	seqNums := searchResult.AllSeqNums()
	if len(seqNums) == 0 {
		m.lo.Warn("IMAP message not found for deletion", "message_id", messageID, "inbox_id", inboxID)
		return nil // Not an error — message may have already been deleted or moved.
	}

	// Flag as \Deleted and expunge.
	seqSet := imap.SeqSet{}
	seqSet.AddNum(seqNums...)

	storeFlags := imap.StoreFlags{
		Op:    imap.StoreFlagsAdd,
		Flags: []imap.Flag{imap.FlagDeleted},
	}
	if err := client.Store(seqSet, &storeFlags, nil).Close(); err != nil {
		return fmt.Errorf("failed to flag message as deleted: %w", err)
	}

	if err := client.Expunge().Wait(); err != nil {
		return fmt.Errorf("failed to expunge message: %w", err)
	}

	m.lo.Info("deleted PCI email from IMAP", "message_id", messageID, "inbox_id", inboxID)
	return nil
}
