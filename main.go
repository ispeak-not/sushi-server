package main

import (
	"log"
	"os"
	"sushi/server"
	"sushi/worker"
)

func main() {

	args := os.Args[1:]
	argsLen := len(args)
	if argsLen > 0 && args[0] == "worker" {
		log.Fatal(worker.Start())
	} else {
		log.Fatal(server.Start())
	}
}
