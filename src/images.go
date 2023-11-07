package src

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/rs/zerolog/log"
)

type ImageAnnotation struct {
	Hash      string         `json:"hash"`  // Sha254 hash of the base64 encoded contents of the image.
	URI       string         `json:"uri"`   // The original URI where the image resides on the web.
	Error     sql.NullString `json:"error"` // Any error returned when trying to filter the image.
	DateAdded time.Time      `json:"dateAdded"`

	// from SafeSearchAnnotation fields
	Adult    int16 `json:"adult"`
	Spoof    int16 `json:"spoof"`
	Medical  int16 `json:"medical"`
	Violence int16 `json:"violence"`
	Racy     int16 `json:"racy"`
}

// FindByURI returns an image with the matching URI.
func FindByURI(conn *pg.DB, imgURI string) (*ImageAnnotation, error) {
	var img ImageAnnotation

	err := conn.Model(&img).Where("uri = ?", imgURI).Select()
	if err != nil {
		log.Error().Msgf("err: %v", err)
		return nil, nil
	}

	return &img, nil
}

// FindAnnotationsByURI returns annotations that have matching URI's.
func FindAnnotationsByURI(conn *pg.DB, uris []string) ([]ImageAnnotation, error) {
	var annotations []ImageAnnotation

	if len(uris) == 0 {
		return nil, fmt.Errorf("imgURIList cannot be empty")
	}

	conn.Model(&annotations).Where("uri IN (?)", pg.In(uris)).Select()

	return annotations, nil
}

// Insert inserts the annotation into the DB.
func Insert(conn *pg.DB, image *ImageAnnotation) error {
	_, err := conn.Model(image).Insert()
	if err != nil {
		return err
	}
	logger.Debug().Msgf("inserted image: %s", image.URI)

	return nil
}

func InsertAll(conn *pg.DB, images []*ImageAnnotation) error {
	if len(images) == 0 {
		return nil
	}

	_, err := conn.Model(&images).Insert()
	if err != nil {
		return err
	}
	logger.Debug().Msgf("inserted %d images", len(images))

	return nil
}

// DeleteByURI deletes the images with matching URI.
func DeleteByURI(conn *pg.DB, uri string) error {
	img := ImageAnnotation{URI: uri}

	if _, err := conn.Model(&img).Where("uri = ?", uri).Delete(); err != nil {
		return err
	}

	return nil
}
