package src

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
	"testing"
	"time"

	"github.com/rs/zerolog"
)

const testLicenseID = "797e2754-7547-49c2-acfb-fa7b8357ab03"

var serverTestErr error

type TestServe struct{}

type FilterTestExpect struct {
	Code  int
	Error error
	Res   []*ImageAnnotation
}

type FilterTest struct {
	Given  []string
	Expect FilterTestExpect
}

type junkData struct {
	Name  string
	Color int
}

func TestHealthEndpoint(t *testing.T) {
	ctx, err := getTestCtx()
	if err != nil {
		t.Error(err)
	}

	t.Run("returns 200 if there is no POST body", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/health", nil)
		if err != nil {
			t.Error("Failed to create test HTTP request")
		}

		rr := httptest.NewRecorder()

		code, err := handleHealth(ctx, rr, req)
		if err != nil {
			t.Error(err)
		}
		if code != 200 {
			t.Errorf("Health endpoint expected response 200 but got %d", rr.Code)
		}
	})

	t.Run("returns 200 if there is a junk POST body", func(t *testing.T) {
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

		code, err := handleHealth(ctx, rr, req)
		if err != nil {
			t.Error(err)
		}
		if code != 200 {
			t.Errorf("Health endpoint expected response 200 but got %d", rr.Code)
		}
	})
}

func TestFilterEndpoint(t *testing.T) {
	ctx, err := getTestCtx()
	if err != nil {
		t.Fatal(err)
	}
	ctx.logger = zerolog.Logger{}

	t.Cleanup(func() {
		_, err := ctx.db.Model(&ImageAnnotation{}).Where("1=1").Delete()
		if err != nil {
			fmt.Println("error: ", err)
		}
		_, err = ctx.db.Model(&License{}).Where("1=1").Delete()
		if err != nil {
			fmt.Println("error: ", err)
		}
		defer ctx.db.Close()
	})

	t.Run("filters different URI lists", func(t *testing.T) {
		if serverTestErr != nil {
			log.Fatal(serverTestErr)
		}

		license := &License{
			ID:             testLicenseID,
			Email:          "test@email.com",
			StripeID:       "stripe id",
			IsValid:        true,
			SubscriptionID: os.Getenv("STRIPE_TEST_SUB_ID"),
			ValidityReason: "",
		}

		if _, serverTestErr = ctx.db.Model(license).Insert(); serverTestErr != nil {
			t.Error("failed to create test license")
		}

		tests := []FilterTest{
			{
				Given: []string{},
				Expect: FilterTestExpect{
					Code:  400,
					Error: errors.New("ImgUriList cannot be empty"),
					Res:   []*ImageAnnotation{},
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
					Res: []*ImageAnnotation{
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
					Res: []*ImageAnnotation{
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
			rec, code, err := testFilterHandler(ctx, req)

			if test.Expect.Error != nil {
				if err.Error() != test.Expect.Error.Error() {
					t.Error("expected error but didn't get one")
				}
			}

			if test.Expect.Error == nil && err != nil {
				t.Error("didn't expect error but got: ", err.Error())
			}
			if code != test.Expect.Code {
				t.Errorf("expected status %d but got %d", test.Expect.Code, rec.Code)
			}
			var annotations []*ImageAnnotation
			_ = json.Unmarshal(rec.Body.Bytes(), &annotations)

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
	})

}

func testFilterHandler(ctx appContext, fr *AnnotateReq) (*httptest.ResponseRecorder, int, error) {
	b, err := json.Marshal(fr)
	if err != nil {
		return nil, -1, fmt.Errorf("Failed to marshal request body struct")
	}
	r := bytes.NewReader(b)

	req, err := http.NewRequest("POST", "/filter", r)
	req.Header.Add("LicenseID", testLicenseID)
	if err != nil {
		return nil, -1, errors.New("Failed to create test HTTP request")
	}

	rr := httptest.NewRecorder()

	code, err := handleBatchFilter(ctx, rr, req)
	return rr, code, err
}
