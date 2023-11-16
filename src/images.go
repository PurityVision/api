package src

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/go-pg/pg/v10"
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
func FindByURI(conn pg.DB, imgURI string) (ImageAnnotation, error) {
	var img ImageAnnotation

	err := conn.Model(&img).Where("uri = ?", imgURI).Select()
	if err != nil {
		return img, err
	}

	return img, nil
}

// FindAnnotationsByURI returns annotations that have matching URI's.
func FindAnnotationsByURI(conn pg.DB, uris []string) ([]ImageAnnotation, error) {
	var annotations []ImageAnnotation

	if len(uris) == 0 {
		return nil, fmt.Errorf("imgURIList cannot be empty")
	}

	if err := conn.Model(&annotations).Where("uri IN (?)", pg.In(uris)).Select(); err != nil {
		return nil, err
	}

	return annotations, nil
}

// Insert inserts the annotation into the DB.
func Insert(conn pg.DB, image ImageAnnotation) error {
	_, err := conn.Model(&image).Insert()
	if err != nil {
		return err
	}

	return nil
}

// InsertAll inserts all the image safe search annotations into the DB.
func InsertAll(conn pg.DB, images []*ImageAnnotation) error {
	if len(images) == 0 {
		return nil
	}

	_, err := conn.Model(&images).Insert()
	if err != nil {
		return err
	}

	return nil
}

// DeleteByURI deletes the images with matching URI.
func DeleteByURI(conn pg.DB, uri string) error {
	img := ImageAnnotation{URI: uri}

	if _, err := conn.Model(&img).Where("uri = ?", uri).Delete(); err != nil {
		return err
	}

	return nil
}
