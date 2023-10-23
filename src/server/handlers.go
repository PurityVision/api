package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"purity-vision-filter/src/images"
	"purity-vision-filter/src/mail"
	"purity-vision-filter/src/utils"

	lic "purity-vision-filter/src/license"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/webhook"
)

func health(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte("All Good ☮️"))
}

const MAX_IMAGES_PER_REQUEST = 16

func removeDuplicates(vals []string) []string {
	res := make([]string, 0)
	strMap := make(map[string]bool, 0)

	for _, v := range vals {
		if found := strMap[v]; found == true {
			logger.Debug().Msgf("Found duplicate image: %s in request. Removing.", v)
			continue
		}
		res = append(res, v)
		strMap[v] = true

	}

	return res
}

func handleBatchFilter(logger zerolog.Logger) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		var filterReqPayload AnnotateReq

		decoder := json.NewDecoder(req.Body)
		if err := decoder.Decode(&filterReqPayload); err != nil {
			writeError(400, "JSON body missing or malformed", w)
			return
		}

		if len(filterReqPayload.ImgURIList) == 0 {
			writeError(400, "ImgUriList cannot be empty", w)
			return
		}

		var res []*images.ImageAnnotation

		uris := removeDuplicates(filterReqPayload.ImgURIList)

		// Validate the request payload URIs
		for _, uri := range uris {
			if _, err := url.ParseRequestURI(uri); err != nil {
				writeError(400, fmt.Sprintf("%s is not a valid URI", uri), w)
				return
			}
		}

		// Filter images in pages of size MAX_IMAGES_PER_REQUEST.
		for i := 0; i < len(uris); {
			var endIdx int
			if i+MAX_IMAGES_PER_REQUEST > len(uris)-1 {
				endIdx = len(uris)
			} else {
				endIdx = i + MAX_IMAGES_PER_REQUEST
			}

			temp, err := filterImages(uris[i:endIdx], req.Header.Get("LicenseID"))
			if err != nil {
				logger.Error().Msgf("error while filtering: %s", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			res = append(res, temp...)

			i += MAX_IMAGES_PER_REQUEST
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
	}
}

func handleWebhook(w http.ResponseWriter, req *http.Request) {
	const MaxBodyBytes = int64(65536)
	req.Body = http.MaxBytesReader(w, req.Body, MaxBodyBytes)
	payload, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logger.Debug().Msgf("Error reading request body: %v", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	endpointSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	event, err := webhook.ConstructEvent(payload, req.Header.Get("Stripe-Signature"),
		endpointSecret)

	if err != nil {
		logger.Debug().Msgf("error verifying webhook signature: %v", err)
		w.WriteHeader(http.StatusBadRequest) // Return a 400 error on a bad signature
		return
	}

	// Unmarshal the event data into an appropriate struct depending on its Type
	switch event.Type {
	// case "customer.subscription.created"
	case "invoice.payment_succeeded":
		invoice := stripe.Invoice{}
		err := json.Unmarshal(event.Data.Raw, &invoice)
		if err != nil {
			fmt.Fprintf(w, "error parsing webhook JSON: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		subscriptionID := invoice.Subscription.ID
		stripeID := invoice.Customer.ID
		email := invoice.CustomerEmail

		license, err := pgStore.GetLicenseByStripeID(stripeID)
		if err != nil {
			logger.Debug().Msgf("Error fetching license: %v", err)
			PrintSomethingWrong(w)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// if license exists ensure IsValid is true and return
		if license != nil {
			logger.Debug().Msg("Existing license found, ensuring IsValid is true")
			license.IsValid = true
			if err := pgStore.UpdateLicense(license); err != nil {
				PrintSomethingWrong(w)
				w.WriteHeader(http.StatusInternalServerError)
			}
			// TODO: email person to remind them their subscription is renewed.
			return
		}

		// else create new license and store in db
		logger.Debug().Msg("No license found. Creating one")
		licenseID := utils.GenerateLicenseKey()
		logger.Debug().Msgf("Generated license: %s", licenseID)

		license = &lic.License{
			ID:             licenseID,
			Email:          email,
			StripeID:       stripeID,
			SubscriptionID: subscriptionID,
			IsValid:        true,
		}

		if _, err = conn.Model(license).Insert(); err != nil {
			logger.Debug().Msgf("error creating: %v", err)
			PrintSomethingWrong(w)
			w.WriteHeader(http.StatusInternalServerError)
		}

		if err = mail.SendLicenseMail(license.Email, license.ID); err != nil {
			// TODO: retry sending email so user can get their license.
			logger.Debug().Msgf("error sending license email: %v", err)
			PrintSomethingWrong(w)
			w.WriteHeader(http.StatusInternalServerError)
		}
	case "customer.subscription.updated":
		sub := stripe.Subscription{}
		err := json.Unmarshal(event.Data.Raw, &sub)
		if err != nil {
			fmt.Fprintf(w, "error parsing webhook JSON: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		license, err := pgStore.GetLicenseByStripeID(sub.Customer.ID)
		if err != nil {
			logger.Debug().Msgf("error finding license for valid subscriber: %v", err)
		}
		if license == nil {
			logger.Debug().Msg("failed to find license for existing subscriber. Something is terribly wrong")
			PrintSomethingWrong(w)
			return
		}

		if sub.CancellationDetails.Reason != "" {
			license.IsValid = false
			logger.Debug().Msgf("invalidated license: %s", license.ID)
		} else {
			license.IsValid = true
			logger.Debug().Msgf("activated license: %s", license.ID)
		}

		if err = pgStore.UpdateLicense(license); err != nil {
			logger.Debug().Msgf("error updating license: %v", err)
			PrintSomethingWrong(w)
			return
		}
	default:
		fmt.Fprintf(os.Stderr, "Unhandled event type: %s", event.Type)
	}

	w.WriteHeader(http.StatusOK)
}

func getLicense(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	licenseID := vars["id"]

	logger.Debug().Msgf("verifying license: %s", licenseID)

	license, err := pgStore.GetLicenseByID(licenseID)
	if err != nil {
		logger.Error().Msgf("failed to get license: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		PrintSomethingWrong(w)
		return
	}

	// if license == nil {
	// 	w.WriteHeader(http.StatusBadRequest)
	// 	fmt.Fprintf(w, "License %s was not found", licenseID)
	// 	return
	// }

	json.NewEncoder(w).Encode(license)
	w.WriteHeader(http.StatusOK)
}

type TrialRegisterReq struct {
	Email string
}

func handleTrialRegister(w http.ResponseWriter, req *http.Request) {
	var trialReq TrialRegisterReq

	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&trialReq); err != nil {
		writeError(400, "JSON body missing or malformed", w)
		return
	}

	if trialReq.Email == "" {
		fmt.Fprint(w, "Email cannot be empty")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	license, err := pgStore.GetLicenseByEmail(trialReq.Email)
	if err != nil {
		logger.Debug().Msgf("failed to fetch license by email: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if license != nil {
		logger.Debug().Msg("email is already registered")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "email is already registered")
		return
	}

	if err = RegisterNewUser(trialReq.Email); err != nil {
		logger.Debug().Msgf("something went wrong registering a new user: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func RegisterNewUser(email string) error {
	licenseID := utils.GenerateLicenseKey()
	logger.Debug().Msgf("generated license: %s", licenseID)

	license := &lic.License{
		ID:       licenseID,
		Email:    email,
		StripeID: "trial",
		IsValid:  true,
	}

	if _, err := conn.Model(license).Insert(); err != nil {
		logger.Debug().Msgf("error creating: %v", err)
		return err
	}

	if err := mail.SendLicenseMail(license.Email, license.ID); err != nil {
		// TODO: retry sending email so user can get their license.
		logger.Debug().Msgf("error sending license email: %v", err)
		return err
	}

	return nil
}
