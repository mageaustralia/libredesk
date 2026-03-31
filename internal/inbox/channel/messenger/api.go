package messenger

import (
	"bytes"
	"encoding/json"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"
)

const graphAPIBase = "https://graph.facebook.com/v22.0"

// MetaConfig holds the configuration for a Messenger/Instagram inbox.
// Stored encrypted in the inboxes.config JSONB column.
type MetaConfig struct {
	PageID          string `json:"page_id"`
	PageAccessToken string `json:"page_access_token"` // encrypted at rest
	AppSecret       string `json:"app_secret"`         // for webhook signature verification
	VerifyToken     string `json:"verify_token"`        // for webhook subscription handshake
	IGAccountID     string `json:"ig_account_id"`       // Instagram only
	AutoAssignOnReply bool `json:"auto_assign_on_reply"`
}

// sendAPIPayload is the request body for the Send API.
type sendAPIPayload struct {
	Recipient  recipient   `json:"recipient"`
	Message    sendMessage `json:"message,omitempty"`
	Attachment *sendAttach `json:"attachment,omitempty"`
}

type recipient struct {
	ID string `json:"id"`
}

type sendMessage struct {
	Text       string          `json:"text,omitempty"`
	Attachment *sendAttachBody `json:"attachment,omitempty"`
}

type sendAttach struct {
	Type    string         `json:"type"`
	Payload attachPayload  `json:"payload"`
}

type sendAttachBody struct {
	Type    string        `json:"type"`
	Payload attachPayload `json:"payload"`
}

type attachPayload struct {
	URL string `json:"url"`
}

// sendAPIResponse is the response from the Send API.
type sendAPIResponse struct {
	RecipientID string `json:"recipient_id"`
	MessageID   string `json:"message_id"`
	Error       *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    int    `json:"code"`
	} `json:"error"`
}

// UserProfile holds profile info from the Graph API.
type UserProfile struct {
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	ProfilePic string `json:"profile_pic"`
}

var httpClient = &http.Client{Timeout: 30 * time.Second}

// SendTextMessage sends a plain text message via the Send API.
func SendTextMessage(pageToken, recipientID, text string) (string, error) {
	payload := map[string]interface{}{
		"recipient":    map[string]string{"id": recipientID},
		"message":      map[string]string{"text": text},
		"messaging_type": "RESPONSE",
	}
	return callSendAPI(pageToken, payload)
}

// SendAttachment sends an attachment (image, video, file, audio) via the Send API.
func SendAttachment(pageToken, recipientID, attachType, url string) (string, error) {
	payload := map[string]interface{}{
		"recipient": map[string]string{"id": recipientID},
		"message": map[string]interface{}{
			"attachment": map[string]interface{}{
				"type": attachType,
				"payload": map[string]string{
					"url":         url,
					"is_reusable": "true",
				},
			},
		},
		"messaging_type": "RESPONSE",
	}
	return callSendAPI(pageToken, payload)
}

// callSendAPI makes a POST to the Send API and returns the message ID.
func callSendAPI(pageToken string, payload interface{}) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshalling send payload: %w", err)
	}

	url := fmt.Sprintf("%s/me/messages?access_token=%s", graphAPIBase, pageToken)
	resp, err := httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("calling Send API: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading Send API response: %w", err)
	}

	var result sendAPIResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parsing Send API response: %w", err)
	}
	if result.Error != nil {
		return "", fmt.Errorf("Send API error %d: %s (%s)", result.Error.Code, result.Error.Message, result.Error.Type)
	}

	return result.MessageID, nil
}

// GetUserProfile fetches a user's profile from the Graph API.
func GetUserProfile(pageToken, userID string) (UserProfile, error) {
	url := fmt.Sprintf("%s/%s?fields=first_name,last_name,profile_pic&access_token=%s", graphAPIBase, userID, pageToken)
	resp, err := httpClient.Get(url)
	if err != nil {
		return UserProfile{}, fmt.Errorf("fetching user profile: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return UserProfile{}, fmt.Errorf("reading profile response: %w", err)
	}

	var profile UserProfile
	if err := json.Unmarshal(body, &profile); err != nil {
		return UserProfile{}, fmt.Errorf("parsing profile response: %w", err)
	}
	return profile, nil
}

// DownloadAttachment downloads a file from a temporary Meta attachment URL.
func DownloadAttachment(url string) ([]byte, string, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, "", fmt.Errorf("downloading attachment: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("reading attachment: %w", err)
	}

	contentType := resp.Header.Get("Content-Type")
	return data, contentType, nil
}

// VerifySignature verifies the X-Hub-Signature-256 header from Meta webhooks.
func VerifySignature(appSecret string, payload []byte, headerSig string) bool {
	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write(payload)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(headerSig))
}
