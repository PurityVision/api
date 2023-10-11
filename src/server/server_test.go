package server

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"purity-vision-filter/src/config"
	"purity-vision-filter/src/db"
	"purity-vision-filter/src/images"
	lic "purity-vision-filter/src/license"
	"testing"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/joho/godotenv"
)

type TestServe struct {
}

func (s *TestServe) Init(_conn *pg.DB) {
	conn = _conn
}

type FilterTestExpect struct {
	Code  int
	Error error
	Res   []*images.ImageAnnotation
}

type FilterTest struct {
	Given  []string
	Expect FilterTestExpect
}

func testHealthNoBody(t *testing.T) {
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Error("Failed to create test HTTP request")
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(health)

	handler.ServeHTTP(rr, req)

	if rr.Code != 200 {
		t.Errorf("Health endpoint expected response 200 but got %d", rr.Code)
	}
}

type junkData struct {
	Name  string
	Color int
}

// The health endpoint given junk POST data should still simply return a 200 code.
func testHealthJunkBody(t *testing.T) {
	someData := junkData{
		Name:  "pil",
		Color: 221,
	}
	b, err := json.Marshal(someData)
	if err != nil {
		t.Error("Failed to marshal request body struct")
	}
	r := bytes.NewReader(b)
	req, err := http.NewRequest("POST", "/health", r)
	if err != nil {
		t.Error("Failed to create test HTTP request")
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(health)

	handler.ServeHTTP(rr, req)

	if rr.Code != 200 {
		t.Errorf("Health endpoint expected response 200 but got %d", rr.Code)
	}
}

func TestHealthHandler(t *testing.T) {
	testHealthNoBody(t)
	testHealthJunkBody(t)
}

func testCleanup() {
	conn.Model(&images.ImageAnnotation{}).Where("1=1").Delete()
	_, err := conn.Model(&lic.License{}).Where("1=1").Delete()
	if err != nil {
		fmt.Println("error: ", err)
	}
	defer conn.Close()
}

const testLicenseID = "797e2754-7547-49c2-acfb-fa7b8357ab03"

var err error

func TestMain(m *testing.M) {
	if err := godotenv.Load("../../.env"); err != nil {
		log.Fatal(err)
	}
	config.Init()

	conn, err = db.Init(config.DefaultDBTestName)
	if err != nil {
		log.Fatal(err)
	}

	exitCode := m.Run()

	os.Exit(exitCode)
}

func TestFilterHandlerTable(t *testing.T) {
	t.Cleanup(testCleanup)

	pgStore = NewPGStore(conn)
	if err != nil {
		log.Fatal(err)
	}

	license := &lic.License{
		ID:       testLicenseID,
		Email:    "test@email.com",
		StripeID: "stripe id",
		IsValid:  true,
	}

	if _, err = conn.Model(license).Insert(); err != nil {
		t.Error("failed to create test license")
	}

	s := TestServe{}
	s.Init(conn)

	tests := []FilterTest{
		{
			Given: []string{},
			Expect: FilterTestExpect{
				Code:  400,
				Error: errors.New("ImgUriList cannot be empty"),
				Res:   []*images.ImageAnnotation{},
			},
		},
		{
			Given: []string{
				"https://i.imgur.com/FEpwOY8.jpg",
				"https://i.imgur.com/FEpwOY8.jpg",
				"https://i.imgur.com/FEpwOY8.jpg",
				"https://i.imgur.com/FEpwOY8.jpg",
				"https://i.imgur.com/FEpwOY8.jpg",
			},
			Expect: FilterTestExpect{
				Code:  200,
				Error: nil,
				Res: []*images.ImageAnnotation{
					{
						Hash:      "87408bebb6a1d42cd7cc1bbffb6d7dcc6aff14af4aea5c9af9fc5b624cf7c93a",
						URI:       "https://i.imgur.com/FEpwOY8.jpg",
						Error:     sql.NullString{},
						DateAdded: time.Now(),
						Adult:     2,
						Spoof:     1,
						Medical:   2,
						Violence:  3,
						Racy:      5,
					},
				},
			},
		},
		{
			Given: []string{
				"https://i.imgur.com/FEpwOY8.jpg",
				"https://i.imgur.com/6ZOubbU.png",
				"https://i.imgur.com/qtTfzH6.jpg",
				"https://i.imgur.com/RwHI4jk.jpg",
			},
			Expect: FilterTestExpect{
				Code:  200,
				Error: nil,
				Res: []*images.ImageAnnotation{
					{
						Hash:      "87408bebb6a1d42cd7cc1bbffb6d7dcc6aff14af4aea5c9af9fc5b624cf7c93a",
						URI:       "https://i.imgur.com/FEpwOY8.jpg",
						Error:     sql.NullString{},
						DateAdded: time.Now(),
						Adult:     2,
						Spoof:     1,
						Medical:   2,
						Violence:  3,
						Racy:      5,
					},
					{
						Hash:      "65d2ad788998a350e7476c4a110ece346d4d56ab76670d48ddd896444a0029b1",
						URI:       "https://i.imgur.com/6ZOubbU.png",
						Error:     sql.NullString{},
						DateAdded: time.Now(),
						Adult:     1,
						Spoof:     1,
						Medical:   2,
						Violence:  1,
						Racy:      5,
					},
					{
						Hash:      "2a5cdbc5148669ec4efc788d03f535cf99f13756ccd200ae48faf59fac30b811",
						URI:       "https://i.imgur.com/qtTfzH6.jpg",
						Error:     sql.NullString{},
						DateAdded: time.Now(),
						Adult:     2,
						Spoof:     3,
						Medical:   2,
						Violence:  4,
						Racy:      2,
					},
					{
						Hash:      "b2047dfb0412f815859b269288a948528587b77d9b3e0395cd57faf2ba4c37f5",
						URI:       "https://i.imgur.com/RwHI4jk.jpg",
						Error:     sql.NullString{},
						DateAdded: time.Now(),
						Adult:     5,
						Spoof:     1,
						Medical:   3,
						Violence:  3,
						Racy:      5,
					},
				},
			},
		},
	}

	for _, test := range tests {
		req := &AnnotateReq{ImgURIList: test.Given}
		rec, err := testFilterHandler(req)

		if test.Expect.Error != nil {
			decoder := json.NewDecoder(rec.Body)
			var errRes ErrorRes
			if err := decoder.Decode(&errRes); err != nil {
				t.Error("JSON body missing or malformed")
			}
			if errRes.Message != test.Expect.Error.Error() {
				t.Error("expected error but didn't get one")
			}
		}

		if test.Expect.Error == nil && err != nil {
			t.Error("didn't expect error but got: ", err.Error())
		}
		if rec.Code != test.Expect.Code {
			t.Errorf("expected status %d but got %d", rec.Code, test.Expect.Code)
		}
		var annotations []*images.ImageAnnotation
		json.Unmarshal(rec.Body.Bytes(), &annotations)

		if len(annotations) != len(test.Expect.Res) {
			t.Errorf("expected %d annotation results but got %d", len(test.Expect.Res), len(annotations))
		}

		for i, annotation := range annotations {
			expected := test.Expect.Res[i]
			if annotation.Adult != expected.Adult {
				t.Errorf("expected adult to be %d but got %d", annotation.Adult, expected.Adult)
			}

			if annotation.Spoof != expected.Spoof {
				t.Errorf("expected spoof to be %d but got %d", annotation.Spoof, expected.Spoof)
			}

			if annotation.Medical != expected.Medical {
				t.Errorf("expected medical to be %d but got %d", annotation.Medical, expected.Medical)
			}

			if annotation.Violence != expected.Violence {
				t.Errorf("expected violence to be %d but got %d", annotation.Violence, expected.Violence)
			}

			if annotation.Racy != expected.Racy {
				t.Errorf("expected racy to be %d but got %d", annotation.Racy, expected.Racy)
			}
		}
	}
}

func testFilterHandler(fr *AnnotateReq) (*httptest.ResponseRecorder, error) {
	b, err := json.Marshal(fr)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal request body struct")
	}
	r := bytes.NewReader(b)

	req, err := http.NewRequest("POST", "/filter", r)
	req.Header.Add("LicenseID", testLicenseID)
	if err != nil {
		return nil, errors.New("Failed to create test HTTP request")
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleBatchFilter(logger))

	handler.ServeHTTP(rr, req)

	return rr, nil
}
