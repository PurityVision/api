package src

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
)

// var conn *pg.DB
func getTestCtx() (appContext, error) {
	var ctx appContext
	config, err := newConfig()
	if err != nil {
		return ctx, err
	}
	config.DBName = "purity_test"
	conn, err := InitDB(config)
	if err != nil {
		return ctx, err
	}
	ctx.db = *conn
	ctx.logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
	ctx.licenseStore = NewLicenseStore(conn)
	ctx.annotationStore = nil
	ctx.config = config
	return ctx, nil
}

func TestMain(m *testing.M) {
	godotenv.Load()
}

func TestImages(t *testing.T) {
	ctx, err := getTestCtx()
	if err != nil {
		t.Fatal(err)
	}

	var imgURIList = []string{
		"https://hatrabbits.com/wp-content/uploads/2017/01/random.jpg",
		"https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcT1ZgCJADylizZLNnOnyuhtwR2qVk5yOi0UoQ&usqp=CAU",
		"https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcRKsJoGKlOJnxl-GNgfUtluGobgx_M8JBdsng&usqp=CAU",
	}

	t.Run("inserts images", func(t *testing.T) {
		for _, uri := range imgURIList {
			fakeHash := Hash(uri)
			anno := ImageAnnotation{
				Hash:      fakeHash,
				URI:       uri,
				Error:     sql.NullString{},
				DateAdded: time.Now(),
				Adult:     0,
				Spoof:     0,
				Medical:   0,
				Violence:  0,
				Racy:      0,
			}
			err := Insert(ctx.db, anno)
			if err != nil {
				t.Fatal(err.Error())
			}
		}
	})

	t.Run("finds images by URI", func(t *testing.T) {
		smallURIList := imgURIList[:1]

		imgList, err := FindAnnotationsByURI(ctx.db, smallURIList)
		if err != nil {
			t.Fatal(err.Error())
		}

		if len(imgList) != 1 {
			t.Fatalf("Expected 1 image in response but received %d", len(imgList))
			t.FailNow()
		}

		smallURIList = []string{}
		_, err = FindAnnotationsByURI(ctx.db, smallURIList)
		if err == nil {
			t.Fatal("Expected FindImagesByURI to return an error because imgURIList cannot be empty")
		}
	})

	t.Run("deletes images by URI", func(t *testing.T) {
		for _, uri := range imgURIList {
			err := DeleteByURI(ctx.db, uri)
			if err != nil {
				t.Fatal(err)
			}
		}
	})

}
