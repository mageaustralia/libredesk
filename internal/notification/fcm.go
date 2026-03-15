package notifier

import (
	"context"
	"fmt"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/jmoiron/sqlx"
	"github.com/zerodha/logf"
	"google.golang.org/api/option"
)

// FCMSender sends push notifications via Firebase Cloud Messaging.
type FCMSender struct {
	client *messaging.Client
	db     *sqlx.DB
	lo     *logf.Logger
}

// NewFCMSender initializes the FCM sender with a service account key file.
func NewFCMSender(serviceAccountPath string, db *sqlx.DB, lo *logf.Logger) (*FCMSender, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile(serviceAccountPath))
	if err != nil {
		return nil, fmt.Errorf("initializing firebase app: %w", err)
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("initializing firebase messaging: %w", err)
	}

	lo.Info("FCM sender initialized")
	return &FCMSender{client: client, db: db, lo: lo}, nil
}

// pushToken represents a stored FCM token.
type pushToken struct {
	Token    string `db:"token"`
	Platform string `db:"platform"`
}

// SendToUser sends a push notification to all registered devices for the given user.
func (f *FCMSender) SendToUser(userID int, title, body, conversationUUID string) {
	var tokens []pushToken
	if err := f.db.Select(&tokens, "SELECT token, platform FROM user_push_tokens WHERE user_id = $1", userID); err != nil {
		f.lo.Error("error fetching push tokens", "user_id", userID, "error", err)
		return
	}

	if len(tokens) == 0 {
		return
	}

	for _, t := range tokens {
		msg := &messaging.Message{
			Token: t.Token,
			Notification: &messaging.Notification{
				Title: title,
				Body:  body,
			},
			Data: map[string]string{
				"conversation_uuid": conversationUUID,
				"click_action":      "FLUTTER_NOTIFICATION_CLICK",
			},
		}

		// Set platform-specific config.
		if t.Platform == "android" {
			msg.Android = &messaging.AndroidConfig{
				Priority: "high",
				Notification: &messaging.AndroidNotification{
					ChannelID: "libredesk_tickets",
					Sound:     "default",
				},
			}
		} else if t.Platform == "ios" {
			msg.APNS = &messaging.APNSConfig{
				Payload: &messaging.APNSPayload{
					Aps: &messaging.Aps{
						Sound:            "default",
						MutableContent:   true,
						ContentAvailable: true,
					},
				},
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err := f.client.Send(ctx, msg)
		cancel()

		if err != nil {
			// If token is invalid, remove it.
			if messaging.IsUnregistered(err) || messaging.IsInvalidArgument(err) {
				f.lo.Warn("removing invalid FCM token", "user_id", userID, "platform", t.Platform)
				f.db.Exec("DELETE FROM user_push_tokens WHERE user_id = $1 AND token = $2", userID, t.Token)
			} else {
				f.lo.Error("error sending FCM push", "user_id", userID, "platform", t.Platform, "error", err)
			}
		} else {
			f.lo.Debug("FCM push sent", "user_id", userID, "platform", t.Platform)
		}
	}
}
