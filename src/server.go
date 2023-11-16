package src

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/go-pg/pg/v10"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// AnnotateReq is the form of an incoming JSON payload
// for retrieving pass/fail status of each supplied image URI.
type AnnotateReq struct {
	ImgURIList []string `json:"imgURIList"`
}

type AnnotationStore interface {
	GetAnnotations([]string) ([]*ImageAnnotation, error)
	PutAnnotations([]*ImageAnnotation) error
}

// Serve is an instance of a Purity API Web Server.
type appContext struct {
	db              pg.DB
	logger          zerolog.Logger
	licenseStore    LicenseStorer
	annotationStore AnnotationStore
	config          Config
}

type appHandler struct {
	appContext
	H func(appContext, http.ResponseWriter, *http.Request) (int, error)
}

// Our ServeHTTP method is mostly the same, and also has the ability to
// access our *appContext's fields (templates, loggers, etc.) as well.
func (ah appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Updated to pass ah.appContext as a parameter to our handler type.
	status, err := ah.H(ah.appContext, w, r)
	if err != nil {
		ah.appContext.logger.Printf("HTTP %d: %q", status, err)
		switch status {
		case http.StatusNotFound:
			http.NotFound(w, r)
			// And if we wanted a friendlier error page, we can
			// now leverage our context instance - e.g.
			// err := ah.renderTemplate(w, "http_404.tmpl", nil)
		case http.StatusInternalServerError:
			http.Error(w, http.StatusText(status), status)
		default:
			http.Error(w, http.StatusText(status), status)
		}
	}
}

// InitServer intializes an HTTP server and registers listeners.
func InitServer() {
	var portFlag int

	config, err := newConfig()
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	zerolog.SetGlobalLevel(zerolog.Level(zerolog.ErrorLevel))

	conn, err := InitDB(config)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	defer conn.Close()

	ctx := appContext{
		db:              *conn,
		logger:          zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, NoColor: true}).With().Timestamp().Logger(),
		licenseStore:    NewLicenseStore(conn),
		annotationStore: nil,
		config:          config,
	}

	flag.IntVar(&portFlag, "port", 8080, "port to run the service on")
	flag.Parse()

	logLevel, err := strconv.Atoi(ctx.config.LogLevel)
	if err != nil {
		panic(err)
	}
	zerolog.SetGlobalLevel(zerolog.Level(logLevel))

	r := mux.NewRouter()

	r.Use(addCorsHeaders)
	r.Handle("/", http.FileServer(http.Dir("./"))).Methods("GET")
	r.Handle("/health", &appHandler{ctx, handleHealth}).Methods("GET", "OPTIONS")
	r.Handle("/license/{id}", &appHandler{ctx, handleGetLicense}).Methods("GET", "OPTIONS")
	r.Handle("/webhook", &appHandler{ctx, handleWebhook}).Methods("POST")
	// r.HandleFunc("/trial-register", handleTrialRegister).Methods("POST", "OPTIONS")

	// Paywalled filter routes.
	filterR := r.PathPrefix("/filter").Subrouter()
	filterR.Use(paywallMiddleware(ctx))
	filterR.Handle("/batch", &appHandler{ctx, handleBatchFilter}).Methods("POST", "OPTIONS")

	listenAddr := ""
	listenAddr = fmt.Sprintf("%s:%d", listenAddr, portFlag)
	ctx.logger.Info().Msgf("Web server now listening on %s", listenAddr)
	ctx.logger.Fatal().Msg(http.ListenAndServe(listenAddr, r).Error())
}
