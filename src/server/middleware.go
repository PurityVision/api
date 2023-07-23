package server

import (
	"errors"
	"net/http"
	"purity-vision-filter/src/license"

	"github.com/google/uuid"
)

func getLicenseFromReq(ls license.LicenseStore, r *http.Request) (*license.License, error) {
	licenseID := r.Header.Get("LicenseID")

	_, err := uuid.Parse(licenseID)
	if err != nil {
		return nil, errors.New("invalid license ID")
	}

	license, err := ls.GetLicenseByID(licenseID)
	if err != nil || license == nil {
		return nil, errors.New("invalid license")
	}

	return license, nil
}

func paywallMiddleware(ls license.LicenseStore) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			license, err := getLicenseFromReq(ls, r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			// licenseID := r.Header.Get("LicenseID")

			// _, err := uuid.Parse(licenseID)
			// if err != nil {
			// 	http.Error(w, "Invalid license ID", http.StatusUnauthorized)
			// 	return
			// }

			// license, err := licenseStore.GetLicenseByID(licenseID)
			// if err != nil || license == nil {
			// 	http.Error(w, "Invalid license", http.StatusUnauthorized)
			// 	return
			// }

			if !license.IsValid {
				http.Error(w, "Expired license", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

var addCorsHeaders = func(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowedHeaders := "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization, X-CSRF-Token, licenseID"
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
		w.Header().Set("Access-Control-Expose-Headers", "Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
