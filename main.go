package main

import (
	"flag"
	"fmt"
	"github.com/vvampirius/parcel-tracker/config"
	"log"
	"net/http"
	"os"
)

const VERSION  = `0.3.1`

var (
	ErrorLog = log.New(os.Stderr, `error#`, log.Lshortfile)
	DebugLog = log.New(os.Stdout, `debug#`, log.Lshortfile)
)

func helpText() {
	fmt.Println(`# https://github.com/vvampirius/parcel-tracker`)
	flag.PrintDefaults()
}

func main() {
	help := flag.Bool("h", false, "print this help")
	ver := flag.Bool("v", false, "Show version")
	configFilePath := flag.String("c", "config.yaml", "configuration file")
	flag.Parse()

	if *help {
		helpText()
		os.Exit(0)
	}

	if *ver {
		fmt.Println(VERSION)
		os.Exit(0)
	}

	fmt.Printf("Starting version %s...\n", VERSION)

	configFile, err := config.NewConfigFile(*configFilePath)
	if err != nil { os.Exit(1) }

	core, err := NewCore(configFile)
	if err != nil { os.Exit(1) }

	go core.ScanRoutine()

	http.HandleFunc("/", core.TelegramHttpHandler)
	err = http.ListenAndServe(configFile.Config.Listen, nil) // set listen port
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
