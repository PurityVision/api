package images

import (
	"database/sql"
	"time"
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
