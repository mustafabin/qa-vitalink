package models

import (
	"time"
)

type PaymentPage struct {
	MerchantID   string         `gorm:"primaryKey" json:"merchant_id"`
	PageUID      string         `gorm:"primaryKey" json:"page_uid"`
	AmountCents  int64          `json:"amount_cents"`
	Currency     string         `gorm:"default:USD" json:"currency"`
	Title        string         `json:"title"`
	Description  string         `json:"description"`
	StoreName    string         `json:"store_name"`
	Status       string         `gorm:"index" json:"status"`
	ExpireAt     *time.Time     `json:"expire_at"`


	InvoiceNo             string `json:"invoice_no"`
	IncludeTip            bool   `json:"include_tip"`
	AllowedTipPercentages string `gorm:"type:text" json:"allowed_tip_percentages" default:"[0.15,0.18,0.20]"`
	PaymentFeeAmount      string `json:"payment_fee_amount"`
	PaymentFeeDescription string `json:"payment_fee_description"`
	SurchargeAmount       string `json:"surcharge_amount"`
	TaxAmount             string `json:"tax_amount"`
	Items                 string `gorm:"type:text" json:"items" default:"[]"`

	PublicToken string `json:"public_token"`
	PaymentTypesAllowed string `gorm:"type:text" json:"payment_types_allowed" default:"CREDIT_DEBIT"`
	ApplePayMid string `json:"apple_pay_mid"`
	GooglePayMid string `json:"google_pay_mid"`
	FeatureGraphic string `json:"feature_graphic"`
	Logo string `json:"logo"`

	Last4 string `json:"last4" default:""`
	Brand string `json:"brand" default:""`

	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

func (p *PaymentPage) IsExpired(now time.Time) bool {
	if p.ExpireAt == nil {
		return false
	}
	return now.After(*p.ExpireAt)
}
