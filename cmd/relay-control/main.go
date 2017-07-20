package main

import (
	"bufio"
	"os"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/gurupras/thermabox"
	log "github.com/sirupsen/logrus"

	"github.com/stianeikeland/go-rpio"
)

var (
	app     = kingpin.New("relay-control", "Relay control")
	pin     = app.Arg("pin", "Pin to control").Default("24").Int()
	verbose = app.Flag("verbose", "Verbose logging").Default("false").Bool()
)

func main() {
	kingpin.MustParse(app.Parse(os.Args[1:]))

	if *verbose {
		log.SetLevel(log.DebugLevel)
	}
	log.Infof("Testing pin: %v", *pin)

	err := rpio.Open()
	if err != nil {
		log.Fatalf("Error in rpio.Open: %v", err)
	}

	relay, err := thermabox.NewRelay(false, []int{*pin})
	if err != nil {
		log.Fatalf("Error in new relay: %v", err)
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		_, _ = reader.ReadString('\n')
		relay.Toggle(1)
		time.Sleep(3 * time.Second)
	}
}
