package thermabox

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/parnurzeal/gorequest"
)

type HTTPProbe struct {
	Url string `yaml:"url"`
}

func (p *HTTPProbe) GetTemperature() (float64, error) {
	resp, body, errs := gorequest.New().Get(p.Url).End()
	if len(errs) > 0 {
		return 0, fmt.Errorf("Failed to get temperature: %v", errs)
	}

	if resp != nil && resp.StatusCode != 200 {
		return 0, fmt.Errorf("Failed to get temperature: Received response code: %v", resp.StatusCode)
	}

	bodyStr := strings.TrimSpace(string(body))
	temp, err := strconv.ParseFloat(bodyStr, 64)
	if err != nil {
		return 0, fmt.Errorf("Failed to get temperature: %v", err)
	}
	return temp, nil
}
