package main

import (
	"net/http"
	"os"

	"github.com/alecthomas/kingpin"
	"github.com/gurupras/go-stoppable-net-listener"
	"github.com/gurupras/thermabox/webserver"
	log "github.com/sirupsen/logrus"
)

type DummyThermaBoxInterface struct {
	temperature float64
	threshold   float64
}

func (d *DummyThermaBoxInterface) GetTemperature() float64 {
	return 100.00
}
func (d *DummyThermaBoxInterface) GetLimits() (float64, float64) {
	log.Debugf("get-temperature: Returning: %v +/- %v", d.temperature, d.threshold)
	return d.temperature, d.threshold
}

func (d *DummyThermaBoxInterface) SetLimits(temp float64, threshold float64) {
	log.Infof("Setting limits to: %v +/- %v", temp, threshold)
	d.temperature = temp
	d.threshold = threshold
}

var (
	app     = kingpin.New("webserver", "ThermaBox WebServer")
	port    = app.Flag("port", "webserver port").Short('p').Default("8080").Int()
	verbose = app.Flag("verbose", "Verbose logging").Short('v').Default("false").Bool()
)

func main() {
	kingpin.MustParse(app.Parse(os.Args[1:]))

	if *verbose {
		log.SetLevel(log.DebugLevel)
	}

	dummy := &DummyThermaBoxInterface{}
	handler, err := webserver.InitializeWebServer("../../", "/", dummy, nil)
	if err != nil {
		log.Fatalf("%v", err)
	}
	http.Handle("/", handler)
	server := http.Server{}
	snl, err := stoppablenetlistener.New(*port)
	if err != nil {
		log.Fatalf("%v", err)
	}
	server.Serve(snl)
}
