package main

import (
	"net/http"

	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

// mjmlReq represents a request to compile MJML.
type mjmlReq struct {
	Body string `json:"body"`
}

// mjmlResp represents the response from MJML compilation.
type mjmlResp struct {
	HTML string `json:"html"`
}

// CompileMJML handles MJML to HTML compilation requests.
// This endpoint allows users to preview their MJML templates
// before saving them as campaigns.
func (a *App) CompileMJML(c echo.Context) error {
	var req mjmlReq
	if err := c.Bind(&req); err != nil {
		return err
	}

	if req.Body == "" {
		return echo.NewHTTPError(http.StatusBadRequest,
			a.i18n.T("campaigns.fieldInvalidBody"))
	}

	// Compile MJML to HTML.
	html, err := models.CompileMJML(req.Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, okResp{mjmlResp{HTML: html}})
}
