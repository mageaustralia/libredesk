package main

import (
	"fmt"
	"os"
	"time"

	"github.com/abhinavxd/libredesk/internal/crypto"
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

type xoauth2Client struct {
	username string
	token    string
}

func (c *xoauth2Client) Start() (string, []byte, error) {
	return "XOAUTH2", []byte(fmt.Sprintf("user=%s\x01auth=Bearer %s\x01\x01", c.username, c.token)), nil
}

func (c *xoauth2Client) Next(challenge []byte) ([]byte, error) { return nil, nil }

func main() {
	key := os.Getenv("LD_ENC_KEY")
	tokenEnc := os.Getenv("LD_TOKEN_ENC")
	user := os.Getenv("LD_USER")
	mailbox := os.Getenv("LD_MAILBOX")
	subject := os.Getenv("LD_SUBJECT")
	since := os.Getenv("LD_SINCE") // YYYY-MM-DD
	if mailbox == "" {
		mailbox = "[Gmail]/Spam"
	}
	if key == "" || tokenEnc == "" || user == "" {
		fmt.Fprintln(os.Stderr, "missing LD_ENC_KEY / LD_TOKEN_ENC / LD_USER")
		os.Exit(1)
	}

	token, err := crypto.Decrypt(tokenEnc, key)
	if err != nil {
		fmt.Fprintln(os.Stderr, "decrypt failed:", err)
		os.Exit(1)
	}

	client, err := imapclient.DialTLS("imap.gmail.com:993", nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "dial failed:", err)
		os.Exit(1)
	}
	defer client.Logout()

	if err := client.Authenticate(&xoauth2Client{username: user, token: token}); err != nil {
		fmt.Fprintln(os.Stderr, "auth failed:", err)
		os.Exit(1)
	}

	sel, err := client.Select(mailbox, &imap.SelectOptions{ReadOnly: true}).Wait()
	if err != nil {
		fmt.Fprintln(os.Stderr, "select failed:", err)
		os.Exit(1)
	}
	if sel.NumMessages == 0 {
		fmt.Printf("Mailbox %q: empty\n", mailbox)
		return
	}

	var seqSet imap.SeqSet
	if subject != "" || since != "" {
		criteria := &imap.SearchCriteria{}
		if subject != "" {
			criteria.Header = []imap.SearchCriteriaHeaderField{{Key: "Subject", Value: subject}}
		}
		if since != "" {
			t, err := time.Parse("2006-01-02", since)
			if err != nil {
				fmt.Fprintln(os.Stderr, "bad LD_SINCE:", err)
				os.Exit(1)
			}
			criteria.Since = t
		}
		res, err := client.Search(criteria, nil).Wait()
		if err != nil {
			fmt.Fprintln(os.Stderr, "search failed:", err)
			os.Exit(1)
		}
		nums := res.AllSeqNums()
		fmt.Printf("Mailbox %q: %d matches\n", mailbox, len(nums))
		if len(nums) == 0 {
			return
		}
		seqSet.AddNum(nums...)
	} else {
		fmt.Printf("Mailbox %q: %d messages\n", mailbox, sel.NumMessages)
		seqSet.AddRange(1, sel.NumMessages)
	}

	fetchCmd := client.Fetch(seqSet, &imap.FetchOptions{Envelope: true})
	defer fetchCmd.Close()

	for {
		msg := fetchCmd.Next()
		if msg == nil {
			break
		}
		data, err := msg.Collect()
		if err != nil {
			continue
		}
		env := data.Envelope
		if env == nil {
			continue
		}
		from := ""
		if len(env.From) > 0 {
			from = env.From[0].Addr()
			if env.From[0].Name != "" {
				from = env.From[0].Name + " <" + from + ">"
			}
		}
		fmt.Printf("%s | %s | %s\n", env.Date.Format("2006-01-02 15:04"), from, env.Subject)
	}
}
