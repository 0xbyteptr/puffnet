package main

import (
	"log"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: puffctl [host|serve|get|post|keygen] ...")
	}

	switch os.Args[1] {
	case "host":
		runHost()
	case "serve":
		runServe()
	case "get":
		runGet()
	case "post":
		runPost()
	case "keygen":
		if err := GenerateKeys(); err != nil {
			log.Fatal("Key generation failed:", err)
		}
	default:
		log.Fatal("Unknown command:", os.Args[1])
	}
}
