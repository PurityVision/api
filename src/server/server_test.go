package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBatchImgFilterHandler(t *testing.T) {
	// fr := BatchImgFilterReq{
	// 	ImgURIList: []string{},
	// }

	// res, err := testBatchImgFilterHandler(fr)
	// if err != nil {

	// }
	// if res.Code != 400 || res.Body.String() != "ImgUriList cannot be empty" {
	// 	t.Error("Web server should have returned a 400 because the ImgURIList was empty")
	// }

	fr := BatchImgFilterReq{
		ImgURIList: []string{"https://i.ytimg.com/vi/19VZZpzbh6s/maxresdefault.jpg"},
	}

	res, err := testBatchImgFilterHandler(fr)
	if err != nil {
		t.Error(err)
	}
	if res.Code != 200 {
		t.Error("Web server should have returned a 200")
	}
	var fRes BatchImgFilterRes
	fRes = BatchImgFilterRes{
		ImgFilterResList: []ImgFilterRes{},
	}
	json.Unmarshal(res.Body.Bytes(), &fRes)
	if len(fRes.ImgFilterResList) != 1 || fRes.ImgFilterResList[0].Pass != true {
		t.Error("Handler didn't return the right results")
	}
}

// TODO rename to something more descriptive
func testBatchImgFilterHandler(fr BatchImgFilterReq) (*httptest.ResponseRecorder, error) {
	b, err := json.Marshal(fr)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal request body struct")
	}
	r := bytes.NewReader(b)

	req, err := http.NewRequest("POST", "/filter", r)
	if err != nil {
		return nil, fmt.Errorf("Failed to create test HTTP request")
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(batchImgFilterHandler)

	handler.ServeHTTP(rr, req)

	return rr, nil
}