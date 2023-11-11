package main

import (
	"flag"
	"os"
	"purity-vision-filter/src"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var portFlag int

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal().Err(err)
	}

	if err := src.InitConfig(); err != nil {
		log.Fatal().Msg(err.Error())
	}

	flag.IntVar(&portFlag, "port", src.DefaultPort, "port to run the service on")
	flag.Parse()

	logLevel, err := strconv.Atoi(src.LogLevel)
	if err != nil {
		panic(err)
	}
	zerolog.SetGlobalLevel(zerolog.Level(logLevel))

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, NoColor: true}).With().Caller().Logger()

	conn, err := src.InitDB(src.DBName)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	defer conn.Close()

	s := src.NewServe()
	s.InitServer(portFlag, conn)
}
