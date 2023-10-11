package server

import (
	"bufio"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"purity-vision-filter/src/config"
	"purity-vision-filter/src/images"
	"purity-vision-filter/src/license"
	"purity-vision-filter/src/utils"
	"purity-vision-filter/src/vision"
	"time"

	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/subscription"
	"github.com/stripe/stripe-go/v74/usagerecord"
	pb "google.golang.org/genproto/googleapis/cloud/vision/v1"
)

func filterImages(uris []string, licenseID string) ([]*images.ImageAnnotation, error) {
	var res []*images.ImageAnnotation
	license, err := pgStore.GetLicenseByID(licenseID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch license: %s", err.Error())
	}
	if license == nil {
		return nil, errors.New("license not found")
	}

	annotations, err := images.FindAnnotationsByURI(conn, uris)
	if err != nil {
		return nil, err
	}

	newURIs := make([]string, 0)

	for _, uri := range uris {
		found := false
		for _, anno := range annotations {
			if anno.URI == uri {
				res = append(res, &anno)
				found = true
				logger.Debug().Msgf("Found cached image: %s", uri)
				break
			}
		}
		if !found {
			newURIs = append(newURIs, uri)
		}
	}

	uris = newURIs

	batchRes, err := vision.BatchGetImgAnnotation(uris)
	if err != nil {
		return nil, err
	}

	defer func(quantity int) {
		if quantity < 1 {
			return
		}

		license.RequestCount += len(batchRes.Responses)
		if err = pgStore.UpdateLicense(license); err != nil {
			logger.Debug().Msgf("failed to update license request count: %s", err.Error())
		}

		if err = incrementSubscriptionMeter(license, int64(quantity)); err != nil {
			logger.Debug().Msgf("failed to update stripe subscription usage: %s", err.Error())
		}
	}(len(batchRes.Responses))

	newAnnos := make([]*images.ImageAnnotation, 0)
	for i, annotateRes := range batchRes.Responses {
		uri := uris[i]
		newAnnos = append(newAnnos, visionToAnnotation(uri, annotateRes))
	}
	res = append(res, newAnnos...)

	err = cacheAnnotations(newAnnos)
	if err != nil {
		logger.Debug().Msgf("failed to cache with uris: %v", uris)
		return nil, err
	}

	logger.Debug().Msgf("license: %s added %d to request count", licenseID, len(batchRes.Responses))

	return res, nil
}

func fetchStripeSubscription(lic *license.License) (*stripe.Subscription, error) {
	if config.StripeKey == "" {
		return nil, errors.New("STRIPE_KEY env var not found")
	}

	stripe.Key = config.StripeKey

	sub, err := subscription.Get(lic.SubscriptionID, nil)
	if err != nil {
		return nil, err
	}

	fmt.Println("got sub: ", sub.ID)

	return sub, nil
}

func incrementSubscriptionMeter(lic *license.License, quantity int64) error {
	if config.StripeKey == "" {
		return errors.New("STRIPE_KEY env var not found")
	}

	stripe.Key = config.StripeKey

	s, err := fetchStripeSubscription(lic)
	if err != nil {
		return err
	}

	params := &stripe.UsageRecordParams{
		SubscriptionItem: &s.Items.Data[0].ID,
		Action:           stripe.String(string(stripe.UsageRecordActionIncrement)),
		Quantity:         &quantity,
	}

	_, err = usagerecord.New(params)

	return err
}

func visionToAnnotation(uri string, air *pb.AnnotateImageResponse) *images.ImageAnnotation {
	var err sql.NullString

	if air.Error != nil {
		err = sql.NullString{
			String: fmt.Sprintf("Failed to annotate image: %s with error: %s", uri, air.Error),
			Valid:  true,
		}
	} else {
		err = sql.NullString{String: "", Valid: false}
	}

	if air.SafeSearchAnnotation != nil {
		return &images.ImageAnnotation{
			Hash:      utils.Hash(uri),
			URI:       uri,
			DateAdded: time.Now(),
			Adult:     int16(air.SafeSearchAnnotation.Adult),
			Spoof:     int16(air.SafeSearchAnnotation.Spoof),
			Medical:   int16(air.SafeSearchAnnotation.Medical),
			Violence:  int16(air.SafeSearchAnnotation.Violence),
			Racy:      int16(air.SafeSearchAnnotation.Racy),
			Error:     err,
		}

	}

	return &images.ImageAnnotation{
		Hash:      utils.Hash(uri),
		URI:       uri,
		DateAdded: time.Now(),
		Adult:     0,
		Spoof:     0,
		Medical:   0,
		Violence:  0,
		Racy:      0,
		Error:     err,
	}
}

func cacheAnnotation(anno *images.ImageAnnotation) error {
	if err := images.Insert(conn, anno); err != nil {
		return err
	} else {
		logger.Debug().Msgf("Adding %s to DB cache", anno.URI)
		return nil
	}
}

func cacheAnnotations(annos []*images.ImageAnnotation) error {
	if err := images.InsertAll(conn, annos); err != nil {
		return err
	}

	for _, anno := range annos {
		logger.Debug().Msgf("Adding %s to DB cache", anno.URI)
	}

	return nil
}

func fetchAndReadFile(uri string) (string, string, error) {
	path, err := utils.Download(uri)

	// If the download fails, log the error and skip to the next download.
	if err != nil {
		return "", "", err
	}
	f, err := os.Open(path)
	if err != nil {
		return "", "", err
	}
	defer f.Close()

	r := bufio.NewReader(f)
	content, err := ioutil.ReadAll(r)
	if err != nil {
		return "", "", err
	}

	return path, utils.Hash(base64.StdEncoding.EncodeToString(content)), nil
}
