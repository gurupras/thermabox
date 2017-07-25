package thermabox

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gurupras/thermabox/webserver"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type State string

const (
	HEATING_UP   string = "heating_up"
	COOLING_DOWN string = "cooling_down"
	STABLE       string = "stable"
	UNKNOWN      string = "unknown"
)

type TemperatureSensorInterface interface {
	Temperature() (float64, error)
}

type Element struct {
	relay       RelayInterface `yaml:"relay"`
	ToggleDelay time.Duration  `yaml:"toggle_delay_sec"`
	lastOn      time.Time
}

func (e *Element) On() error {
	if e.ToggleDelay > 0 {
		now := time.Now()
		sinceLastOn := now.Sub(e.lastOn)
		sleepDuration := e.ToggleDelay.Nanoseconds() - sinceLastOn.Nanoseconds()
		if sleepDuration > 0 {
			time.Sleep(time.Duration(sleepDuration))
		}
	}
	return e.relay.On(1)
}

func (e *Element) Off() error {
	e.lastOn = time.Now()
	return e.relay.Off(1)
}

func (e *Element) Toggle() error {
	if isOn, err := e.relay.IsOn(1); err != nil {
		if isOn {
			return e.Off()
		} else {
			return e.On()
		}
	} else {
		return err
	}
}

func (e *Element) UnmarshalYAML(unmarshal func(i interface{}) error) error {
	m := make(map[string]interface{})
	if err := unmarshal(&m); err != nil {
		return err
	}
	log.Debugf("element.UnmarshalYAML: m=%v", m)
	relayUnmarshaler := func(i interface{}) error {
		b, _ := yaml.Marshal(m["relay"])
		return yaml.Unmarshal(b, i)
	}
	if e.relay == nil {
		e.relay = &Relay{}
	}
	if err := e.relay.UnmarshalYAML(relayUnmarshaler); err != nil {
		return err
	}
	if _, ok := m["toggle_delay_sec"]; !ok {
		m["toggle_delay_sec"] = 0
	}
	val := m["toggle_delay_sec"].(int)
	e.ToggleDelay = time.Duration(val) * time.Second
	return nil
}

type Thermabox struct {
	heatingElement       *Element `yaml:"heating_element"`
	coolingElement       *Element `yaml:"cooling_element"`
	temperature          float64  `yaml:"temperature"`
	threshold            float64  `yaml:"threshold"`
	probe                TemperatureSensorInterface
	*webserver.Webserver `yaml:"webserver"`
}

func (t *Thermabox) UnmarshalYAML(unmarshal func(i interface{}) error) error {
	m := make(map[string]interface{})

	if err := unmarshal(&m); err != nil {
		return err
	}
	log.Debugf("thermabox.UnmarshalYAML m=%v", m)

	elementUnmarshaler := func(key string, i interface{}) error {
		log.Debugf("elementUnmarshaler unmarshaling key: %v: %v", key, m[key])
		b, err := yaml.Marshal(m[key])
		if err != nil {
			return fmt.Errorf("Failed to marshal key '%v': %v: %v", key, m[key], err)
		}
		return yaml.Unmarshal(b, i)
	}

	heatingElementUnmarshaler := func(i interface{}) error {
		return elementUnmarshaler("heating_element", i)
	}
	coolingElementUnmarshaler := func(i interface{}) error {
		return elementUnmarshaler("cooling_element", i)
	}

	if _, ok := m["temperature"]; !ok {
		m["temperature"] = 0.0
	}
	if _, ok := m["threshold"]; !ok {
		m["threshold"] = 0.0
	}

	temperature, err := strconv.ParseFloat(fmt.Sprintf("%v", m["temperature"]), 64)
	if err != nil {
		return fmt.Errorf("Failed while parsing temperature: %v", err)
	}
	threshold, err := strconv.ParseFloat(fmt.Sprintf("%v", m["threshold"]), 64)
	if err != nil {
		return fmt.Errorf("Failed while parsing threshold: %v", err)
	}

	if t.heatingElement == nil {
		t.heatingElement = &Element{}
	}
	t.heatingElement.UnmarshalYAML(heatingElementUnmarshaler)

	if t.coolingElement == nil {
		t.coolingElement = &Element{}
	}
	t.coolingElement.UnmarshalYAML(coolingElementUnmarshaler)

	// Parse webserver
	if _, ok := m["webserver"]; ok {
		ws := &webserver.Webserver{}
		b, _ := yaml.Marshal(m["webserver"])
		if err := yaml.Unmarshal(b, ws); err != nil {
			return err
		}
		t.Webserver = ws
	}
	t.temperature = temperature
	t.threshold = threshold
	return nil
}

func (t *Thermabox) GetTemperature() (float64, error) {
	return t.probe.Temperature()
}

func (t *Thermabox) SetLimits(temperature float64, threshold float64) {
	t.temperature = temperature
	t.threshold = threshold
}

func (t *Thermabox) GetLimits() (float64, float64) {
	return t.temperature, t.threshold
}

func (t *Thermabox) SetProbe(probe TemperatureSensorInterface) {
	t.probe = probe
}

func (t *Thermabox) Run() error {
	lastState := UNKNOWN
	curState := UNKNOWN

	for {
		temp, err := t.GetTemperature()
		if err != nil {
			log.Errorf("Failed to get temperature: %v", err)
		}

		if temp <= t.temperature-t.threshold {
			// Temperature has dropped below threshold
			// Start heating element to warm it back up
			curState = HEATING_UP
			if err := t.coolingElement.Off(); err != nil {
				log.Errorf("Failed to turn off cooling element: %v", err)
			}
			if err := t.heatingElement.On(); err != nil {
				log.Errorf("Failed to turn on heating element: %v", err)
			}
		} else if temp >= t.temperature+t.threshold {
			curState = COOLING_DOWN
			if err := t.heatingElement.Off(); err != nil {
				log.Errorf("Failed to turn off heating element: %v", err)
			}
			if err := t.coolingElement.On(); err != nil {
				log.Errorf("Failed to turn on heating element: %v", err)
			}
		} else if temp > t.temperature-t.threshold && temp < t.temperature+t.threshold {
			curState = STABLE
			if err := t.heatingElement.Off(); err != nil {
				log.Errorf("Failed to turn off heating element: %v", err)
			}
			if err := t.coolingElement.Off(); err != nil {
				log.Errorf("Failed to turn off heating element: %v", err)
			}
		}
		if lastState != curState {
			log.Infof("temp=%.2f target=%.2f threshold=%.2f -> %v", temp, t.temperature, t.threshold, curState)
			lastState = curState
		}
		time.Sleep(100 * time.Millisecond)
	}
}
