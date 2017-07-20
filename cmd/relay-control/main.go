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
	app = kingpin.New("relay-control", "Relay control")
	pin = app.Arg("pin", "Pin to control").Default("24").Int()
)

func main() {
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
