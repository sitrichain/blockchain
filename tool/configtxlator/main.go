package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/tool/configtxlator/rest"
	"gopkg.in/alecthomas/kingpin.v2"
)

// command line flags
var (
	app = kingpin.New("configtxlator", "Utility for generating Rongzer Blockchain channel configurations")

	start    = app.Command("start", "Start the configtxlator REST server")
	hostname = start.Flag("hostname", "The hostname or IP on which the REST server will listen").Default("0.0.0.0").String()
	port     = start.Flag("port", "The port on which the REST server will listen").Default("7059").Int()
)

func main() {
	kingpin.Version("0.0.1")
	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	// "start" command
	case start.FullCommand():
		startServer(fmt.Sprintf("%s:%d", *hostname, *port))
	}
}

func startServer(address string) {
	log.Logger.Infof("Serving HTTP requests on %s", address)
	err := http.ListenAndServe(address, rest.NewRouter())

	app.Fatalf("Error starting server:[%s]\n", err)
}
