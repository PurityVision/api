package main

import (
	"fmt"
	"os"
	"purity-vision-filter/src"
	"strings"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	if len(os.Args) < 2 {
		src.InitServer()
	} else {
		switch strings.ToLower(os.Args[1]) {
		case "server":
			src.InitServer()
		case "license":
			fmt.Println(src.GenerateLicenseKey())
		default:
			fmt.Println("unsupported command")
		}
	}
}
