package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"

	"github.com/go-pg/pg/v10"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Server listens on localhost:8080 by default.
var listenAddr string = ""

// Store the db connection passed from main.go.
var conn *pg.DB

type PGStore struct {
	db *pg.DB
}

func NewPGStore(db *pg.DB) *PGStore {
	return &PGStore{db: db}
}

func (store *PGStore) GetLicenseByID(id string) (*License, error) {
	license := new(License)
	err := store.db.Model(license).Where("id = ?", id).Select()
	if err != nil {
		if err == pg.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return license, nil
}

func (store *PGStore) GetLicenseByStripeID(stripeID string) (*License, error) {
	license := new(License)
	err := store.db.Model(license).Where("stripe_id = ?", stripeID).Select()
	if err != nil {
		if err == pg.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return license, nil
}

func (store *PGStore) UpdateLicense(license *License) error {
	_, err := store.db.Model(license).Update(license)
	return err
}

func (store *PGStore) GetLicenseByEmail(email string) (*License, error) {
	license := new(License)
	err := store.db.Model(license).Where("email = ?", email).Select()
	if err != nil {
		if err == pg.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return license, nil
}

var pgStore *PGStore

var logger zerolog.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr}).With().Caller().Logger()

// AnnotateReq is the form of an incoming JSON payload
// for retrieving pass/fail status of each supplied image URI.
type AnnotateReq struct {
	ImgURIList []string `json:"imgURIList"`
}

// ErrorRes is a JSON response containing an error message from the API.
type ErrorRes struct {
	Message string `json:"message"`
}

// Server defines the actions of a Purity API Web Server.
type Server interface {
	Init(int, *sql.DB)
}

// Serve is an instance of a Purity API Web Server.
type Serve struct {
}

// NewServe returns an uninitialized Serve instance.
func NewServe() *Serve {
	return &Serve{}
}

func writeError(code int, message string, w http.ResponseWriter) {
	logger.Info().Msg(message)
	w.WriteHeader(code)
	err := ErrorRes{
		Message: message,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(err)
}
