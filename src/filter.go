package src

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	pb "google.golang.org/genproto/googleapis/cloud/vision/v1"
)

func filterImages(uris []string, licenseID string) ([]*ImageAnnotation, error) {
	var res []*ImageAnnotation
	license, err := licenseStore.GetLicenseByID(licenseID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch license: %s", err.Error())
	}
	if license == nil {
		return nil, errors.New("license not found")
	}

	cachedSSAs, err := FindAnnotationsByURI(conn, uris)
	if err != nil {
		return nil, err
	}

	noncachedURIs := make([]string, 0)

	for _, uri := range uris {
		found := false
		for _, cachedSSA := range cachedSSAs {
			if cachedSSA.URI == uri {
				res = append(res, &cachedSSA)
				found = true
				logger.Debug().Msgf("Found cached image: %s", uri)
				break
			}
		}
		if !found {
			noncachedURIs = append(noncachedURIs, uri)
		}
	}

	uris = noncachedURIs

	safeSearchAnnotations, errs, err := GetImgSSAs(uris, license, licenseStore)
	if err != nil {
		return nil, err
	}

	imageAnnotations := make([]*ImageAnnotation, 0)
	for i, safeSearchAnnotation := range safeSearchAnnotations {
		if safeSearchAnnotation == nil {
			continue
		}
		uri := uris[i]
		imageAnnotations = append(imageAnnotations, visionToAnnotation(uri, safeSearchAnnotation, errs[i]))
	}
	res = append(res, imageAnnotations...)

	err = cacheAnnotations(imageAnnotations)
	if err != nil {
		logger.Debug().Msgf("failed to cache with uris: %v", uris)
		return nil, err
	}

	logger.Debug().Msgf("license: %s added %d to request count", licenseID, len(safeSearchAnnotations))

	return res, nil
}

func visionToAnnotation(uri string, safeSearchAnno *pb.SafeSearchAnnotation, annoErr error) *ImageAnnotation {
	var err sql.NullString
	if annoErr != nil {
		err = sql.NullString{String: annoErr.Error(), Valid: true}
	} else {
		err = sql.NullString{String: "", Valid: false}
	}

	if safeSearchAnno != nil {
		return &ImageAnnotation{
			Hash:      Hash(uri),
			URI:       uri,
			Error:     err,
			DateAdded: time.Now(),
			Adult:     int16(safeSearchAnno.Adult),
			Spoof:     int16(safeSearchAnno.Spoof),
			Medical:   int16(safeSearchAnno.Medical),
			Violence:  int16(safeSearchAnno.Violence),
			Racy:      int16(safeSearchAnno.Racy),
		}
	} else {
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
}

func cacheAnnotations(annos []*ImageAnnotation) error {
	if err := InsertAll(conn, annos); err != nil {
		return err
	}

	for _, anno := range annos {
		logger.Debug().Msgf("Adding %s to DB cache", anno.URI)
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
