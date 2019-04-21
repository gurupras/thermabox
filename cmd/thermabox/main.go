package main

import (
	"io/ioutil"
	"os"

	yaml "gopkg.in/yaml.v2"

	"github.com/alecthomas/kingpin"
	temperusb "github.com/gurupras/go-TEMPerUSB"
	easyfiles "github.com/gurupras/go-easyfiles"
	"github.com/gurupras/thermabox"
	"github.com/gurupras/thermabox/interfaces"
	log "github.com/sirupsen/logrus"
)

var (
	app          = kingpin.New("ThermaBox", "Temperature-controller")
	conf         = app.Arg("conf", "Configuration file (YAML)").Required().String()
	verbose      = app.Flag("verbose", "Verbose logging").Short('v').Default("false").Bool()
	sensorSource = app.Flag("sensor", "Temperature sensor source").Short('S').Default("usb").String()
	temperature  = app.Flag("temperature", "Override conf temperature").Short('t').Default("-100").Float64()
	threshold    = app.Flag("threshold", "Override conf threshold").Short('T').Default("-100").Float64()
)

func main() {
	kingpin.MustParse(app.Parse(os.Args[1:]))
	if *verbose {
		log.SetLevel(log.DebugLevel)
	}

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

	def_temperature, def_threshold := tbox.GetLimits()
	if *temperature != -100 {
		def_temperature = *temperature
	}
	if *threshold != -100 {
		def_threshold = *threshold
	}
	tbox.SetLimits(def_temperature, def_threshold)

	// Get a hold of the temperature sensor
	var sensor interfaces.TemperatureSensorInterface
	switch *sensorSource {
	case "usb":
		fallthrough
	case "USB":
		s, err := temperusb.New()
		if err != nil {
			log.Fatalf("Failed to acquire temperature sensor: %v", err)
		}
		sensor = &thermabox.ProbeTemperUSB{
			s,
			"usb-temperusb-sensor",
		}
	default:
		// Assumes HTTP
		sensor = &thermabox.HTTPProbe{
			Url: *sensorSource,
			Name: "thermabox-probe",
		}
	}
	tbox.SetProbe(sensor)

	// We now have the thermabox ready
	log.Fatalf("%v", tbox.Run())
}
