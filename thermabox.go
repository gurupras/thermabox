package thermabox

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gurupras/thermabox/webserver"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type TemperatureSensorInterface interface {
	Temperature() (float64, error)
}

type Element struct {
	RelayInterface `yaml:"relay"`
	ToggleDelay    time.Duration `yaml:"toggle_delay_sec"`
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
	if e.RelayInterface == nil {
		e.RelayInterface = &Relay{}
	}
	if err := e.RelayInterface.UnmarshalYAML(relayUnmarshaler); err != nil {
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
