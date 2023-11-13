package src

import (
	"github.com/go-pg/pg/v10"
)

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

type LicenseStorer interface {
	GetLicenseByID(id string) (*License, error)
	GetLicenseByStripeID(id string) (*License, error)
	UpdateLicense(*License) error
	GetLicenseByEmail(email string) (*License, error)
	ExpireTrial(*License) (*License, error)
}

type licenseStore struct {
	db *pg.DB
}

func NewLicenseStore(db *pg.DB) *licenseStore {
	return &licenseStore{db: db}
}

// GetLicenseByID fetches a license from DB by license ID
func (store *licenseStore) GetLicenseByID(id string) (*License, error) {
	license := new(License)
	err := store.db.Model(license).Where("id = ?", id).Select()
	if err != nil {
		if err == pg.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return license, nil
}

func (store *licenseStore) GetLicenseByStripeID(stripeID string) (*License, error) {
	license := new(License)
	err := store.db.Model(license).Where("stripe_id = ?", stripeID).Select()
	if err != nil {
		if err == pg.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return license, nil
}

func (store *licenseStore) UpdateLicense(license *License) error {
	_, err := store.db.Model(license).Where("id = ?", license.ID).Update(license)
	return err
}

func (store *licenseStore) GetLicenseByEmail(email string) (*License, error) {
	license := new(License)
	err := store.db.Model(license).Where("email = ?", email).Select()
	if err != nil {
		if err == pg.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return license, nil
}

func (store *licenseStore) ExpireTrial(license *License) (*License, error) {
	license.IsValid = false
	license.ValidityReason = "trial license has expired"
	if err := store.UpdateLicense(license); err != nil {
		return license, err
	}
	return license, nil
}
