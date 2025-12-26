package models

import (
	"encoding/json"
	"time"

	"github.com/lib/pq"
)

// Webhook event types.
const (
	// Subscriber events.
	EventSubscriberCreated     = "subscriber.created"
	EventSubscriberUpdated     = "subscriber.updated"
	EventSubscriberDeleted     = "subscriber.deleted"
	EventSubscriberOptinStart  = "subscriber.optin_start"
	EventSubscriberOptinFinish = "subscriber.optin_finish"

	// Subscription events.
	EventSubscriberAddedToList     = "subscriber.added_to_list"
	EventSubscriberRemovedFromList = "subscriber.removed_from_list"
	EventSubscriberUnsubscribed    = "subscriber.unsubscribed"

	// Bounce events.
	EventSubscriberBounced = "subscriber.bounced"

	// Campaign events.
	EventCampaignStarted   = "campaign.started"
	EventCampaignPaused    = "campaign.paused"
	EventCampaignCancelled = "campaign.cancelled"
	EventCampaignFinished  = "campaign.finished"
)

// Webhook auth types.
const (
	WebhookAuthTypeNone  = "none"
	WebhookAuthTypeBasic = "basic"
	WebhookAuthTypeHMAC  = "hmac"
)

// Webhook status values.
const (
	WebhookStatusEnabled  = "enabled"
	WebhookStatusDisabled = "disabled"
)

// Webhook log status values.
const (
	WebhookLogStatusPending = "pending"
	WebhookLogStatusSuccess = "success"
	WebhookLogStatusFailed  = "failed"
)

// Webhook represents a webhook endpoint configuration.
type Webhook struct {
	ID             int            `db:"id" json:"id"`
	UUID           string         `db:"uuid" json:"uuid"`
	Name           string         `db:"name" json:"name"`
	URL            string         `db:"url" json:"url"`
	Status         string         `db:"status" json:"status"`
	Events         pq.StringArray `db:"events" json:"events"`
	AuthType       string         `db:"auth_type" json:"auth_type"`
	AuthBasicUser  string         `db:"auth_basic_user" json:"auth_basic_user"`
	AuthBasicPass  string         `db:"auth_basic_pass" json:"auth_basic_pass,omitempty"`
	AuthHMACSecret string         `db:"auth_hmac_secret" json:"auth_hmac_secret,omitempty"`
	MaxRetries     int            `db:"max_retries" json:"max_retries"`
	RetryInterval  string         `db:"retry_interval" json:"retry_interval"`
	Timeout        string         `db:"timeout" json:"timeout"`
	CreatedAt      time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time      `db:"updated_at" json:"updated_at"`

	// Pseudofield for getting the total count in paginated queries.
	Total int `db:"total" json:"-"`
}

// WebhookLog represents a webhook delivery log entry.
type WebhookLog struct {
	ID           int64           `db:"id" json:"id"`
	WebhookID    int             `db:"webhook_id" json:"webhook_id"`
	Event        string          `db:"event" json:"event"`
	URL          string          `db:"url" json:"url"`
	Payload      json.RawMessage `db:"payload" json:"payload"`
	Status       string          `db:"status" json:"status"`
	ResponseCode *int            `db:"response_code" json:"response_code"`
	ResponseBody string          `db:"response_body" json:"response_body"`
	Error        string          `db:"error" json:"error"`
	Attempts     int             `db:"attempts" json:"attempts"`
	NextRetryAt  *time.Time      `db:"next_retry_at" json:"next_retry_at"`
	CreatedAt    time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time       `db:"updated_at" json:"updated_at"`

	// Pseudofield for getting the total count in paginated queries.
	Total int `db:"total" json:"-"`

	// Joined webhook name for display.
	WebhookName string `db:"webhook_name" json:"webhook_name,omitempty"`
}

// WebhookEvent represents an event payload to be sent to webhooks.
type WebhookEvent struct {
	Event     string    `json:"event"`
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data"`
}

// AllWebhookEvents returns a list of all available webhook events.
func AllWebhookEvents() []string {
	return []string{
		EventSubscriberCreated,
		EventSubscriberUpdated,
		EventSubscriberDeleted,
		EventSubscriberOptinStart,
		EventSubscriberOptinFinish,
		EventSubscriberAddedToList,
		EventSubscriberRemovedFromList,
		EventSubscriberUnsubscribed,
		EventSubscriberBounced,
		EventCampaignStarted,
		EventCampaignPaused,
		EventCampaignCancelled,
		EventCampaignFinished,
	}
}
