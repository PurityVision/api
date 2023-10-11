package license

type License struct {
	ID             string `json:"id"`
	Email          string `json:"email"`
	StripeID       string `json:"stripeID"`
	SubscriptionID string `json:"subscriptionID"`
	IsValid        bool   `json:"isValid"`
	RequestCount   int    `json:"requestCount"`
}

type LicenseStore interface {
	GetLicenseByID(id string) (*License, error)
	GetLicenseByStripeID(id string) (*License, error)
	UpdateLicense(license *License) error
}
