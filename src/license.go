package src

type License struct {
	ID             string `json:"id"`
	Email          string `json:"email"`
	StripeID       string `json:"stripeID"`
	SubscriptionID string `json:"subscriptionID"`
	IsValid        bool   `json:"isValid"`
	ValidityReason string `json:"validityReason"`
	RequestCount   int    `json:"requestCount"`
	IsTrial        bool   `json:"isTrial"`
}

type licenseStoreInterface interface {
	GetLicenseByID(id string) (*License, error)
	GetLicenseByStripeID(id string) (*License, error)
	UpdateLicense(license *License) error
}
