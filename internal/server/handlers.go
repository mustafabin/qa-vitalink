package server

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/skip2/go-qrcode"
	"gorm.io/gorm"

	"vitalink/internal/models"
)

func handleCreatePaymentPage(c echo.Context, db *gorm.DB) error {
	var req struct {
		MerchantID  string     `json:"merchant_id" validate:"required"`
		PageUID     string     `json:"page_uid" validate:"required"`
		AmountCents int64      `json:"amount_cents" validate:"required"`
		Currency    string     `json:"currency"`
		Title       string     `json:"title"`
		Description string     `json:"description"`
		StoreName   string     `json:"store_name"`
		ExpireAt    *time.Time `json:"expire_at"`

		InvoiceNo             string `json:"invoice_no"`
		IncludeTip            bool   `json:"include_tip"`
		AllowedTipPercentages string `json:"allowed_tip_percentages"`
		PaymentFeeAmount      string `json:"payment_fee_amount"`
		PaymentFeeDescription string `json:"payment_fee_description"`
		SurchargeAmount       string `json:"surcharge_amount"`
		TaxAmount             string `json:"tax_amount"`
		Items                 string `json:"items"`
		PaymentTypesAllowed   string `json:"payment_types_allowed"`
		ApplePayMid           string `json:"apple_pay_mid"`
		GooglePayMid          string `json:"google_pay_mid"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
	}
	if req.MerchantID == "" || req.PageUID == "" || req.AmountCents == 0 {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "merchant_id, page_uid, amount_cents are required"})
	}

	if req.Currency == "" { req.Currency = "USD" }

	pp := models.PaymentPage{
		MerchantID:  req.MerchantID,
		PageUID:     req.PageUID,
		AmountCents: req.AmountCents,
		Currency:    req.Currency,
		Title:       req.Title,
		Description: req.Description,
		StoreName:   req.StoreName,
		Status:      "open",
		ExpireAt:    req.ExpireAt,

		InvoiceNo:             req.InvoiceNo,
		IncludeTip:            req.IncludeTip,
		AllowedTipPercentages: req.AllowedTipPercentages,
		PaymentFeeAmount:      req.PaymentFeeAmount,
		PaymentFeeDescription: req.PaymentFeeDescription,
		SurchargeAmount:       req.SurchargeAmount,
		TaxAmount:             req.TaxAmount,
		Items:                 req.Items,
		PaymentTypesAllowed:   req.PaymentTypesAllowed,
		ApplePayMid:           req.ApplePayMid,
		GooglePayMid:          req.GooglePayMid,
	}

	if err := db.Create(&pp).Error; err != nil {
		if isUnique(err) {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "payment page exists",
				"details": err.Error(),
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "create failed",
			"details": err.Error(),
		})
	}

	scheme := "https"
	if c.Scheme() != "" { scheme = c.Scheme() }
	host := c.Request().Host
	base := scheme + "://" + host
	paymentURL := base + "/p/" + pp.MerchantID + "/" + pp.PageUID
	qrURL := base + "/qr/" + pp.MerchantID + "/" + pp.PageUID

	return c.JSON(http.StatusCreated, map[string]any{
		"payment_url": paymentURL,
		"qr_url":      qrURL,
	})
}

func handleViewPaymentPage(c echo.Context, db *gorm.DB) error {
	merchantID := c.Param("merchant_id")
	pageUID := c.Param("page_uid")

	var pp models.PaymentPage
	err := db.First(&pp, "merchant_id = ? AND page_uid = ?", merchantID, pageUID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return c.Render(http.StatusNotFound, "not_found.html", map[string]any{})
	} else if err != nil {
		return c.String(http.StatusInternalServerError, "error")
	}

	if pp.Status != "open" || pp.IsExpired(time.Now()) {
		return c.Render(http.StatusOK, "expired.html", map[string]any{"page": pp})
	}
	return c.Render(http.StatusOK, "payment.html", map[string]any{"page": pp})
}

func handleQRPaymentPage(c echo.Context) error {
	merchantID := c.Param("merchant_id")
	pageUID := c.Param("page_uid")

	scheme := "https"
	if c.Scheme() != "" { scheme = c.Scheme() }
	host := c.Request().Host
	url := scheme + "://" + host + "/p/" + merchantID + "/" + pageUID

	sz := 256
	if q := c.QueryParam("size"); q != "" {
		if v, err := strconv.Atoi(q); err == nil && v >= 64 && v <= 2048 {
			sz = v
		}
	}

	png, err := qrcode.Encode(url, qrcode.Medium, sz)
	if err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.Blob(http.StatusOK, "image/png", png)
}

func isUnique(err error) bool {
	if err == nil { return false }
	s := err.Error()
	return strings.Contains(s, "duplicate key value") || strings.Contains(strings.ToLower(s), "unique")
}
