package main

import (
	"net/http"
	"strconv"

	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

// GetWebhooks handles retrieval of webhooks.
func (a *App) GetWebhooks(c echo.Context) error {
	out, err := a.core.GetWebhooks(0)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// GetWebhook handles retrieval of a single webhook.
func (a *App) GetWebhook(c echo.Context) error {
	id := getID(c)
	out, err := a.core.GetWebhook(id)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// CreateWebhook handles creation of a new webhook.
func (a *App) CreateWebhook(c echo.Context) error {
	var w models.Webhook
	if err := c.Bind(&w); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("globals.messages.invalidData", "error", err.Error()))
	}

	out, err := a.core.CreateWebhook(w)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// UpdateWebhook handles updating a webhook.
func (a *App) UpdateWebhook(c echo.Context) error {
	id := getID(c)

	var w models.Webhook
	if err := c.Bind(&w); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("globals.messages.invalidData", "error", err.Error()))
	}

	out, err := a.core.UpdateWebhook(id, w)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// DeleteWebhooks handles deletion of webhooks.
func (a *App) DeleteWebhooks(c echo.Context) error {
	ids, err := parseStringIDs(c.Request().URL.Query()["id"])
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("globals.messages.invalidID", "error", err.Error()))
	}
	if len(ids) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.Ts("globals.messages.invalidID"))
	}

	if err := a.core.DeleteWebhooks(ids); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// DeleteWebhook handles deletion of a single webhook.
func (a *App) DeleteWebhook(c echo.Context) error {
	id := getID(c)
	if err := a.core.DeleteWebhooks([]int{id}); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// GetWebhookLogs handles retrieval of webhook delivery logs.
func (a *App) GetWebhookLogs(c echo.Context) error {
	var (
		webhookID, _ = strconv.Atoi(c.QueryParam("webhook_id"))
		status       = c.QueryParam("status")
		event        = c.QueryParam("event")
		orderBy      = c.QueryParam("order_by")
		order        = c.QueryParam("order")

		pg = a.pg.NewFromURL(c.Request().URL.Query())
	)

	res, total, err := a.core.QueryWebhookLogs(webhookID, status, event, orderBy, order, pg.Offset, pg.Limit)
	if err != nil {
		return err
	}

	if len(res) == 0 {
		return c.JSON(http.StatusOK, okResp{models.PageResults{Results: []models.WebhookLog{}}})
	}

	out := models.PageResults{
		Results: res,
		Total:   total,
		Page:    pg.Page,
		PerPage: pg.PerPage,
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// DeleteWebhookLogs handles deletion of webhook logs.
func (a *App) DeleteWebhookLogs(c echo.Context) error {
	var (
		all, _       = strconv.ParseBool(c.QueryParam("all"))
		webhookID, _ = strconv.Atoi(c.QueryParam("webhook_id"))
		status       = c.QueryParam("status")
	)

	var ids []int
	if !all && webhookID == 0 && status == "" {
		res, err := parseStringIDs(c.Request().URL.Query()["id"])
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest,
				a.i18n.Ts("globals.messages.invalidID", "error", err.Error()))
		}
		ids = res
	}

	if err := a.core.DeleteWebhookLogs(all, ids, webhookID, status); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// GetWebhookEvents returns the list of available webhook events.
func (a *App) GetWebhookEvents(c echo.Context) error {
	return c.JSON(http.StatusOK, okResp{models.AllWebhookEvents()})
}

// TestWebhook triggers a test webhook delivery.
func (a *App) TestWebhook(c echo.Context) error {
	id := getID(c)

	// Get the webhook to verify it exists.
	wh, err := a.core.GetWebhook(id)
	if err != nil {
		return err
	}

	// Send a test event.
	testData := map[string]any{
		"message": "This is a test webhook delivery from listmonk",
		"webhook": map[string]any{
			"id":   wh.ID,
			"name": wh.Name,
		},
	}

	if err := a.webhooks.Trigger("webhook.test", testData); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			a.i18n.Ts("globals.messages.errorCreating", "name", "test webhook", "error", err.Error()))
	}

	return c.JSON(http.StatusOK, okResp{true})
}
