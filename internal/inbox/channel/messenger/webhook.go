package messenger

// WebhookPayload is the top-level payload from Meta webhooks.
type WebhookPayload struct {
	Object string         `json:"object"` // "page" or "instagram"
	Entry  []WebhookEntry `json:"entry"`
}

// WebhookEntry represents a single entry in the webhook payload.
type WebhookEntry struct {
	ID        string            `json:"id"`   // Page ID or IG account ID
	Time      int64             `json:"time"`
	Messaging []MessagingEvent  `json:"messaging"`
}

// MessagingEvent represents a single messaging event.
type MessagingEvent struct {
	Sender    IDField          `json:"sender"`
	Recipient IDField          `json:"recipient"`
	Timestamp int64            `json:"timestamp"`
	Message   *IncomingMsg     `json:"message,omitempty"`
	Delivery  *DeliveryReceipt `json:"delivery,omitempty"`
	Read      *ReadReceipt     `json:"read,omitempty"`
	Postback  *Postback        `json:"postback,omitempty"`
}

// IDField holds a single ID string.
type IDField struct {
	ID string `json:"id"`
}

// IncomingMsg represents an incoming message from a user.
type IncomingMsg struct {
	MID         string       `json:"mid"`
	Text        string       `json:"text"`
	Attachments []Attachment `json:"attachments,omitempty"`
	ReplyTo     *ReplyTo     `json:"reply_to,omitempty"`
	IsEcho      bool         `json:"is_echo"`
}

// Attachment represents a message attachment.
type Attachment struct {
	Type    string           `json:"type"` // image, video, audio, file, fallback, location, sticker
	Payload AttachmentDetail `json:"payload"`
}

// AttachmentDetail holds attachment-specific data.
type AttachmentDetail struct {
	URL       string  `json:"url,omitempty"`
	StickerID int64   `json:"sticker_id,omitempty"`
	Title     string  `json:"title,omitempty"`
	Lat       float64 `json:"coordinates.lat,omitempty"`
	Long      float64 `json:"coordinates.long,omitempty"`
}

// ReplyTo holds reference info when a message is a reply.
type ReplyTo struct {
	MID string `json:"mid"`
}

// DeliveryReceipt indicates messages were delivered.
type DeliveryReceipt struct {
	MIDs      []string `json:"mids"`
	Watermark int64    `json:"watermark"`
}

// ReadReceipt indicates messages were read.
type ReadReceipt struct {
	Watermark int64 `json:"watermark"`
}

// Postback represents a postback from a button or menu.
type Postback struct {
	Title   string `json:"title"`
	Payload string `json:"payload"`
}
