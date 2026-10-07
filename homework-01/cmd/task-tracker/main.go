package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/UnusualNick/ozon-go-homework/homework-01/internal/business"
	"github.com/UnusualNick/ozon-go-homework/homework-01/internal/storage/jsonfile"
	"github.com/UnusualNick/ozon-go-homework/homework-01/internal/ui"
)

func main() {
	dataPath := flag.String("data", "tasks.json", "path to the JSON data file")
	flag.Parse()

	repository := jsonfile.New(*dataPath)
	service := business.NewService(repository)
	console := ui.NewConsole(service, os.Stdin, os.Stdout, os.Stderr)

	if err := console.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %s\n", err)
		os.Exit(1)
	}
}
