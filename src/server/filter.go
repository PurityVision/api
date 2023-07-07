package server

import (
	"bufio"
	"database/sql"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"purity-vision-filter/src/images"
	"purity-vision-filter/src/utils"
	"purity-vision-filter/src/vision"
	"time"

	pb "google.golang.org/genproto/googleapis/cloud/vision/v1"
)

func filterImages(uris []string) ([]*images.ImageAnnotation, error) {
	var res []*images.ImageAnnotation

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

	for i, annotateRes := range batchRes.Responses {
		uri := uris[i]
		annotation := visionToAnnotation(uri, annotateRes)
		res = append(res, annotation)
		err = cacheAnnotation(annotation)
		if err != nil {
			logger.Debug().Msgf("Failed to cache with uris: %v", uris)
			return nil, err
		}
	}

	return res, nil
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
