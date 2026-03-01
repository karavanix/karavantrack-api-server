package firebase

import (
	"context"
	"fmt"
	"time"

	firebase "firebase.google.com/go/v4"
	messaging "firebase.google.com/go/v4/messaging"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/security"
	"github.com/karavanix/karavantrack-api-server/pkg/utils"
	"github.com/shogo82148/pointer"

	"google.golang.org/api/option"
)

var (
	ErrInvalidToken   = fmt.Errorf("invalid token")
	ErrEmptyTokens    = fmt.Errorf("no tokens provided")
	ErrBatchSizeLimit = fmt.Errorf("batch size exceeds limit")
)

const (
	MaxBatchSize = 500 // FCM limit for multicast messages
)

type FCM struct {
	app       *firebase.App
	msgClient *messaging.Client
}

func New(ctx context.Context, cfg *config.Config) (*FCM, error) {
	serviceKey, err := security.DecodeBase64(cfg.Firebase.AuthKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode service key: %v", err)
	}

	opt := option.WithAuthCredentialsJSON(option.ServiceAccount, []byte(serviceKey))
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize app: %v", err)
	}

	msgClient, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize messaging client: %v", err)
	}

	return &FCM{
		app:       app,
		msgClient: msgClient,
	}, nil
}

// Send sends a notification to a single device token
func (f *FCM) Send(ctx context.Context, token string, n *Notification) (*MulticastResult, error) {
	if token == "" {
		return nil, ErrInvalidToken
	}

	ttl := time.Hour * 24

	message := &messaging.Message{
		Notification: &messaging.Notification{
			Title:    n.Title,
			Body:     n.Body,
			ImageURL: n.ImageURL,
		},
		Data:  n.Metadata,
		Token: token,
		FCMOptions: &messaging.FCMOptions{
			AnalyticsLabel: "general_push",
		},
		// ANDROID config
		Android: &messaging.AndroidConfig{
			Priority:     utils.If(n.Options.Priority != "", n.Options.Priority, "high"),
			TTL:          &ttl,
			DirectBootOK: true,
			FCMOptions: &messaging.AndroidFCMOptions{
				AnalyticsLabel: "android_push",
			},
			Notification: &messaging.AndroidNotification{
				ChannelID:         "general",
				Sound:             "default",
				NotificationCount: pointer.IntOrNil(n.Options.NotificationCount),
				Title:             n.Title,
				Body:              n.Body,
				ImageURL:          n.ImageURL,
			},
		},

		// iOS / APNS config
		APNS: &messaging.APNSConfig{
			Headers: map[string]string{
				"apns-priority": utils.If(n.Options.Priority != "", utils.If(n.Options.Priority == "high", "10", "5"), "10"),
			},
			FCMOptions: &messaging.APNSFCMOptions{
				AnalyticsLabel: "ios_push",
				ImageURL:       n.ImageURL,
			},

			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					// Visible notification
					Alert: &messaging.ApsAlert{
						Title: n.Title,
						Body:  n.Body,
					},

					// Badge count (only set if your backend controls it)
					Badge: pointer.IntOrNil(n.Options.NotificationCount),

					// Sound
					Sound: "default",

					// If you use Notification Service Extension (rich media)
					MutableContent: true,

					// Optional grouping on iOS notification center
					ThreadID: "general",
				},
			},
		},
	}

	messageID, err := f.msgClient.Send(ctx, message)
	if err != nil {
		result := &MulticastResult{
			SuccessCount:  0,
			FailureCount:  1,
			InvalidTokens: []string{},
		}

		if messaging.IsUnregistered(err) {
			result.InvalidTokens = append(result.InvalidTokens, token)
			return result, ErrInvalidToken
		}

		return result, fmt.Errorf("failed to send message: %v", err)
	}

	return &MulticastResult{
		SuccessCount:  1,
		FailureCount:  0,
		MessageID:     messageID,
		InvalidTokens: []string{},
	}, nil
}

// SendMulticast sends a notification to multiple device tokens
func (f *FCM) SendMulticast(ctx context.Context, tokens []string, n *Notification) (*MulticastResult, error) {
	if len(tokens) == 0 {
		return nil, ErrEmptyTokens
	}

	if len(tokens) > MaxBatchSize {
		return nil, ErrBatchSizeLimit
	}

	ttl := time.Hour * 24

	multicastMessage := &messaging.MulticastMessage{
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title:    n.Title,
			Body:     n.Body,
			ImageURL: n.ImageURL,
		},
		Data: n.Metadata,
		FCMOptions: &messaging.FCMOptions{
			AnalyticsLabel: "general_push",
		},

		// ANDROID config
		Android: &messaging.AndroidConfig{
			Priority:     utils.If(n.Options.Priority != "", n.Options.Priority, "high"),
			TTL:          &ttl,
			DirectBootOK: true,
			FCMOptions: &messaging.AndroidFCMOptions{
				AnalyticsLabel: "android_push",
			},
			Notification: &messaging.AndroidNotification{
				ChannelID:         "general",
				Sound:             "default",
				NotificationCount: pointer.IntOrNil(n.Options.NotificationCount),
				Title:             n.Title,
				Body:              n.Body,
				ImageURL:          n.ImageURL,
			},
		},

		// iOS / APNS config
		APNS: &messaging.APNSConfig{
			Headers: map[string]string{
				"apns-priority": utils.If(n.Options.Priority != "", utils.If(n.Options.Priority == "high", "10", "5"), "10"),
			},
			FCMOptions: &messaging.APNSFCMOptions{
				AnalyticsLabel: "ios_push",
				ImageURL:       n.ImageURL,
			},

			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					// Visible notification
					Alert: &messaging.ApsAlert{
						Title: n.Title,
						Body:  n.Body,
					},

					// Badge count (only set if your backend controls it)
					Badge: pointer.IntOrNil(n.Options.NotificationCount),

					// Sound
					Sound: "default",

					// If you use Notification Service Extension (rich media)
					MutableContent: true,

					// Optional grouping on iOS notification center
					ThreadID: "general",
				},
			},
		},
	}

	// Use the newer SendEachForMulticast method which replaces the deprecated SendMulticast
	response, err := f.msgClient.SendEachForMulticast(ctx, multicastMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to send multicast message: %v", err)
	}

	result := &MulticastResult{
		SuccessCount:  response.SuccessCount,
		FailureCount:  response.FailureCount,
		InvalidTokens: make([]string, 0),
	}

	// Collect invalid tokens by checking each response
	for i, resp := range response.Responses {
		if !resp.Success {
			if messaging.IsUnregistered(resp.Error) {
				result.InvalidTokens = append(result.InvalidTokens, tokens[i])
			} else {
				logger.WarnContext(ctx, "failed to send message", "token", tokens[i], "error", resp.Error)
			}
		}
	}

	return result, nil
}

// SendBatch handles large token lists by splitting them into smaller batches
func (f *FCM) SendBatch(ctx context.Context, tokens []string, n *Notification) (*MulticastResult, error) {
	if len(tokens) == 0 {
		return nil, ErrEmptyTokens
	}

	totalResult := &MulticastResult{
		InvalidTokens: make([]string, 0),
	}

	// Process tokens in batches of MaxBatchSize
	for i := 0; i < len(tokens); i += MaxBatchSize {
		end := min(i+MaxBatchSize, len(tokens))

		batch := tokens[i:end]
		result, err := f.SendMulticast(ctx, batch, n)
		if err != nil {
			return nil, fmt.Errorf("failed to send batch %d: %v", i/MaxBatchSize+1, err)
		}

		totalResult.SuccessCount += result.SuccessCount
		totalResult.FailureCount += result.FailureCount
		totalResult.InvalidTokens = append(totalResult.InvalidTokens, result.InvalidTokens...)
	}

	return totalResult, nil
}

// CreateNotification is a helper to create a notification
func CreateNotification(title, body string) *Notification {
	return &Notification{
		Title: title,
		Body:  body,
	}
}

// CreateNotificationWithMetadata creates a notification with custom data
func CreateNotificationWithMetadata(title, body string, metadata map[string]string) *Notification {
	return &Notification{
		Title:    title,
		Body:     body,
		Metadata: metadata,
	}
}

// CreateNotificationWithImage creates a notification with an image
func CreateNotificationWithImage(title, body, imageURL string) *Notification {
	return &Notification{
		Title:    title,
		Body:     body,
		ImageURL: imageURL,
	}
}
