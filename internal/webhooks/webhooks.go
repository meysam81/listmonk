// Package webhooks implements an outgoing webhook delivery system for listmonk.
// It handles the delivery of events to configured webhook endpoints with
// retry logic, HMAC signatures, and delivery logging.
package webhooks

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/listmonk/models"
)

// Manager handles webhook event delivery.
type Manager struct {
	opts   Opt
	log    *log.Logger
	client *http.Client

	mu        sync.RWMutex
	isRunning bool
	stopCh    chan struct{}
	wg        sync.WaitGroup
}

// Opt contains options for initializing the webhook manager.
type Opt struct {
	DB       *sqlx.DB
	Queries  *Queries
	Log      *log.Logger
	Workers  int
	Interval time.Duration
}

// Queries contains prepared SQL queries for webhook operations.
type Queries struct {
	GetWebhooksByEvent    *sqlx.Stmt
	CreateWebhookLog      *sqlx.Stmt
	UpdateWebhookLog      *sqlx.Stmt
	GetPendingWebhookLogs *sqlx.Stmt
}

// New creates a new webhook manager.
func New(opt Opt) *Manager {
	if opt.Workers <= 0 {
		opt.Workers = 2
	}
	if opt.Interval <= 0 {
		opt.Interval = 5 * time.Second
	}

	return &Manager{
		opts: opt,
		log:  opt.Log,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		stopCh: make(chan struct{}),
	}
}

// Run starts the webhook delivery workers.
func (m *Manager) Run() {
	m.mu.Lock()
	if m.isRunning {
		m.mu.Unlock()
		return
	}
	m.isRunning = true
	m.mu.Unlock()

	m.log.Printf("starting webhook manager with %d workers", m.opts.Workers)

	// Start worker goroutines.
	for i := 0; i < m.opts.Workers; i++ {
		m.wg.Add(1)
		go m.worker(i)
	}
}

// Close stops the webhook manager.
func (m *Manager) Close() {
	m.mu.Lock()
	if !m.isRunning {
		m.mu.Unlock()
		return
	}
	m.isRunning = false
	m.mu.Unlock()

	close(m.stopCh)
	m.wg.Wait()
	m.log.Println("webhook manager stopped")
}

// Trigger queues an event for delivery to all matching webhooks.
func (m *Manager) Trigger(event string, data any) error {
	// Get all enabled webhooks that are subscribed to this event.
	var webhooks []models.Webhook
	if err := m.opts.Queries.GetWebhooksByEvent.Select(&webhooks, event); err != nil {
		m.log.Printf("error getting webhooks for event %s: %v", event, err)
		return err
	}

	if len(webhooks) == 0 {
		return nil
	}

	// Build the event payload.
	payload := models.WebhookEvent{
		Event:     event,
		Timestamp: time.Now().UTC(),
		Data:      data,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		m.log.Printf("error marshaling webhook payload: %v", err)
		return err
	}

	// Create a log entry for each webhook to be delivered.
	for _, wh := range webhooks {
		_, err := m.opts.Queries.CreateWebhookLog.Exec(
			wh.ID,
			event,
			wh.URL,
			payloadBytes,
			models.WebhookLogStatusPending,
			nil, // next_retry_at is null for immediate delivery
		)
		if err != nil {
			m.log.Printf("error creating webhook log for webhook %d: %v", wh.ID, err)
		}
	}

	return nil
}

// worker processes pending webhook deliveries.
func (m *Manager) worker(id int) {
	defer m.wg.Done()

	ticker := time.NewTicker(m.opts.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.processPendingLogs()
		}
	}
}

// pendingLog represents a pending webhook log with associated webhook info.
type pendingLog struct {
	models.WebhookLog
	MaxRetries     int    `db:"max_retries"`
	Timeout        string `db:"timeout"`
	AuthType       string `db:"auth_type"`
	AuthBasicUser  string `db:"auth_basic_user"`
	AuthBasicPass  string `db:"auth_basic_pass"`
	AuthHMACSecret string `db:"auth_hmac_secret"`
}

// processPendingLogs fetches and processes pending webhook deliveries.
func (m *Manager) processPendingLogs() {
	var logs []pendingLog
	if err := m.opts.Queries.GetPendingWebhookLogs.Select(&logs, 100); err != nil {
		m.log.Printf("error fetching pending webhook logs: %v", err)
		return
	}

	for _, l := range logs {
		m.deliverWebhook(l)
	}
}

// deliverWebhook attempts to deliver a webhook and updates the log.
func (m *Manager) deliverWebhook(l pendingLog) {
	// Parse timeout.
	timeout, err := time.ParseDuration(l.Timeout)
	if err != nil {
		timeout = 30 * time.Second
	}

	// Create HTTP request.
	req, err := http.NewRequest(http.MethodPost, l.URL, bytes.NewReader(l.Payload))
	if err != nil {
		m.updateLogFailed(l, 0, "", fmt.Sprintf("error creating request: %v", err))
		return
	}

	// Set headers.
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "listmonk-webhook/1.0")
	req.Header.Set("X-Listmonk-Event", l.Event)
	req.Header.Set("X-Listmonk-Delivery", fmt.Sprintf("%d", l.ID))

	// Apply authentication.
	switch l.AuthType {
	case models.WebhookAuthTypeBasic:
		req.SetBasicAuth(l.AuthBasicUser, l.AuthBasicPass)

	case models.WebhookAuthTypeHMAC:
		timestamp := time.Now().Unix()
		signature := m.computeHMAC(l.Payload, l.AuthHMACSecret, timestamp)
		req.Header.Set("X-Listmonk-Signature", signature)
		req.Header.Set("X-Listmonk-Timestamp", fmt.Sprintf("%d", timestamp))
	}

	// Create a client with the specific timeout.
	client := &http.Client{Timeout: timeout}

	// Make the request.
	resp, err := client.Do(req)
	if err != nil {
		m.handleDeliveryError(l, 0, "", fmt.Sprintf("request failed: %v", err))
		return
	}
	defer resp.Body.Close()

	// Read response body (limit to 1KB to prevent memory issues).
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	bodyStr := string(body)

	// Check if delivery was successful (2xx status).
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		m.updateLogSuccess(l, resp.StatusCode, bodyStr)
	} else {
		m.handleDeliveryError(l, resp.StatusCode, bodyStr, fmt.Sprintf("non-2xx status: %d", resp.StatusCode))
	}
}

// computeHMAC computes the HMAC-SHA256 signature for the payload.
func (m *Manager) computeHMAC(payload []byte, secret string, timestamp int64) string {
	// Signature is computed as HMAC-SHA256(timestamp.payload, secret)
	data := fmt.Sprintf("%d.%s", timestamp, string(payload))
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}

// updateLogSuccess marks a webhook log as successfully delivered.
func (m *Manager) updateLogSuccess(l pendingLog, statusCode int, responseBody string) {
	_, err := m.opts.Queries.UpdateWebhookLog.Exec(
		l.ID,
		models.WebhookLogStatusSuccess,
		statusCode,
		responseBody,
		"",
		l.Attempts+1,
		nil, // next_retry_at
	)
	if err != nil {
		m.log.Printf("error updating webhook log %d: %v", l.ID, err)
	}
}

// updateLogFailed marks a webhook log as permanently failed.
func (m *Manager) updateLogFailed(l pendingLog, statusCode int, responseBody, errMsg string) {
	_, err := m.opts.Queries.UpdateWebhookLog.Exec(
		l.ID,
		models.WebhookLogStatusFailed,
		statusCode,
		responseBody,
		errMsg,
		l.Attempts+1,
		nil, // next_retry_at
	)
	if err != nil {
		m.log.Printf("error updating webhook log %d: %v", l.ID, err)
	}
}

// handleDeliveryError handles a failed delivery attempt, scheduling a retry if allowed.
func (m *Manager) handleDeliveryError(l pendingLog, statusCode int, responseBody, errMsg string) {
	attempts := l.Attempts + 1

	// Check if we've exhausted retries. MaxRetries represents the number of retry
	// attempts allowed after the initial delivery attempt.
	if attempts > l.MaxRetries {
		m.updateLogFailed(l, statusCode, responseBody, errMsg)
		return
	}

	// Calculate next retry time with exponential backoff.
	// 30s, 2m, 8m, 32m, 2h (approximately)
	backoff := time.Duration(1<<uint(attempts)) * 30 * time.Second
	if backoff > 2*time.Hour {
		backoff = 2 * time.Hour
	}
	nextRetry := time.Now().Add(backoff)

	_, err := m.opts.Queries.UpdateWebhookLog.Exec(
		l.ID,
		models.WebhookLogStatusPending, // Keep pending for retry
		statusCode,
		responseBody,
		errMsg,
		attempts,
		nextRetry,
	)
	if err != nil {
		m.log.Printf("error scheduling retry for webhook log %d: %v", l.ID, err)
	}
}
