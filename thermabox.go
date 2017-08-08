package thermabox

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/gurupras/thermabox/interfaces"
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

type ElementToggleDelayError struct {
	msg string
}

func (e ElementToggleDelayError) Error() string {
	return e.msg
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
			return ElementToggleDelayError{"Minimum delay not elapsed"}
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
	cutoffAtThreshold    bool     `yaml:"cutoff_at_threshold"`
	probe                interfaces.TemperatureSensorInterface
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
	if _, ok := m["cutoff_at_threshold"]; !ok {
		m["cutoff_at_threshold"] = false
	}
	cutoffAtThreshold, ok := m["cutoff_at_threshold"].(bool)
	if !ok {
		return fmt.Errorf("Failed while parsing cutoff_at_threshold: %t", m["cutoff_at_threshold"])
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
	t.cutoffAtThreshold = cutoffAtThreshold
	return nil
}

func (t *Thermabox) GetTemperature() (float64, error) {
	return t.probe.GetTemperature()
}

func (t *Thermabox) SetLimits(temperature float64, threshold float64) {
	t.temperature = temperature
	t.threshold = threshold
}

func (t *Thermabox) GetLimits() (float64, float64) {
	return t.temperature, t.threshold
}

func (t *Thermabox) SetProbe(probe interfaces.TemperatureSensorInterface) {
	t.probe = probe
}

func (t *Thermabox) Run() error {
	if t.Webserver != nil {
		go t.Webserver.Start(t)
		defer t.Webserver.Stop()
	}

	lastState := UNKNOWN
	curState := UNKNOWN
	var (
		upperLimit float64
		lowerLimit float64
	)
	if t.cutoffAtThreshold {
		lowerLimit = t.temperature - t.threshold
		upperLimit = t.temperature + t.threshold
	} else {

	}

	for {
		temp, err := t.GetTemperature()
		if err != nil {
			log.Errorf("Failed to get temperature: %v", err)
			// Turn off all elements and exit
			t.heatingElement.Off()
			t.coolingElement.Off()
			log.Fatalf("Shutting down!")
			break
		}

		if !t.cutoffAtThreshold {
			// Use integer values
		}
		if temp < lowerLimit {
			// Temperature has dropped below threshold
			// Start heating element to warm it back up
			curState = HEATING_UP
			if err := t.coolingElement.Off(); err != nil {
				log.Errorf("Failed to turn off cooling element: %v", err)
			}
			if err := t.heatingElement.On(); err != nil {
				if _, ok := err.(ElementToggleDelayError); !ok {
					log.Errorf("Failed to turn on heating element: %v", err)
				} else {
					// This was just a regular toggle delay error..just continue
				}
			}
		} else if temp > upperLimit {
			curState = COOLING_DOWN
			if err := t.heatingElement.Off(); err != nil {
				log.Errorf("Failed to turn off heating element: %v", err)
			}
			if err := t.coolingElement.On(); err != nil {
				if _, ok := err.(ElementToggleDelayError); !ok {
					log.Errorf("Failed to turn on cooling element: %v", err)
				} else {
					// This was just a regular toggle delay error..just continue
				}
			}
		} else {
			// Check if stable
			// We now have 2 cases to deal with
			cutoff := false
			if t.cutoffAtThreshold {
				if temp >= lowerLimit && temp <= upperLimit {
					cutoff = true
				}
			} else if !t.cutoffAtThreshold {
				// We only deal with 1 decimal point of precision
				if toFixed(temp, 1) == toFixed(t.temperature, 1) {
					cutoff = true
				}
			}
			// Common cutoff logic
			if cutoff {
				curState = STABLE
				if err := t.heatingElement.Off(); err != nil {
					log.Errorf("Failed to turn off heating element: %v", err)
				}
				if err := t.coolingElement.Off(); err != nil {
					log.Errorf("Failed to turn off heating element: %v", err)
				}
			}
		}
		if lastState != curState {
			log.Infof("temp=%.2f target=%.2f threshold=%.2f -> %v", temp, t.temperature, t.threshold, curState)
			lastState = curState
		}
		log.Debugf("temp=%v", temp)
		time.Sleep(300 * time.Millisecond)
	}
	return nil
}

func toFixed(num float64, precision int) float64 {
	round := func(num float64) int {
		return int(num + math.Copysign(0.5, num))
	}

	output := math.Pow(10, float64(precision))
	return float64(round(num*output)) / output
}
