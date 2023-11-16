package main

import (
	"purity-vision-filter/src"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	src.InitServer()
}
