package core

import (
	"net/http"
	"strings"

	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
)

var webhookLogQuerySortFields = []string{"created_at", "status", "event"}

// GetWebhooks retrieves all webhooks or a specific one by ID.
func (c *Core) GetWebhooks(id int) ([]models.Webhook, error) {
	var out []models.Webhook
	if err := c.q.GetWebhooks.Select(&out, id); err != nil {
		c.log.Printf("error fetching webhooks: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.webhook}", "error", pqErrMsg(err)))
	}

	// Mask secrets for security.
	for i := range out {
		if out[i].AuthBasicPass != "" {
			out[i].AuthBasicPass = strings.Repeat("•", 8)
		}
		if out[i].AuthHMACSecret != "" {
			out[i].AuthHMACSecret = strings.Repeat("•", 8)
		}
	}

	return out, nil
}

// GetWebhook retrieves a single webhook by ID.
func (c *Core) GetWebhook(id int) (models.Webhook, error) {
	out, err := c.GetWebhooks(id)
	if err != nil {
		return models.Webhook{}, err
	}

	if len(out) == 0 {
		return models.Webhook{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.webhook}"))
	}

	return out[0], nil
}

// CreateWebhook creates a new webhook.
func (c *Core) CreateWebhook(w models.Webhook) (models.Webhook, error) {
	// Validate.
	if !strHasLen(w.Name, 1, 200) {
		return models.Webhook{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.invalidFields", "name", "name"))
	}
	if !strHasLen(w.URL, 1, 2000) {
		return models.Webhook{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.invalidFields", "name", "url"))
	}
	if len(w.Events) == 0 {
		return models.Webhook{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.invalidFields", "name", "events"))
	}

	// Validate events.
	validEvents := make(map[string]bool)
	for _, e := range models.AllWebhookEvents() {
		validEvents[e] = true
	}
	for _, e := range w.Events {
		if !validEvents[e] {
			return models.Webhook{}, echo.NewHTTPError(http.StatusBadRequest,
				c.i18n.Ts("globals.messages.invalidFields", "name", "events"))
		}
	}

	// Validate auth type.
	if w.AuthType != models.WebhookAuthTypeNone &&
		w.AuthType != models.WebhookAuthTypeBasic &&
		w.AuthType != models.WebhookAuthTypeHMAC {
		w.AuthType = models.WebhookAuthTypeNone
	}

	// Set defaults.
	if w.Status == "" {
		w.Status = models.WebhookStatusEnabled
	}
	if w.MaxRetries <= 0 {
		w.MaxRetries = 3
	}
	if w.RetryInterval == "" {
		w.RetryInterval = "30s"
	}
	if w.Timeout == "" {
		w.Timeout = "30s"
	}

	var (
		id   int
		uuid string
	)
	if err := c.q.CreateWebhook.QueryRow(
		w.Name,
		w.URL,
		w.Status,
		pq.Array(w.Events),
		w.AuthType,
		w.AuthBasicUser,
		w.AuthBasicPass,
		w.AuthHMACSecret,
		w.MaxRetries,
		w.RetryInterval,
		w.Timeout,
	).Scan(&id, &uuid); err != nil {
		c.log.Printf("error creating webhook: %v", err)
		return models.Webhook{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.webhook}", "error", pqErrMsg(err)))
	}

	return c.GetWebhook(id)
}

// UpdateWebhook updates an existing webhook.
func (c *Core) UpdateWebhook(id int, w models.Webhook) (models.Webhook, error) {
	// Validate.
	if !strHasLen(w.Name, 1, 200) {
		return models.Webhook{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.invalidFields", "name", "name"))
	}
	if !strHasLen(w.URL, 1, 2000) {
		return models.Webhook{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.invalidFields", "name", "url"))
	}
	if len(w.Events) == 0 {
		return models.Webhook{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.invalidFields", "name", "events"))
	}

	// Validate events.
	validEvents := make(map[string]bool)
	for _, e := range models.AllWebhookEvents() {
		validEvents[e] = true
	}
	for _, e := range w.Events {
		if !validEvents[e] {
			return models.Webhook{}, echo.NewHTTPError(http.StatusBadRequest,
				c.i18n.Ts("globals.messages.invalidFields", "name", "events"))
		}
	}

	// Validate auth type.
	if w.AuthType != models.WebhookAuthTypeNone &&
		w.AuthType != models.WebhookAuthTypeBasic &&
		w.AuthType != models.WebhookAuthTypeHMAC {
		w.AuthType = models.WebhookAuthTypeNone
	}

	res, err := c.q.UpdateWebhook.Exec(
		id,
		w.Name,
		w.URL,
		w.Status,
		pq.Array(w.Events),
		w.AuthType,
		w.AuthBasicUser,
		w.AuthBasicPass,
		w.AuthHMACSecret,
		w.MaxRetries,
		w.RetryInterval,
		w.Timeout,
	)
	if err != nil {
		c.log.Printf("error updating webhook: %v", err)
		return models.Webhook{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.webhook}", "error", pqErrMsg(err)))
	}

	if n, _ := res.RowsAffected(); n == 0 {
		return models.Webhook{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.webhook}"))
	}

	return c.GetWebhook(id)
}

// DeleteWebhooks deletes one or more webhooks.
func (c *Core) DeleteWebhooks(ids []int) error {
	if _, err := c.q.DeleteWebhooks.Exec(pq.Array(ids)); err != nil {
		c.log.Printf("error deleting webhooks: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.webhook}", "error", pqErrMsg(err)))
	}
	return nil
}

// QueryWebhookLogs retrieves webhook logs based on filters.
func (c *Core) QueryWebhookLogs(webhookID int, status, event, orderBy, order string, offset, limit int) ([]models.WebhookLog, int, error) {
	if !strSliceContains(orderBy, webhookLogQuerySortFields) {
		orderBy = "created_at"
	}
	if order != SortAsc && order != SortDesc {
		order = SortDesc
	}

	out := []models.WebhookLog{}
	stmt := strings.ReplaceAll(c.q.QueryWebhookLogs, "%order%", orderBy+" "+order)
	if err := c.db.Select(&out, stmt, webhookID, status, event, offset, limit); err != nil {
		c.log.Printf("error fetching webhook logs: %v", err)
		return nil, 0, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.webhook}", "error", pqErrMsg(err)))
	}

	total := 0
	if len(out) > 0 {
		total = out[0].Total
	}

	return out, total, nil
}

// DeleteWebhookLogs deletes webhook logs based on filters.
func (c *Core) DeleteWebhookLogs(all bool, ids []int, webhookID int, status string) error {
	if _, err := c.q.DeleteWebhookLogs.Exec(all, pq.Array(ids), webhookID, status); err != nil {
		c.log.Printf("error deleting webhook logs: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.webhook}", "error", pqErrMsg(err)))
	}
	return nil
}
