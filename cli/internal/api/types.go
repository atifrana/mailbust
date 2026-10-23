package api

// MailboxInfo mirrors the backend's mailbox create/info response.
type MailboxInfo struct {
	Address   string `json:"address"`
	Public    bool   `json:"public"`
	Token     string `json:"token,omitempty"`
	CreatedAt int64  `json:"createdAt"`
	ExpiresAt int64  `json:"expiresAt"`
}

// Attachment is one attachment's metadata on a message.
type Attachment struct {
	N           int    `json:"n"`
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	Size        int64  `json:"size"`
}

// Summary is a compact inbox-list row.
type Summary struct {
	ID             string  `json:"id"`
	From           string  `json:"from"`
	FromName       *string `json:"fromName"`
	Subject        string  `json:"subject"`
	Code           *string `json:"code"`
	Link           *string `json:"link"`
	ReceivedAt     int64   `json:"receivedAt"`
	HasAttachments bool    `json:"hasAttachments"`
}

// Message is a full stored message.
type Message struct {
	ID          string       `json:"id"`
	From        string       `json:"from"`
	FromName    *string      `json:"fromName"`
	Subject     string       `json:"subject"`
	Text        string       `json:"text"`
	HTML        *string      `json:"html"`
	Code        *string      `json:"code"`
	Link        *string      `json:"link"`
	ReceivedAt  int64        `json:"receivedAt"`
	Size        int64        `json:"size"`
	Attachments []Attachment `json:"attachments"`
}

// Sender returns the display name if present, else the bare address.
func (s Summary) Sender() string {
	if s.FromName != nil && *s.FromName != "" {
		return *s.FromName
	}
	return s.From
}
