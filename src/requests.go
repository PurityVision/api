package src

type Requests struct {
	LicenseID string `json:"licenseID"`
	Count     int64  `json:"count"`
}

type RequestStore interface {
	GetRequests(licenseID string) *Requests
	UpdateRequests(licenseID string) error
}
