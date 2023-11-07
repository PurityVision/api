package src

import (
	"fmt"
	"net/http"

	"github.com/go-pg/pg/v10"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

var PrintSomethingWrong = func(w http.ResponseWriter) { fmt.Fprint(w, "Something went wrong") }

// Init intializes the Serve instance and exposes it based on the port parameter.
func (s *Serve) InitServer(port int, _conn *pg.DB) {
	// Store the database connection in a global var.
	conn = _conn
	licenseStore = NewLicenseStore(conn)

	r := mux.NewRouter()

	r.Use(addCorsHeaders)
	r.Handle("/", http.FileServer(http.Dir("./"))).Methods("GET")
	r.HandleFunc("/health", health).Methods("GET", "OPTIONS")
	r.HandleFunc("/license/{id}", handleGetLicense).Methods("GET")
	r.HandleFunc("/webhook", handleWebhook).Methods("POST")
	r.HandleFunc("/trial-register", handleTrialRegister).Methods("POST", "OPTIONS")

	// Paywalled filter routes.
	filterR := r.PathPrefix("/filter").Subrouter()
	filterR.Use(paywallMiddleware(licenseStore))
	filterR.HandleFunc("/batch", handleBatchFilter(logger)).Methods("POST", "OPTIONS")

	listenAddr = fmt.Sprintf("%s:%d", listenAddr, port)
	log.Info().Msgf("Web server now listening on %s", listenAddr)
	log.Fatal().Msg(http.ListenAndServe(listenAddr, r).Error())
}
