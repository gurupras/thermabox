package main

import (
	"io/ioutil"
	"os"

	yaml "gopkg.in/yaml.v2"

	"github.com/alecthomas/kingpin"
	temperusb "github.com/gurupras/go-TEMPerUSB"
	easyfiles "github.com/gurupras/go-easyfiles"
	"github.com/gurupras/thermabox"
	log "github.com/sirupsen/logrus"
)

var (
	app  = kingpin.New("ThermaBox", "Temperature-controller")
	conf = app.Arg("conf", "Configuration file (YAML)").Required().String()
)

func main() {
	kingpin.MustParse(app.Parse(os.Args[1:]))

	if !easyfiles.Exists(*conf) {
		log.Fatalf("Configuration file '%v' does not exist")
	}

	tbox := thermabox.Thermabox{}
	data, err := ioutil.ReadFile(*conf)
	if err != nil {
		log.Fatalf("Failed to read conf file: %v", err)
	}

	if err := yaml.Unmarshal(data, &tbox); err != nil {
		log.Fatalf("Failed to unmarshal yaml: %v", err)
	}

	// Get a hold of the temperature sensor
	sensor, err := temperusb.New()
	if err != nil {
		log.Fatalf("Failed to acquire temperature sensor: %v", err)
	}
	tbox.SetProbe(sensor)

	// We now have the thermabox ready
	log.Fatalf("%v", tbox.Run())
}
