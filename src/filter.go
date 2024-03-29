package src

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	pb "google.golang.org/genproto/googleapis/cloud/vision/v1"
)

// return URIs that are not cached in annotations
func getCachedSSAs(ctx appContext, uris []string) ([]*ImageAnnotation, []string, error) {
	var res []*ImageAnnotation
	cachedSSAs, err := FindAnnotationsByURI(ctx.db, uris)
	if err != nil {
		return nil, nil, err
	}

	uncachedURIs := make([]string, 0, len(uris))

	for _, uri := range uris {
		found := false
		for _, cachedSSA := range cachedSSAs {
			if cachedSSA.URI == uri {
				res = append(res, &cachedSSA)
				found = true
				ctx.logger.Debug().Msgf("found cached image: %s", uri)
				break
			}
		}
		if !found {
			uncachedURIs = append(uncachedURIs, uri)
		}
	}

	return res, uncachedURIs, nil
}

func filterImages(ctx appContext, uris []string, licenseID string) ([]*ImageAnnotation, error) {
	res, uris, err := getCachedSSAs(ctx, uris)
	if err != nil {
		return nil, err
	}
	if len(uris) == 0 {
		return res, nil
	}

	license, err := ctx.licenseStore.GetLicenseByID(licenseID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch license: %s", err.Error())
	}
	if license == nil {
		return nil, errors.New("license not found")
	}
	if license.IsTrial {
		remainingUsage := ctx.config.TrialLicenseMaxUsage - license.RequestCount
		if remainingUsage < len(uris) {
			uris = uris[:remainingUsage]
		}
		if remainingUsage <= 0 { // return early if trial license is expired
			license, err = ctx.licenseStore.ExpireTrial(license)
			if err != nil {
				return res, fmt.Errorf("failed to mark trial license as expired: %s", err.Error())
			} else {
				return res, fmt.Errorf("trial license %s has reached max usage and is now invalid", license.ID)
			}
		}
	}

	annotateImageResponses, err := GetURIAnnotations(uris)
	if err != nil {
		return nil, err
	}

	if len(annotateImageResponses) > 0 {
		license.RequestCount += len(annotateImageResponses)
		if err = ctx.licenseStore.UpdateLicense(license); err != nil {
			ctx.logger.Error().Msgf("failed to update license request count: %s", err)
		}
		if err = IncrementSubscriptionMeter(ctx.config.StripeKey, license, int64(len(annotateImageResponses))); err != nil {
			ctx.logger.Error().Msgf("failed to update stripe subscription usage: %s", err.Error())
		}
	}

	var buildSSARes = func(annotations []*pb.AnnotateImageResponse) []*ImageAnnotation {
		var annoRes []*ImageAnnotation
		for i, annotation := range annotations {
			if annotation == nil {
				continue
			}
			uri := uris[i]
			annoRes = append(annoRes, annotationToSafeSearchResponseRes(uri, annotation))
		}
		return annoRes
	}

	safeSearchAnnotationsRes := buildSSARes(annotateImageResponses)
	res = append(res, safeSearchAnnotationsRes...)

	err = cacheAnnotations(ctx, safeSearchAnnotationsRes)
	if err != nil {
		ctx.logger.Error().Msgf("failed to cache with uris: %v", uris)
	}

	ctx.logger.Info().Msgf("license: %s added %d to request count", licenseID, len(annotateImageResponses))

	return res, nil
}

func annotationToSafeSearchResponseRes(uri string, annotation *pb.AnnotateImageResponse) *ImageAnnotation {
	var err sql.NullString
	if annotation.Error != nil {
		err = sql.NullString{String: annotation.Error.Message, Valid: true}
	} else {
		err = sql.NullString{String: "", Valid: false}
	}

	if annotation == nil || annotation.SafeSearchAnnotation == nil {
		return &ImageAnnotation{
			Hash:      Hash(uri),
			URI:       uri,
			Error:     err,
			DateAdded: time.Now(),
			Adult:     0,
			Spoof:     0,
			Medical:   0,
			Violence:  0,
			Racy:      0,
		}
	}

	return &ImageAnnotation{
		Hash:      Hash(uri),
		URI:       uri,
		Error:     err,
		DateAdded: time.Now(),
		Adult:     int16(annotation.SafeSearchAnnotation.Adult),
		Spoof:     int16(annotation.SafeSearchAnnotation.Spoof),
		Medical:   int16(annotation.SafeSearchAnnotation.Medical),
		Violence:  int16(annotation.SafeSearchAnnotation.Violence),
		Racy:      int16(annotation.SafeSearchAnnotation.Racy),
	}
}

func cacheAnnotations(ctx appContext, annos []*ImageAnnotation) error {
	if err := InsertAll(ctx.db, annos); err != nil {
		return err
	}

	for _, anno := range annos {
		ctx.logger.Info().Msgf("adding %s to DB cache", anno.URI)
	}

	return nil
}

// func cacheAnnotation(anno *images.ImageAnnotation) error {
// 	if err := images.Insert(conn, anno); err != nil {
// 		return err
// 	} else {
// 		logger.Debug().Msgf("Adding %s to DB cache", anno.URI)
// 		return nil
// 	}
// }

// func fetchAndReadFile(uri string) (string, string, error) {
// 	path, err := utils.Download(uri)

// 	// If the download fails, log the error and skip to the next download.
// 	if err != nil {
// 		return "", "", err
// 	}
// 	f, err := os.Open(path)
// 	if err != nil {
// 		return "", "", err
// 	}
// 	defer f.Close()

// 	r := bufio.NewReader(f)
// 	content, err := ioutil.ReadAll(r)
// 	if err != nil {
// 		return "", "", err
// 	}

// 	return path, utils.Hash(base64.StdEncoding.EncodeToString(content)), nil
// }
