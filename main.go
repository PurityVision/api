package main

import (
	"purity-vision-filter/src"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	src.InitServer()
}
