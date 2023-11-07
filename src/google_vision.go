package src

import (
	"context"

	vision "cloud.google.com/go/vision/apiv1"
	"github.com/rs/zerolog/log"
	pb "google.golang.org/genproto/googleapis/cloud/vision/v1"
)

type BatchAnnotateResponse map[string]*pb.AnnotateImageResponse

// GetImgSSas returns the SafeSearchAnnotations and any associated errors given uris, and an optional application error.
func GetImgSSAs(uris []string, license *License, store licenseStoreInterface) ([]*pb.SafeSearchAnnotation, []error, error) {
	ctx := context.Background()
	client, err := vision.NewImageAnnotatorClient(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer client.Close()

	res := make([]*pb.SafeSearchAnnotation, 0)
	errs := make([]error, 0)
	requestCount := 0
	remainingUsage := 0
	if license.IsTrial {
		remainingUsage = TrialLicenseMaxUsage - license.RequestCount
	}

	for _, uri := range uris {
		var ssa *pb.SafeSearchAnnotation

		if license.IsTrial && remainingUsage <= 0 {
			log.Logger.Debug().Msgf("skipping %s because trial license is expired", uri)
			if license.ValidityReason == "" {
				log.Logger.Debug().Msgf("trial license %s has reached max usage", license.ID)
				license.ValidityReason = "trial license has expired"
				if err = store.UpdateLicense(license); err != nil {
					log.Logger.Debug().Msgf("failed to mark trial license as expired: %s", err.Error())
				}
			}
			ssa = nil
			errs = append(errs, nil)
		} else {
			ssa, err = client.DetectSafeSearch(ctx, vision.NewImageFromURI(uri), nil)
			remainingUsage-- // only applicable for trial licenses
			requestCount++
			if err != nil {
				log.Logger.Error().Msgf("failed to safe search detect image: %s, err: %s", uris, err.Error())
				errs = append(errs, err)
			} else {
				errs = append(errs, nil)
			}
		}

		res = append(res, ssa)
	}

	if requestCount > 0 {
		license.RequestCount += requestCount
		if err = store.UpdateLicense(license); err != nil {
			log.Logger.Debug().Msgf("failed to update license request count: %s", err.Error())
		}

		if err = IncrementSubscriptionMeter(license, int64(requestCount)); err != nil {
			log.Logger.Debug().Msgf("failed to update stripe subscription usage: %s", err.Error())
		}

	}

	return res, errs, nil
}

// GetImgAnnotation annotates an image.
func GetImgAnnotation(uri string) (*pb.AnnotateImageResponse, error) {
	ctx := context.Background()

	client, err := vision.NewImageAnnotatorClient(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	// f, err := os.Open(filePath)
	// if err != nil {
	// 	return nil, err
	// }
	// defer f.Close()

	// image, err := vision.NewImageFromReader(f)
	// if err != nil {
	// 	return nil, err
	// }

	image := vision.NewImageFromURI(uri)

	req := &pb.AnnotateImageRequest{
		Image: image,
		Features: []*pb.Feature{
			{Type: pb.Feature_SAFE_SEARCH_DETECTION, MaxResults: 5},
		},
	}

	res, err := client.AnnotateImage(ctx, req)
	if err != nil {
		return nil, err
	}

	return res, nil
}
