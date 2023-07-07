package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"purity-vision-filter/src/config"
	"purity-vision-filter/src/db"
	"purity-vision-filter/src/images"
	"testing"

	"github.com/go-pg/pg/v10"
)

type TestServe struct {
}

func (s *TestServe) Init(_conn *pg.DB) {
	conn = _conn
}

func TestHealthHandler(t *testing.T) {
	testHealthNoBody(t)
	testHealthJunkBody(t)
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

func TestFilterEmpty(t *testing.T) {
	conn, err := db.Init(config.DefaultDBTestName)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	s := TestServe{}
	s.Init(conn)
	uri := "https://i.ytimg.com/vi/19VZZpzbh6s/maxresdefault.jpg"

	req := AnnotateReq{
		ImgURIList: []string{},
	}

	var errRes ErrorRes
	res, err := testBatchImgFilterHandler(req)
	if err != nil {
		t.Error("Shouldn't have thrown an error")
	}

	decoder := json.NewDecoder(res.Body)
	if err := decoder.Decode(&errRes); err != nil {
		t.Error("JSON body missing or malformed")
	}

	if res.Code != 400 || errRes.Message != "ImgUriList cannot be empty" {
		t.Error("Web server should have returned a 400 because the ImgURIList was empty")
	}

	// Delete the img from the DB.
	if err = images.DeleteByURI(conn, uri); err != nil {
		t.Log(err)
	}
}

func TestFilter(t *testing.T) {

	conn, err := db.Init(config.DefaultDBTestName)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	s := TestServe{}
	s.Init(conn)

	req := AnnotateReq{
		ImgURIList: []string{
			"https://i.ytimg.com/vi/19VZZpzbh6s/maxresdefault.jpg",
			"https://i.redd.it/23tzr9qimgf51.png",
			"https://www.pandesiaworld.com/wp-content/uploads/2019/11/Adina-Barbu-8.jpg",
			"https://www.captainmitchs.com/wp-content/uploads/2018/01/wood-duck-PFYHVZN.jpg",
			"https://titis.org/uploads/posts/2022-12/thumbs/1671427956_titis-org-p-nude-girls-forest-chastnaya-erotika-6.jpg",
			"https://i.imgur.com/5JdV3Uo.jpg",
		},
	}

	res, err := testBatchImgFilterHandler(req)
	if err != nil {
		t.Error(err)
	}
	if res.Code != 200 {
		t.Error("Web server should have returned a 200")
	}
	var annotation []*images.ImageAnnotation
	json.Unmarshal(res.Body.Bytes(), &annotation)
	if len(annotation) != len(req.ImgURIList) {
		t.Error("Handler didn't return the right results")
	}

	// Cleanup DB
	for _, uri := range req.ImgURIList {
		if err = images.DeleteByURI(conn, uri); err != nil {
			t.Log(err)
		}
	}
}

func TestFilterDuplicates(t *testing.T) {
	conn, err := db.Init(config.DefaultDBTestName)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	s := TestServe{}
	s.Init(conn)
	uri := "https://i.ytimg.com/vi/19VZZpzbh6s/maxresdefault.jpg"

	req := AnnotateReq{
		ImgURIList: []string{
			"https://i.ytimg.com/vi/19VZZpzbh6s/maxresdefault.jpg",
			"https://i.ytimg.com/vi/19VZZpzbh6s/maxresdefault.jpg",
			"https://i.ytimg.com/vi/19VZZpzbh6s/maxresdefault.jpg",
			"https://i.ytimg.com/vi/19VZZpzbh6s/maxresdefault.jpg",
		},
	}

	//var errRes ErrorRes
	res, err := testBatchImgFilterHandler(req)
	if err != nil {
		t.Error("Shouldn't have thrown an error")
	}

	if res.Code != 200 {
		t.Error("Web server should have returned a 200")
	}
	var annotation []*images.ImageAnnotation
	json.Unmarshal(res.Body.Bytes(), &annotation)
	if len(annotation) != 1 {
		t.Error("Handler didn't return the right results")
	}

	// Delete the img from the DB.
	if err = images.DeleteByURI(conn, uri); err != nil {
		t.Log(err)
	}
}

func testBatchImgFilterHandler(fr AnnotateReq) (*httptest.ResponseRecorder, error) {
	b, err := json.Marshal(fr)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal request body struct")
	}
	r := bytes.NewReader(b)

	req, err := http.NewRequest("POST", "/filter", r)
	if err != nil {
		return nil, errors.New("Failed to create test HTTP request")
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleBatchFilter(logger))

	handler.ServeHTTP(rr, req)

	return rr, nil
}
