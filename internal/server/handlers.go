package server

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"math/big"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/skip2/go-qrcode"
	"gorm.io/gorm"

	"vitalink/internal/models"
	
)

var pageUIDLetters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func generatePageUID(length int) (string, error) {
	b := make([]rune, length)
	for i := range b {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(pageUIDLetters))))
		if err != nil {
			return "", err
		}
		b[i] = pageUIDLetters[idx.Int64()]
	}
	return string(b), nil
}

func grabConfig(token string) (string, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://api.vitabyte.info/api/config", nil) // todo change this to the prod url
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Add("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response struct {
		MerchantID string `json:"merchant_id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("error decoding response: %v", err)
	}

	return response.MerchantID, nil
}



func handleCreatePaymentPage(c echo.Context, db *gorm.DB) error {
	var req struct {
		MerchantID  string     `json:"merchant_id"`
		PageUID     string     `json:"page_uid"`
		RvcID	   string     `json:"rvc_id"`
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
		Items                 json.RawMessage `json:"items"`
		PaymentTypesAllowed   string `json:"payment_types_allowed"`
		PublicToken           string `json:"public_token"`
		ApplePayMid           string `json:"apple_pay_mid"`
		GooglePayMid          string `json:"google_pay_mid"`
		FeatureGraphic        string `json:"feature_graphic"`
		Logo                  string `json:"logo"`
		FavIcon               string `json:"favicon"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
	}
	if req.AmountCents == 0 {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "amount_cents is required"})
	}
	if req.MerchantID == "" {
		api_token := c.Request().Header.Get("Authorization")
		log.Println("api_token", api_token)
		merchantID, err := grabConfig(api_token)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]any{"error": "No merchant ID found and grabbing config failed", "details":err.Error()})
		}
		req.MerchantID = merchantID
	}
	if req.PageUID == "" {
		if s, err := generatePageUID(10); err == nil {
			req.PageUID = s
		} else {
			req.PageUID = strings.ReplaceAll(uuid.New().String()[:12], "-", "")
		}
	}
	if req.RvcID == "" {
		req.RvcID = "1" // default to 1 if not provided
	}

	if req.Currency == "" { req.Currency = "USD" }

	// Validate and normalize items to a JSON string
	itemsJSON := "[]"
	type incomingItem struct {
		Title       string  `json:"title"`
		Description string  `json:"description"`
		Price       float64 `json:"price"`
	}
	if len(req.Items) > 0 {
		var arr []incomingItem
		if err := json.Unmarshal(req.Items, &arr); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]any{"error": "items must be a JSON array of {title, description, price}"})
		}
		for i, it := range arr {
			if strings.TrimSpace(it.Title) == "" || strings.TrimSpace(it.Description) == "" {
				return c.JSON(http.StatusBadRequest, map[string]any{"error": fmt.Sprintf("items[%d] missing title or description", i)})
			}
			// price can be zero but not negative
			if it.Price < 0 {
				return c.JSON(http.StatusBadRequest, map[string]any{"error": fmt.Sprintf("items[%d] price must be >= 0", i)})
			}
		}
		b, _ := json.Marshal(arr)
		itemsJSON = string(b)
	}
	log.Println("Creating payment page for merchant:", req.MerchantID, "page UID:", req.PageUID)
	log.Println("Apple Pay MID:", req.ApplePayMid)

	pp := models.PaymentPage{
		MerchantID:  req.MerchantID,
		PageUID:     req.PageUID,
		RvcID:       req.RvcID,
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
		Items:                 itemsJSON,
		PaymentTypesAllowed:   req.PaymentTypesAllowed,
		PublicToken:           req.PublicToken,
		ApplePayMid:           req.ApplePayMid,
		GooglePayMid:          req.GooglePayMid,
		FeatureGraphic:        req.FeatureGraphic,
		Logo:                  req.Logo,
		FavIcon:               req.FavIcon,
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

	if pp.Status == "paid" {
		return c.Render(http.StatusOK, "paid.html", map[string]any{"page": pp})
	}

	if pp.Status != "open" || pp.IsExpired(time.Now()) {
		return c.Render(http.StatusOK, "expired.html", map[string]any{"page": pp})
	}
	log.Println("Rendering payment page for:", pp.MerchantID, pp.PageUID)
	log.Println("Apple Pay MID:", pp.ApplePayMid)
	log.Println("Google Pay MID:", pp.GooglePayMid)
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

func markPaymentFulfilled(ctx context.Context, db *gorm.DB, page *models.PaymentPage, dcResp map[string]any) error {
    if page == nil {
        return errors.New("nil payment page")
    }

    approved := false
    if v, ok := dcResp["Status"].(string); ok && strings.EqualFold(v, "Approved") {
        approved = true
    }
    if v, ok := dcResp["CmdStatus"].(string); ok && strings.EqualFold(v, "Approved") {
        approved = true
    }
    if !approved {
        return errors.New("transaction not approved")
    }

    // In case of dupes
    if page.Status == "paid" {
        return nil
    }

    tx := db.WithContext(ctx).Begin()
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
        }
    }()

    if err := tx.Model(page).Updates(map[string]any{
        "status": "paid",
        "last4":  dcResp["Last4"],
        "brand":  dcResp["Brand"],
    }).Error; err != nil {
        tx.Rollback()
        return fmt.Errorf("database update failed: %w", err)
    }

    if err := tx.Commit().Error; err != nil {
        return fmt.Errorf("transaction commit failed: %w", err)
    }

    page.Status = "paid"
    return nil
}
func getString(m map[string]any, key string) string {
    if v, ok := m[key]; ok {
        if s, ok := v.(string); ok {
            return s
        }
    }
    return ""
}
func handleChargePayment(c echo.Context, db *gorm.DB) error {
	merchantID := c.Param("merchant_id")
	pageUID := c.Param("page_uid")

	var page models.PaymentPage
	if err := db.First(&page, "merchant_id = ? AND page_uid = ?", merchantID, pageUID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.JSON(http.StatusNotFound, map[string]any{"error": "payment page not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "db error"})
	}
	if page.Status != "open" || page.IsExpired(time.Now()) {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "payment page closed or expired"})
	}

	var req struct {
		DatacapToken string `json:"datacap_token"`
		Last4 string `json:"last4"`
		Brand string `json:"brand"`
	}
	
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "invalid request"})
	}

	if strings.TrimSpace(req.DatacapToken) == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "datacap_token is required"})
	}
	log.Println("Charging payment for page:", page.MerchantID, page.PageUID)
	log.Println("Token:", req.DatacapToken)

	endpoint := "https://api.vitapay.com/v1/credit/sale"

	if page.AmountCents < 1 {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "amount must be at least 0.01"})
	}
	amount := fmt.Sprintf("%.2f", float64(page.AmountCents)/100)

	payload := map[string]string{
		"Token":        req.DatacapToken,
		"Amount":       amount,
		"Tax":          page.TaxAmount,
		"CustomerCode": page.InvoiceNo, // InvoiceNo
		"PartialAuth": "Disallow",
		"CardHolderID": "Allow_V2",
		"InvoiceNo":    page.InvoiceNo,
		"MerchantID":   page.MerchantID,
		"PageUID":      page.PageUID,
	}

	if page.InvoiceNo != "" {
		payload["InvoiceNo"] = page.InvoiceNo
	}
	if page.PaymentFeeAmount != "" {
		payload["PaymentFee"] = page.PaymentFeeAmount
	}
	if page.PaymentFeeDescription != "" {
		payload["PaymentFeeDescription"] = page.PaymentFeeDescription
	}
	if page.SurchargeAmount != "" {
		payload["SurchargeWithLookup"] = page.SurchargeAmount
	}
	if page.TaxAmount != "" {
		payload["Tax"] = page.TaxAmount
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "marshal error"})
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), 15*time.Second)
	defer cancel()

	reqHttp, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "request build error"})
	}
	reqHttp.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(reqHttp)
	if err != nil {
		return c.JSON(http.StatusBadGateway, map[string]any{"error": "datacap request failed", "details": err.Error()})
	}
	defer resp.Body.Close()
	respBytes, _ := io.ReadAll(resp.Body)

	var dcResp map[string]any
	_ = json.Unmarshal(respBytes, &dcResp)

	// Fallback to client-provided metadata if gateway response omits these
	if dcResp == nil { dcResp = map[string]any{} }
	if _, ok := dcResp["Last4"]; !ok && strings.TrimSpace(req.Last4) != "" { dcResp["Last4"] = req.Last4 }
	if _, ok := dcResp["Brand"]; !ok && strings.TrimSpace(req.Brand) != "" { dcResp["Brand"] = req.Brand }

	approved := false
	message := ""
	if v, ok := dcResp["Status"].(string); ok && strings.EqualFold(v, "Approved") { approved = true }
	if v, ok := dcResp["Message"].(string); ok && message == "" { message = v }
	if message == "" { message = strings.TrimSpace(string(respBytes)) }

	if resp.StatusCode >= 400 { approved = false }

	if approved {
		_ = markPaymentFulfilled(ctx, db, &page, dcResp)
		return c.JSON(http.StatusOK, map[string]any{
			"approved":  true,
			"message":   message,
		})
	}

	status := http.StatusBadRequest
	if resp.StatusCode >= 400 { status = resp.StatusCode }
	return c.JSON(status, map[string]any{
		"approved": false,
		"message":  message,
	})
}

