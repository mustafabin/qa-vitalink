package server

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	req, err := http.NewRequest("GET", "https://api.vitapay.com/api/config", nil)
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
		RvcID       string     `json:"rvc_id"`
		AmountCents int64      `json:"amount_cents" validate:"required"`
		Currency    string     `json:"currency"`
		Title       string     `json:"title"`
		Description string     `json:"description"`
		StoreName   string     `json:"store_name"`
		ExpireAt    *time.Time `json:"expire_at"`

		InvoiceNo             string          `json:"invoice_no"`
		IncludeTip            bool            `json:"include_tip"`
		AllowedTipPercentages string          `json:"allowed_tip_percentages"`
		PaymentFeeAmount      string          `json:"payment_fee_amount"`
		PaymentFeeDescription string          `json:"payment_fee_description"`
		SurchargeAmount       string          `json:"surcharge_amount"`
		TaxAmount             string          `json:"tax_amount"`
		Items                 json.RawMessage `json:"items"`
		PaymentTypesAllowed   string          `json:"payment_types_allowed"`
		PublicToken           string          `json:"public_token"`
		ApplePayMid           string          `json:"apple_pay_mid"`
		GooglePayMid          string          `json:"google_pay_mid"`
		FeatureGraphic        string          `json:"feature_graphic"`
		Logo                  string          `json:"logo"`
		Logo2                 string          `json:"logo2"`
		FavIcon               string          `json:"favicon"`
		Environment           string          `json:"environment"`
		WebhookURL            string          `json:"webhook_url" default:""`
		Metadata              map[string]interface{} `json:"metadata"`
	}

	log.Println("Create payment page request received")
	// pretty logging of request body
	bodyBytes, err := io.ReadAll(c.Request().Body)
	if err != nil {
		log.Println("Error reading request body:", err)
	} else {
		log.Println("Request body:", string(bodyBytes))
	}
	c.Request().Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

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
			return c.JSON(http.StatusBadRequest, map[string]any{"error": "No merchant ID found and grabbing config failed", "details": err.Error()})
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

	if req.Currency == "" {
		req.Currency = "USD"
	}

	// Validate and normalize items to a JSON string
	itemsJSON := "[]"
	type incomingItem struct {
		Title       string  `json:"title"`
		Description string  `json:"description"`
		Price       float64 `json:"price"`
		Quantity    int     `json:"quantity"`
		Total       float64 `json:"total"`
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
	webhookURL := "https://61dd73d7a2cceb93bc904c56be0e18.08.environment.api.powerplatform.com:443/powerautomate/automations/direct/workflows/74363f9423c74bc7819b633a39f8609c/triggers/manual/paths/invoke?api-version=1&sp=%2Ftriggers%2Fmanual%2Frun&sv=1.0&sig=CZ-zRD_5XBW4L86VS0TluoQ_Zue25n_XWb3CJOhZyuc"
	if req.WebhookURL != "" && strings.HasPrefix(req.WebhookURL, "http") {
		webhookURL = req.WebhookURL
	}
	// if webhook url is in the metadata, use it and it has http prefix
	if v, ok := req.Metadata["webhook_url"]; ok && strings.HasPrefix(v.(string), "http") {
		webhookURL = v.(string)
	}

	if req.InvoiceNo == "" {
		req.InvoiceNo = req.PageUID
	}
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
		Logo2:                 req.Logo2,
		FavIcon:               req.FavIcon,
		Environment:           req.Environment,
		WebhookURL:            webhookURL,
	}

	if err := db.Create(&pp).Error; err != nil {
		if isUnique(err) {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error":   "payment page exists",
				"details": err.Error(),
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error":   "create failed",
			"details": err.Error(),
		})
	}

	scheme := "https"
	if c.Scheme() != "" {
		scheme = c.Scheme()
	}
	host := c.Request().Host
	base := scheme + "://" + host
	paymentURL := base + "/p/" + pp.MerchantID + "/" + pp.PageUID
	qrURL := base + "/qr/" + pp.MerchantID + "/" + pp.PageUID

	return c.JSON(http.StatusCreated, map[string]any{
		"payment_url": paymentURL,
		"qr_url":      qrURL,
	})
}

func handleViewPaymentPage(c echo.Context) error {
	merchantID := c.Param("merchant_id")
	pageUID := c.Param("page_uid")

	pp, err := grabCheckValues(merchantID, pageUID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "error grabbing check values"})
	}

	if len(pp.Payments) > 0 {
		paymentJSON, _ := json.Marshal(pp.Payments[0])
		log.Println("Payments:", string(paymentJSON))
	}
	paymentsJSON, _ := json.Marshal(pp.Payments)
	log.Println("Payments", string(paymentsJSON))
	var amountPaid int64
	for _, payment := range pp.Payments {
		amountPaid += payment.AmountCents
	}
	log.Println("Amount paid:", amountPaid)
	log.Println("Amount due:", pp.AmountCents)
	log.Println("is payments equal or greater than amount due?", amountPaid >= pp.AmountCents)
	if pp.Status == "paid" {
		return c.Render(http.StatusOK, "paid.html", map[string]any{"page": pp, "paymentsJSON": string(paymentsJSON)})
	}

	if pp.ExpireAt != nil && pp.ExpireAt.Before(time.Now()) {
		return c.Render(http.StatusOK, "expired.html", map[string]any{"page": pp})
	}
	log.Println("Rendering payment page for:", pp.MerchantID, pp.PageUID)
	log.Println("Apple Pay MID:", pp.ApplePayMid)
	log.Println("Google Pay MID:", pp.GooglePayMid)

	return c.Render(http.StatusOK, "payment_revised.html", map[string]any{
		"page": pp,
		"paymentsJSON": string(paymentsJSON),
	})
}

func handleQRPaymentPage(c echo.Context) error {
	merchantID := c.Param("merchant_id")
	pageUID := c.Param("page_uid")

	scheme := "https"
	if c.Scheme() != "" {
		scheme = c.Scheme()
	}
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
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "duplicate key value") || strings.Contains(strings.ToLower(s), "unique")
}

func handleFetchPaymentPageData(c echo.Context, db *gorm.DB) error {
	merchantID := c.Param("merchant_id")
	pageUID := c.Param("page_uid")

	// First get the local payment page data
	var pp models.PaymentPage
	err := db.First(&pp, "merchant_id = ? AND page_uid = ?", merchantID, pageUID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return c.JSON(http.StatusNotFound, map[string]any{"error": "payment page not found"})
	} else if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "database error"})
	}

	// Try to fetch updated data from api.vitapay.com
	client := &http.Client{Timeout: 10 * time.Second}
	apiURL := fmt.Sprintf("https://api.vitapay.com/check/%s/%s", merchantID, pageUID)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		// Return local data if request creation fails
		return c.JSON(http.StatusOK, map[string]any{
			"success": false,
			"message": "Using local data",
			"data":    pp,
		})
	}

	req.Header.Set("User-Agent", "VitaPay/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		// Return local data if API call fails
		if resp != nil {
			resp.Body.Close()
		}
		return c.JSON(http.StatusOK, map[string]any{
			"success": false,
			"message": "Using local data",
			"data":    pp,
		})
	}
	defer resp.Body.Close()

	var apiData models.PaymentPage
	if err := json.NewDecoder(resp.Body).Decode(&apiData); err != nil {
		// Return local data if parsing fails
		return c.JSON(http.StatusOK, map[string]any{
			"success": false,
			"message": "Using local data",
			"data":    pp,
		})
	}

	// Update local database with fresh data if it's different
	if apiData.AmountCents != pp.AmountCents || apiData.Status != pp.Status || apiData.Items != pp.Items ||
		apiData.IncludeTip != pp.IncludeTip || apiData.AllowedTipPercentages != pp.AllowedTipPercentages {
		pp.AmountCents = apiData.AmountCents
		pp.Status = apiData.Status
		pp.Items = apiData.Items
		pp.IncludeTip = apiData.IncludeTip
		pp.AllowedTipPercentages = apiData.AllowedTipPercentages
		pp.UpdatedAt = time.Now()
		// Use selective update to avoid overwriting ExpireAt
		db.Model(&pp).Updates(map[string]any{
			"amount_cents":            pp.AmountCents,
			"status":                  pp.Status,
			"items":                   pp.Items,
			"include_tip":             pp.IncludeTip,
			"allowed_tip_percentages": pp.AllowedTipPercentages,
			"updated_at":              pp.UpdatedAt,
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
		"message": "Data updated from API",
		"data":    pp,
	})
}

func handleChargePayment(c echo.Context) error {
	merchantID := c.Param("merchant_id")
	pageUID := c.Param("page_uid")

	page, err := grabCheckValues(merchantID, pageUID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "error grabbing check values"})
	}
	if page.ExpireAt != nil && page.ExpireAt.Before(time.Now()) {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "payment page closed or expired"})
	}

	pageJSON, _ := json.Marshal(page)
	log.Println("Page details:", string(pageJSON))
	log.Println("Webhook URL:", page.WebhookURL)
	var req struct {
		DatacapToken   string `json:"datacap_token"`
		Last4          string `json:"last4"`
		Brand          string `json:"brand"`
		TipAmountCents int64  `json:"tip_amount_cents"`
		TaxAmountCents int64  `json:"tax_amount_cents"`
		AmountCents    int64  `json:"amount_cents"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "invalid request"})
	}

	if strings.TrimSpace(req.DatacapToken) == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "datacap_token is required"})
	}
	log.Println("Charging payment for page:", page.MerchantID, page.PageUID)

	baseURL := "https://qa-vitasend-26847c8e9c67.herokuapp.com"
	endpoint := baseURL + "/v1/credit/sale"

	if page.AmountCents < 1 {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "amount must be at least 0.01"})
	}

	if req.AmountCents < 1 { // defaults to page amount if not provided
		req.AmountCents = page.AmountCents
	}

	log.Println("Calculating total amount including tip ...")

	amount := fmt.Sprintf("%.2f", float64(req.AmountCents)/100)
	tipAmount := fmt.Sprintf("%.2f", float64(req.TipAmountCents)/100)
	taxAmount := fmt.Sprintf("%.2f", float64(req.TaxAmountCents)/100)

	log.Println("Total amount:", amount)
	log.Println("Tip amount:", tipAmount)
	log.Println("Tax amount:", taxAmount)

	payload := map[string]string{
		"Token":        req.DatacapToken,
		"Amount":       amount,
		"Tip":    tipAmount,
		"Tax":    taxAmount,
		"CustomerCode": page.InvoiceNo,
		"PartialAuth":  "Disallow",
		"CardHolderID": "Allow_V2",
		"InvoiceNo":    page.InvoiceNo,
		"MerchantID":   page.MerchantID,
		"PageUID":      page.PageUID,
		"WebhookURL":   page.WebhookURL,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "marshal error"})
	}

	log.Println("forwarding to main server ...")

	reqHttp, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		log.Println("request build error:", err)
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "request build error"})
	}
	reqHttp.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(reqHttp)

	if err != nil {
		log.Println("vitapay request failed:", err)
		return c.JSON(http.StatusBadGateway, map[string]any{"error": "datacap request failed", "details": err.Error()})
	}
	defer resp.Body.Close()
	respBytes, _ := io.ReadAll(resp.Body)

	var dcResp map[string]any
	_ = json.Unmarshal(respBytes, &dcResp)

	if dcResp == nil {
		dcResp = map[string]any{}
	}

	approved := false
	approvedAmount := 0.0
	dcRespJSON, _ := json.Marshal(dcResp)
	log.Println("Datacap response:", string(dcRespJSON))
	log.Println("Approved amount:", dcResp["ApprovedAmount"])
	message := ""
	if v, ok := dcResp["Status"].(string); ok && strings.EqualFold(v, "Approved") {
		approved = true
	}
	if v, ok := dcResp["Message"].(string); ok && message == "" {
		message = v
	}
	approvedAmount, err = strconv.ParseFloat(dcResp["ApprovedAmount"].(string), 64)
	if err != nil {
		log.Println("Error parsing approved amount:", err)
		approvedAmount = 0.0
	}
	cardholderId := ""
	if v, ok := dcResp["CardHolderID"].(string); ok {
		cardholderId = v
	}

	if message == "" {
		message = strings.TrimSpace(string(respBytes))
	}

	if resp.StatusCode >= 400 {
		approved = false
	}

	if approved {
		log.Println("Payment approved sending to webhook ... ", page.WebhookURL , "amount: ", approvedAmount, "resp: ", resp, "invoice no: ", page.InvoiceNo)
		//cardholder id
		log.Println("Cardholder ID:", cardholderId)

		webhookData := map[string]interface{}{
			"type":"object",
			"properties": map[string]interface{}{
				"invoiceId": page.InvoiceNo,
				"paymentAmount": approvedAmount,
				"paymentMethod": "Credit Card",
				"transactionId": page.PageUID,
			},
			"required": []string{"invoiceId", "paymentAmount", "paymentMethod", "transactionId"},
		}
		if page.InvoiceNo == "" {
			webhookData["properties"].(map[string]interface{})["invoiceId"] = page.PageUID
		}
		if cardholderId != "" {
			webhookData["properties"].(map[string]interface{})["cardholderId"] = cardholderId
		}
		SendToWebhook(page.WebhookURL, webhookData)
		return c.JSON(http.StatusOK, map[string]any{
			"approved": true,
			"message":  message,
		})
	}

	status := http.StatusBadRequest
	if resp.StatusCode >= 400 {
		status = resp.StatusCode
	}
	return c.JSON(status, map[string]any{
		"approved": false,
		"message":  message,
	})
}

func SendToWebhook(webhookURL string, data map[string]interface{}) error {
	if webhookURL == "" {
		log.Println("No webhook URL provided skipping webhook logic")
		return nil
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	log.Println("Sending to webhook: ", webhookURL, "data: ", data)
	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Println("Error creating request: ", err)
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "VitaPay/1.0")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, _ := client.Do(req)

	if resp.StatusCode != http.StatusOK {
		log.Println("Error sending to webhook: ", resp.StatusCode)
		respBytes, _ := io.ReadAll(resp.Body)
		log.Println("Error sending to webhook: ", resp.StatusCode, "data: ", string(respBytes))
		return fmt.Errorf("webhook returned: %d, %s", resp.StatusCode, string(respBytes))
	}
	log.Println("webhook sent successfully")

	defer resp.Body.Close()
	return nil
}
