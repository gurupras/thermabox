package main

import (
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	yaml "gopkg.in/yaml.v2"

	"github.com/alecthomas/kingpin"
	easyfiles "github.com/gurupras/go-easyfiles"
	"github.com/gurupras/go-stoppable-net-listener"
	"github.com/gurupras/thermabox/webserver"
	log "github.com/sirupsen/logrus"
)

type DummyThermaBoxInterface struct {
	temperature float64
	threshold   float64
}

func (d *DummyThermaBoxInterface) GetTemperature() (float64, error) {
	return 100.00, nil
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
	path    = app.Flag("path", "Path to static files").Short('P').Default("../../").String()
	verbose = app.Flag("verbose", "Verbose logging").Short('v').Default("false").Bool()
	conf    = app.Flag("conf", "Webserver YAML conf").Short('c').String()
)

func main() {
	kingpin.MustParse(app.Parse(os.Args[1:]))

	if *verbose {
		log.SetLevel(log.DebugLevel)
	}

	if strings.Compare(*conf, "") != 0 {
		if !easyfiles.Exists(*conf) {
			log.Fatalf("Configuration file '%v' does not exist!")
		}
		ws := webserver.Webserver{}
		data, err := ioutil.ReadFile(*conf)
		if err != nil {
			log.Fatalf("Failed to read configuration file '%v': %v", *conf, err)
		}
		if err := yaml.Unmarshal(data, &ws); err != nil {
			log.Fatalf("Failed to unmarshal configuration file '%v': %v", *conf, err)
		}
		*port = ws.Port
		*path = ws.Path
	}

	dummy := &DummyThermaBoxInterface{}
	handler, err := webserver.InitializeWebServer(*path, "/", dummy, nil)
	if err != nil {
		log.Fatalf("%v", err)
	}

	http.Handle("/", handler)
	server := http.Server{}
	snl, err := stoppablenetlistener.New(*port)
	if err != nil {
		log.Fatalf("%v", err)
	}
	log.Debugf("Starting webserver on port: %v", *port)
	server.Serve(snl)
}
