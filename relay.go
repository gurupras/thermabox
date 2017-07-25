package thermabox

import (
	"fmt"

	"gopkg.in/yaml.v2"

	log "github.com/sirupsen/logrus"
	"github.com/stianeikeland/go-rpio"
)

type RelayInterface interface {
	yaml.Unmarshaler
	ActiveHigh() bool
	Toggle(swtch int) error
	On(swtch int) error
	Off(swtch int) error
	IsOn(swtch int) (bool, error)
	GetSwitchMap() map[int]uint8
}

type Relay struct {
	activeHigh bool       `yaml:"active_high"`
	pins       []rpio.Pin `yaml:"pins"`
	SwitchMap  map[int]uint8
}

func (r *Relay) ActiveHigh() bool {
	return r.activeHigh
}

func (r *Relay) GetSwitchMap() map[int]uint8 {
	return r.SwitchMap
}

func (r *Relay) UnmarshalYAML(unmarshal func(i interface{}) error) error {
	m := make(map[string]interface{})
	err := unmarshal(&m)
	if err != nil {
		return err
	}
	log.Debugf("Relay unmarshalling: %v", m)
	activeHigh := m["active_high"].(bool)
	pinsInterface := m["pins"].([]interface{})
	pins := make([]int, len(pinsInterface))
	for i := 0; i < len(pins); i++ {
		pins[i] = pinsInterface[i].(int)
	}
	r.activeHigh = activeHigh
	r.buildSwitchMap(pins)
	return nil
}

func NewRelay(activeHigh bool, gpioPins []int) (*Relay, error) {
	if err := rpio.Open(); err != nil {
		return nil, fmt.Errorf("Failed to call rpio.Open(): %v", err)
	}
	r := &Relay{}
	r.activeHigh = activeHigh
	r.buildSwitchMap(gpioPins)
	return r, nil
}

func (r *Relay) buildSwitchMap(gpioPins []int) error {
	pins := make([]rpio.Pin, len(gpioPins))
	switchMap := make(map[int]uint8)

	for idx, gpioPin := range gpioPins {
		pin := rpio.Pin(gpioPin)
		// Update switchMap
		switchMap[idx+1] = uint8(pin)
		// Set pin to output mode
		pin.Output()
		// Turn it off
		switch r.activeHigh {
		case true:
			pin.Low()
		case false:
			pin.High()
		}
		pins[idx] = pin
	}
	r.pins = pins
	r.SwitchMap = switchMap
	return nil
}

func (r *Relay) Toggle(swtch int) error {
	if p, ok := r.SwitchMap[swtch]; !ok {
		return fmt.Errorf("Switch %v not initialized in relay", swtch)
	} else {
		pin := rpio.Pin(p)
		pin.Toggle()
		return nil
	}
}

func (r *Relay) On(swtch int) error {
	if p, ok := r.SwitchMap[swtch]; !ok {
		return fmt.Errorf("Switch %v not initialized in relay", swtch)
	} else {
		pin := rpio.Pin(p)
		switch r.activeHigh {
		case true:
			pin.High()
		case false:
			pin.Low()
		}
	}
	return nil
}

func (r *Relay) Off(swtch int) error {
	if p, ok := r.SwitchMap[swtch]; !ok {
		return fmt.Errorf("Switch %v not initialized in relay", swtch)
	} else {
		pin := rpio.Pin(p)
		switch r.activeHigh {
		case true:
			pin.Low()
		case false:
			pin.High()
		}
	}
	return nil
}

func (r *Relay) IsOn(swtch int) (bool, error) {
	if p, ok := r.SwitchMap[swtch]; !ok {
		return false, fmt.Errorf("Switch %v not initialized in relay", swtch)
	} else {
		pin := rpio.Pin(p)
		state := pin.Read()
		var onState rpio.State
		switch r.activeHigh {
		case true:
			onState = rpio.High
		case false:
			onState = rpio.Low
		}
		return state == onState, nil
	}
}
