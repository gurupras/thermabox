package main

import (
	"bufio"
	"os"
	"time"

	"github.com/gurupras/thermabox"
	log "github.com/sirupsen/logrus"

	"github.com/stianeikeland/go-rpio"
)

func main() {
	err := rpio.Open()
	if err != nil {
		log.Fatalf("Error in rpio.Open: %v", err)
	}

	relay, err := thermabox.NewRelay(false, []int{22})
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
