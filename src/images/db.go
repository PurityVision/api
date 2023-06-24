package images

import (
	"fmt"
	"os"

	"github.com/go-pg/pg/v10"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var logger zerolog.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr}).With().Caller().Logger()

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

// DeleteByURI deletes the images with matching URI.
func DeleteByURI(conn *pg.DB, uri string) error {
	img := ImageAnnotation{URI: uri}

	if _, err := conn.Model(&img).Where("uri = ?", uri).Delete(); err != nil {
		return err
	}

	return nil
}
