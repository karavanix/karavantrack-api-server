package firebase

type Notification struct {
	Title    string              `json:"title"`
	Body     string              `json:"body"`
	ImageURL string              `json:"image_url,omitempty"`
	Metadata map[string]string   `json:"metadata,omitempty"`
	Options  NotificationOptions `json:"options,omitempty"`
}

type NotificationOptions struct {
	Priority          string `json:"priority,omitempty"`
	NotificationCount int    `json:"notification_count,omitempty"`
}

type MulticastResult struct {
	SuccessCount  int
	FailureCount  int
	InvalidTokens []string
	MessageID     string // For single sends
}
