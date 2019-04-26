package thermabox

import (
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/gurupras/thermabox/interfaces"
	"github.com/gurupras/thermabox/webserver"
	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
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
	/*	// FIXME: This doesn't seem to be working
		if e.ToggleDelay > 0 {
			now := time.Now()
			sinceLastOn := now.Sub(e.lastOn)
			sleepDuration := e.ToggleDelay.Nanoseconds() - sinceLastOn.Nanoseconds()
			if sleepDuration > 0 {
				return ElementToggleDelayError{"Minimum delay not elapsed"}
			}
		}
	*/
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
	cutoffTemp           float64  `yaml:"cutoff_temperature"`
	probe                interfaces.TemperatureSensorInterface
	extraProbes          []interfaces.TemperatureSensorInterface
	state                interfaces.State
	listeners            []chan *interfaces.ThermaboxState
	*webserver.Webserver `yaml:"webserver"`
	disabled             bool `yaml:"disabled"`
	mutex                sync.Mutex
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
	if _, ok := m["cutoff_temperature"]; !ok {
		m["cutoff_temperature"] = 0.0
	}
	cutoffTemp, err := strconv.ParseFloat(fmt.Sprintf("%v", m["cutoff_temperature"]), 64)
	if err != nil {
		return fmt.Errorf("Failed while parsing cutoff_temperature: %v", err)
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
		ws := webserver.New()
		b, _ := yaml.Marshal(m["webserver"])
		if err := yaml.Unmarshal(b, ws); err != nil {
			return err
		}
		t.Webserver = ws
	}

	if _, ok := m["extra_probes"]; ok {
		b, _ := yaml.Marshal(m["extra_probes"])
		var extraProbes []HTTPProbe
		if err := yaml.Unmarshal(b, &extraProbes); err != nil {
			return err
		}
		t.extraProbes = make([]interfaces.TemperatureSensorInterface, len(extraProbes))
		for idx, probe := range extraProbes {
			t.extraProbes[idx] = &probe
			log.Infof("Added extra probe: %v\n", probe)
		}
	}

	t.temperature = temperature
	t.threshold = threshold
	t.cutoffAtThreshold = cutoffAtThreshold
	t.cutoffTemp = cutoffTemp
	t.listeners = make([]chan *interfaces.ThermaboxState, 0)
	return nil
}

func (t *Thermabox) RegisterChannel(c chan *interfaces.ThermaboxState, name string) {
	log.Infof("Registered channel: %v", name)
	t.listeners = append(t.listeners, c)
}

func (t *Thermabox) GetTemperature() (float64, error) {
	return t.probe.GetTemperature()
}

func (t *Thermabox) GetAllTemperatures() map[string]interface{} {
	ret := make(map[string]interface{})
	probes := make([]interfaces.TemperatureSensorInterface, 0)
	probes = append(probes, t)
	probes = append(probes, t.extraProbes...)
	for _, probe := range probes {
		val, err := probe.GetTemperature()
		entry := make(map[string]interface{})
		if err != nil {
			entry["error"] = fmt.Sprintf("Failed to get temperature from probe: %v: %v", probe.GetName(), err)
			log.Errorf("%v\n", entry["error"])
		} else {
			// log.Debugf("%v: %v\n", probe.GetName(), val)
			entry["temp"] = val
		}
		ret[probe.GetName()] = entry
	}
	return ret
}

func (t *Thermabox) GetName() string {
	return "thermabox"
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

func (t *Thermabox) GetState() string {
	return fmt.Sprintf("%v", t.state)
}

func (t *Thermabox) DisableThermabox() {
	t.disabled = true
	t.heatingElement.Off()
	t.coolingElement.Off()
}

func (t *Thermabox) EnableThermabox() {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.state = interfaces.UNKNOWN
	t.disabled = false
}

func (t *Thermabox) Run() error {
	if t.Webserver != nil {
		go t.Webserver.Start(t)
		defer t.Webserver.Stop()
	}

	lastState := interfaces.UNKNOWN
	t.state = interfaces.UNKNOWN
	lastTempTimestamp := time.Now().UnixNano() / 1000000
	var (
		upperLimit float64
		lowerLimit float64
	)
	for {
		lowerLimit = t.temperature - t.threshold
		upperLimit = t.temperature + t.threshold

		now := time.Now().UnixNano() / 1000000
		temp, err := t.GetTemperature()
		if err != nil {
			if now-lastTempTimestamp > 10*1e3 {
				log.Errorf("Failed to get temperature: %v", err)
				// Turn off all elements and exit
				t.heatingElement.Off()
				t.coolingElement.Off()
				log.Fatalf("Shutting down at time: %v", time.Now())
				break
			}
			continue
		}
		lastTempTimestamp = now

		if t.cutoffTemp != 0.0 && temp > t.cutoffTemp {
			log.Errorf("Temperature > cutoff temperature: %v > %v", temp, t.cutoffTemp)
			t.heatingElement.Off()
			t.coolingElement.Off()
			log.Fatalf("Shutting down at time: %v", time.Now())
			break
		}

		t.mutex.Lock()
		if t.state == interfaces.STABLE || t.state == interfaces.UNKNOWN {
			if temp < lowerLimit {
				// Temperature has dropped below threshold
				// Start heating element to warm it back up
				t.state = interfaces.HEATING_UP
				if !t.disabled {
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
				}
			} else if temp > upperLimit {
				t.state = interfaces.COOLING_DOWN
				if !t.disabled {
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
				}
			} else {
				t.state = interfaces.STABLE
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
				val := round(temp, 0.5, 1)
				expected := round(t.temperature, 0.5, 1)
				switch t.state {
				case interfaces.HEATING_UP:
					if val >= expected {
						cutoff = true
					}
				case interfaces.COOLING_DOWN:
					if val <= expected {
						cutoff = true
					}
				case interfaces.UNKNOWN:
					if val >= lowerLimit && val <= upperLimit {
						cutoff = true
					}
				}
			}
			// Common cutoff logic
			if cutoff {
				log.Debugf("Attempting to turn off heating/cooling elements")
				t.state = interfaces.STABLE
				if err := t.heatingElement.Off(); err != nil {
					log.Errorf("Failed to turn off heating element: %v", err)
				}
				if err := t.coolingElement.Off(); err != nil {
					log.Errorf("Failed to turn off heating element: %v", err)
				}
			}
		}
		if lastState != t.state {
			log.Infof("temp=%.2f target=%.2f threshold=%.2f -> %v", temp, t.temperature, t.threshold, t.state)
			lastState = t.state
		}

		extras := t.GetAllTemperatures()
		go func(temperature float64, timestamp int64, state interfaces.State, extras map[string]interface{}) {
			tboxState := &interfaces.ThermaboxState{
				temperature,
				timestamp,
				state,
				extras,
			}
			for _, channel := range t.listeners {
				channel <- tboxState
			}
		}(temp, now, t.state, extras)
		t.mutex.Unlock()

		log.Debugf("temp=%v", temp)
		time.Sleep(500 * time.Millisecond)
	}
	return nil
}

// Borrowed from https://gist.github.com/DavidVaini/10308388
func round(val float64, roundOn float64, places int) (newVal float64) {
	var round float64
	pow := math.Pow(10, float64(places))
	digit := pow * val
	_, div := math.Modf(digit)
	if div >= roundOn {
		round = math.Ceil(digit)
	} else {
		round = math.Floor(digit)
	}
	newVal = round / pow
	return
}
