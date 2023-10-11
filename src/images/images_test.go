package images

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"purity-vision-filter/src/config"
	"purity-vision-filter/src/db"
	"purity-vision-filter/src/utils"
	"testing"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/joho/godotenv"
)

var conn *pg.DB
var tx *pg.Tx
var err error
var imgURIList = []string{
	"https://hatrabbits.com/wp-content/uploads/2017/01/random.jpg",
	"https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcT1ZgCJADylizZLNnOnyuhtwR2qVk5yOi0UoQ&usqp=CAU",
	"https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcRKsJoGKlOJnxl-GNgfUtluGobgx_M8JBdsng&usqp=CAU",
}

func TestMain(m *testing.M) {
	if err := godotenv.Load("../../.env"); err != nil {
		log.Fatal(err)
	}
	config.Init()

	fmt.Println("Got value: ", os.Getenv("PURITY_DB_HOST"))
	conn, err = db.Init(config.DefaultDBTestName)
	if err != nil {
		log.Fatal(err)
	}

	exitCode := m.Run()

	os.Exit(exitCode)
}

func TestInsertImage(t *testing.T) {
	for _, uri := range imgURIList {
		fakeHash := utils.Hash(uri)
		anno := &ImageAnnotation{
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
		err = Insert(conn, anno)
		if err != nil {
			t.Fatal(err.Error())
		}
	}
}

func TestFindImagesByURI(t *testing.T) {
	smallURIList := imgURIList[:1]

	imgList, err := FindAnnotationsByURI(conn, smallURIList)
	if err != nil {
		t.Fatal(err.Error())
	}

	if len(imgList) != 1 {
		t.Fatalf("Expected 1 image in response but received %d", len(imgList))
		t.FailNow()
	}

	smallURIList = []string{}
	imgList, err = FindAnnotationsByURI(conn, smallURIList)
	if err == nil {
		t.Fatal("Expected FindImagesByURI to return an error because imgURIList cannot be empty")
	}
}

func TestDeleteImagesByURI(t *testing.T) {
	for _, uri := range imgURIList {
		err = DeleteByURI(conn, uri)
		if err != nil {
			t.Fatal(err)
		}
	}
}
