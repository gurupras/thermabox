package thermabox

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/parnurzeal/gorequest"
)

type HTTPProbe struct {
	Url string `yaml:"url"`
	Name string `yaml:"name"`
}

func (p *HTTPProbe) GetTemperature() (float64, error) {
	var err error
	for i := 0; i < 5; i++ {
		var bodyStr string
		var temp float64
		var _err error
		resp, body, errs := gorequest.New().Timeout(1 * time.Second).Get(p.Url).End()
		if len(errs) > 0 {
			err = fmt.Errorf("Failed to get temperature: %v", errs)
			goto retry
		}

		if resp != nil && resp.StatusCode != 200 {
			err = fmt.Errorf("Failed to get temperature: Received response code: %v", resp.StatusCode)
			goto retry
		}

		bodyStr = strings.TrimSpace(string(body))
		temp, _err = strconv.ParseFloat(bodyStr, 64)
		if _err != nil {
			err = fmt.Errorf("Failed to get temperature: %v", _err)
			goto retry
		}
		return temp, nil
	retry:
		time.Sleep(100 * time.Millisecond)
	}
	return 0, err
}

func (p *HTTPProbe) GetName() string {
	return p.Name
}
