package src

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/customer"
	"github.com/stripe/stripe-go/v74/webhook"
)

func health(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte("All Good ☮️"))
}

const MAX_IMAGES_PER_REQUEST = 16

func removeDuplicates(logger zerolog.Logger, vals []string) []string {
	res := make([]string, 0)
	strMap := make(map[string]bool, 0)

	for _, v := range vals {
		if found := strMap[v]; found {
			logger.Info().Msgf("found duplicate image: %s in request. Removing.", v)
			continue
		}
		res = append(res, v)
		strMap[v] = true

	}

	return res
}
func handleBatchFilter(ctx appContext, w http.ResponseWriter, req *http.Request) (int, error) {
	var filterReqPayload AnnotateReq

	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&filterReqPayload); err != nil {
		return http.StatusBadRequest, errors.New("JSON body missing or malformed")
	}

	if len(filterReqPayload.ImgURIList) == 0 {
		return http.StatusBadRequest, errors.New("ImgUriList cannot be empty")
	}

	var res []*ImageAnnotation

	uris := removeDuplicates(ctx.logger, filterReqPayload.ImgURIList)

	for _, uri := range uris {
		if _, err := url.ParseRequestURI(uri); err != nil {
			return http.StatusBadRequest, fmt.Errorf("%s is not a valid URI", uri)
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

		temp, err := filterImages(ctx, uris[i:endIdx], req.Header.Get("LicenseID"))
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("error while filtering: %s", err)
		}

		res = append(res, temp...)

		i += MAX_IMAGES_PER_REQUEST
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
	return http.StatusOK, nil
}

func handleWebhook(ctx appContext, w http.ResponseWriter, req *http.Request) (int, error) {
	const MaxBodyBytes = int64(65536)
	req.Body = http.MaxBytesReader(w, req.Body, MaxBodyBytes)
	payload, err := io.ReadAll(req.Body)
	if err != nil {
		return http.StatusServiceUnavailable, fmt.Errorf("error reading request body: %v", err)
	}

	endpointSecret := ctx.config.StripeWebhookSecret
	event, err := webhook.ConstructEvent(payload, req.Header.Get("Stripe-Signature"), endpointSecret)

	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("error verifying webhook signature: %v", err)
	}

	// Unmarshal the event data into an appropriate struct depending on its Type
	switch event.Type {
	// case "customer.subscription.created"
	case "checkout.session.completed":
		var session stripe.CheckoutSession
		err := json.Unmarshal(event.Data.Raw, &session)
		if err != nil {
			return http.StatusBadRequest, fmt.Errorf("error parsing webhook JSON: %v", err.Error())
		}

		subscriptionID := session.Subscription.ID
		stripeID := session.Customer.ID
		email := session.CustomerDetails.Email

		license, err := ctx.licenseStore.GetLicenseByStripeID(stripeID)
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("error fetching license: %v", err)
		}

		// if license exists ensure IsValid is true and return
		if license != nil {
			ctx.logger.Debug().Msg("existing license found, ensuring IsValid is true")
			license.IsValid = true
			if err := ctx.licenseStore.UpdateLicense(license); err != nil {
				return http.StatusInternalServerError, errors.New("")
			}
			// TODO: email person to remind them their subscription is renewed.
			return http.StatusOK, nil
		}

		// else create new license and store in db
		licenseID := GenerateLicenseKey()
		ctx.logger.Info().Msgf("generating new license: %s", licenseID)

		license = &License{
			ID:             licenseID,
			Email:          email,
			StripeID:       stripeID,
			SubscriptionID: subscriptionID,
			IsValid:        true,
			ValidityReason: "",
			RequestCount:   0,
		}

		if _, err = ctx.db.Model(license).Insert(); err != nil {
			return http.StatusInternalServerError, fmt.Errorf("error creating license: %v", err)
		}

		stripe.Key = ctx.config.StripeKey
		metadata := map[string]string{
			"license": licenseID,
		}
		if _, err := customer.Update(session.Customer.ID, &stripe.CustomerParams{
			Params: stripe.Params{Metadata: metadata},
		}); err != nil {
			return http.StatusInternalServerError, fmt.Errorf("error adding license to customer metadata: %v", err)
		}

		if err = SendLicenseMail(ctx.config, license.Email, license.ID); err != nil {
			// TODO: retry sending email so user can get their license.
			return http.StatusInternalServerError, fmt.Errorf("error sending license email: %v", err)
		}
	case "customer.subscription.updated":
		sub := stripe.Subscription{}
		err := json.Unmarshal(event.Data.Raw, &sub)
		if err != nil {
			return http.StatusBadRequest, fmt.Errorf("error parsing webhook JSON: %v", err)
		}

		license, err := ctx.licenseStore.GetLicenseByStripeID(sub.Customer.ID)
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("error finding license for valid subscriber: %v", err)
		}
		if license == nil {
			return http.StatusInternalServerError, errors.New("failed to find license")
		}

		if sub.CancellationDetails.Reason != "" {
			license.IsValid = false
			license.ValidityReason = fmt.Sprintf("subscription was cancelled: %s", sub.CancellationDetails.Reason)
			ctx.logger.Info().Msgf("invalidated license: %s", license.ID)
		} else {
			license.IsValid = true
			license.ValidityReason = ""
			ctx.logger.Info().Msgf("activated license: %s", license.ID)
		}

		if err = ctx.licenseStore.UpdateLicense(license); err != nil {
			return http.StatusInternalServerError, fmt.Errorf("error updating license: %v", err)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unhandled event type: %s", event.Type)
	}

	w.WriteHeader(http.StatusOK)
	return http.StatusOK, nil
}

func handleGetLicense(ctx appContext, w http.ResponseWriter, req *http.Request) (int, error) {
	vars := mux.Vars(req)
	licenseID := vars["id"]

	if licenseID == "" {
		return http.StatusBadRequest, errors.New("licenseID path parameter was empty")
	}

	ctx.logger.Info().Msgf("verifying license: %s", licenseID)

	license, err := ctx.licenseStore.GetLicenseByID(licenseID)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to get license: %s", err.Error())
	}

	// if license == nil {
	// 	w.WriteHeader(http.StatusBadRequest)
	// 	fmt.Fprintf(w, "License %s was not found", licenseID)
	// 	return
	// }

	json.NewEncoder(w).Encode(license)
	return http.StatusOK, nil
}

type TrialRegisterReq struct {
	Email string
}

// func handleTrialRegister(ctx *appContext, w http.ResponseWriter, req *http.Request) {
// 	var trialReq TrialRegisterReq

// 	decoder := json.NewDecoder(req.Body)
// 	if err := decoder.Decode(&trialReq); err != nil {
// 		writeError(ctx.logger, 400, "JSON body missing or malformed", w)
// 		return
// 	}

// 	if trialReq.Email == "" {
// 		fmt.Fprint(w, "Email cannot be empty")
// 		w.WriteHeader(http.StatusBadRequest)
// 		return
// 	}

// 	license, err := ctx.licenseStore.GetLicenseByEmail(trialReq.Email)
// 	if err != nil {
// 		ctx.logger.Error().Msgf("failed to fetch license by email: %s", err.Error())
// 		w.WriteHeader(http.StatusInternalServerError)
// 		return
// 	}

// 	if license != nil {
// 		ctx.logger.Error().Msg("email is already registered")
// 		w.WriteHeader(http.StatusBadRequest)
// 		fmt.Fprint(w, "email is already registered")
// 		return
// 	}

// 	if err = RegisterNewUser(ctx.config, trialReq.Email); err != nil {
// 		ctx.logger.Error().Msgf("something went wrong registering a new user: %v", err)
// 		w.WriteHeader(http.StatusInternalServerError)
// 		return
// 	}

// 	w.WriteHeader(http.StatusOK)
// }

func RegisterNewUser(ctx appContext, email string) error {
	licenseID := GenerateLicenseKey()
	ctx.logger.Info().Msgf("generated license: %s", licenseID)

	license := &License{
		ID:             licenseID,
		Email:          email,
		StripeID:       "trial",
		IsValid:        true,
		ValidityReason: "",
		RequestCount:   0,
	}

	if _, err := ctx.db.Model(license).Insert(); err != nil {
		ctx.logger.Error().Msgf("error creating: %v", err)
		return err
	}

	if err := SendLicenseMail(ctx.config, license.Email, license.ID); err != nil {
		// TODO: retry sending email so user can get their license.
		ctx.logger.Error().Msgf("error sending license email: %v", err)
		return err
	}

	return nil
}
