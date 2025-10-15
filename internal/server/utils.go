package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

func grabCheckValues(merchantID string, pageUID string) (*CheckResponse, error) {
	baseURL := "https://qa-vitasend-26847c8e9c67.herokuapp.com"// todo change to api.vitapay.com
	url := fmt.Sprintf("%s/check/%s/%s", baseURL, merchantID, pageUID)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var checkResponse CheckResponse
	log.Println("body: ", string(body))
	err = json.Unmarshal(body, &checkResponse)
	if err != nil {
		return nil, err
	}
	return &checkResponse, nil
}

type CheckResponse struct {
	ID uint `json:"id"`;
	MerchantID string `json:"merchant_id"`;
	CheckUID string `json:"check_uid"`;
	SimphonyCheckID string `json:"simphony_check_id"`;
	RvcID string `json:"rvc_id"`;
	AmountCents int64 `json:"amount_cents"`;
	Currency string `json:"currency"`;
	Status string `json:"status"`;
	PageUID string `json:"page_uid"`;
	Title string `json:"title"`;
	Description string `json:"description"`;
	StoreName string `json:"store_name"`;
	InvoiceNo string `json:"invoice_no"`;
	IncludeTip bool `json:"include_tip"`;
	AllowedTipPercentages string `json:"allowed_tip_percentages"`;
	PaymentFeeAmount string `json:"payment_fee_amount"`;
	PaymentFeeDescription string `json:"payment_fee_description"`;
	SurchargeAmount string `json:"surcharge_amount"`;
	TaxAmount string `json:"tax_amount"`;
	Items string `json:"items"`;
	Last4 string `json:"last4"`;
	Brand string `json:"brand"`;
	PublicToken string `json:"public_token"`;
	PaymentTypesAllowed string `json:"payment_types_allowed"`;
	ApplePayMid string `json:"apple_pay_mid"`;
	GooglePayMid string `json:"google_pay_mid"`;
	FeatureGraphic string `json:"feature_graphic"`;
	Logo string `json:"logo"`;
	Logo2 string `json:"logo2"`;
	FavIcon string `json:"favicon"`;
	Environment string `json:"environment"`;
	WebhookURL string `json:"webhook_url"`;
	CreatedAt time.Time `json:"created_at"`;
	UpdatedAt time.Time `json:"updated_at"`;
	ExpireAt *time.Time `json:"expire_at"`;
	Payments []CheckPayments `json:"payments"`;
	UserID uint `json:"user_id"`;
}
type CheckPayments struct {
	ID uint `json:"id"`;
	Last4 string `json:"last4"`;
	Brand string `json:"brand"`;
	AmountCents int64 `json:"amount_cents"`;
	Currency string `json:"currency"`;
	InvoiceNo string `json:"invoice_no"`;
	CheckID uint `json:"check_id"`;
	UserID uint `json:"user_id"`;
	CreatedAt time.Time `json:"created_at"`;
	UpdatedAt time.Time `json:"updated_at"`;
}
