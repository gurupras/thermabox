package thermabox

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type WSProbe struct {
	Url string `yaml:"url"`
	Name string `yaml:"name"`
	conn *websocket.Conn
}

func (p *WSProbe) Initialize() error {
	c, _, err := websocket.DefaultDialer.Dial(p.Url, nil)
	if err != nil {
		return err
	}
	p.conn = c
	return nil
}

func (p *WSProbe) GetTemperature() (float64, error) {
	var err error
	var temp float64
	var (
		body []byte
		bodyStr string
	)
	for i := 0; i < 5; i++ {
		if err := p.conn.WriteMessage(websocket.TextMessage, []byte("temp")); err != nil {
			fmt.Errorf("Failed to request temperature from probe: %v\n", err)
			goto retry
		}
		_, body, err = p.conn.ReadMessage()
		if err != nil {
			fmt.Errorf("Failed to get temperature: %v", err)
			goto retry
		}
		bodyStr = strings.TrimSpace(string(body))
		temp, err = strconv.ParseFloat(bodyStr, 64)
		if err != nil {
			fmt.Errorf("Failed to get temperature: %v", err)
			goto retry
		}
		return temp, nil
	retry:
		time.Sleep(100 * time.Millisecond)
	}
	return 0, err
}

func (p *WSProbe) GetName() string {
	return p.Name
}
