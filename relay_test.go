package thermabox

import (
	"fmt"
	"testing"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/stianeikeland/go-rpio"
	"github.com/stretchr/testify/require"
)

type FakeRelay struct {
	activeHigh bool       `yaml:"active_high"`
	pins       []rpio.Pin `yaml:"pins"`
	SwitchMap  map[int]uint8
}

func (f *FakeRelay) ActiveHigh() bool {
	return f.activeHigh
}
func (f *FakeRelay) commonHandle(swtch int) error {
	if _, ok := f.SwitchMap[swtch]; !ok {
		return fmt.Errorf("Switch %v not initialized in relay", swtch)
	} else {
		return nil
	}
}
func (f *FakeRelay) Toggle(swtch int) error {
	return f.commonHandle(swtch)
}
func (f *FakeRelay) On(swtch int) error {
	return f.commonHandle(swtch)
}
func (f *FakeRelay) Off(swtch int) error {
	return f.commonHandle(swtch)
}

func (f *FakeRelay) GetSwitchMap() map[int]uint8 {
	return f.SwitchMap
}

func (f *FakeRelay) UnmarshalYAML(unmarshal func(i interface{}) error) error {
	m := make(map[string]interface{})
	err := unmarshal(&m)
	if err != nil {
		return err
	}
	if _, ok := m["active_high"]; !ok {
		m["active_high"] = false
	}
	activeHigh := m["active_high"].(bool)
	pinsInterface := m["pins"].([]interface{})
	pins := make([]rpio.Pin, len(pinsInterface))
	sMap := make(map[int]uint8)
	for i := 0; i < len(pins); i++ {
		pins[i] = rpio.Pin(pinsInterface[i].(int))
		sMap[int(pins[i])] = 1
	}
	relay := &FakeRelay{activeHigh, pins, sMap}
	*f = *relay
	return nil
}

func genFakeRelay(activeHigh bool, pins []int) *FakeRelay {
	f := FakeRelay{}
	f.activeHigh = activeHigh
	p := make([]rpio.Pin, len(pins))
	sMap := make(map[int]uint8)
	for idx, pin := range pins {
		p[idx] = rpio.Pin(pin)
		sMap[pin] = 1
	}
	f.pins = p
	f.SwitchMap = sMap
	return &f
}

func getRelay(require *require.Assertions) *Relay {
	relay, err := NewRelay(false, []int{23, 22})
	require.Nil(err)
	require.NotNil(relay)
	return relay
}

func TestParseRelayYaml(t *testing.T) {
	require := require.New(t)

	str := `
active_high: false
pins: [14, 17, 18]
`

	relay := &FakeRelay{}
	err := yaml.Unmarshal([]byte(str), relay)
	require.Nil(err)
	require.True(relay.ActiveHigh())
	require.Equal(3, len(relay.SwitchMap))
	require.NotNil(relay.SwitchMap[14])
	require.NotNil(relay.SwitchMap[17])
	require.NotNil(relay.SwitchMap[18])
}

func TestRelay(t *testing.T) {
	require := require.New(t)
	getRelay(require)
	// Add a small delay here to ensure that multiple tests don't toggle relay too quickly
	time.Sleep(300 * time.Millisecond)
}

func TestRelayToggle(t *testing.T) {
	require := require.New(t)

	relay := getRelay(require)

	// Test switch 1
	err := relay.Toggle(1)
	require.Nil(err)
	time.Sleep(500 * time.Millisecond)
	err = relay.Toggle(1)
	require.Nil(err)
	time.Sleep(500 * time.Millisecond)

	// Test switch 2
	err = relay.Toggle(2)
	require.Nil(err)
	time.Sleep(500 * time.Millisecond)
	err = relay.Toggle(2)
	require.Nil(err)
}

func TestRelayOnOff(t *testing.T) {
	require := require.New(t)

	relay := getRelay(require)

	// Test switch 1
	err := relay.On(1)
	require.Nil(err)
	time.Sleep(500 * time.Millisecond)
	err = relay.Off(1)
	require.Nil(err)
	time.Sleep(500 * time.Millisecond)

	// Test switch 2
	err = relay.On(2)
	require.Nil(err)
	time.Sleep(500 * time.Millisecond)
	err = relay.Off(2)
	require.Nil(err)
}
