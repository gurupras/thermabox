package thermabox

import (
	"testing"
	"time"

	"gopkg.in/yaml.v2"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestParseYamlElement(t *testing.T) {
	require := require.New(t)
	str := `
relay:
  active_high: false
  pins: [22]
toggle_delay_sec: 30
`
	element := &Element{}
	element.relay = &FakeRelay{}

	err := yaml.Unmarshal([]byte(str), element)
	require.Nil(err)

	require.Equal(false, element.relay.ActiveHigh())

	sMap := element.relay.GetSwitchMap()
	require.NotNil(sMap)
	require.Equal(1, len(sMap))
	require.NotNil(sMap[22])

	require.Equal(30*time.Second, element.ToggleDelay)
}

func TestParseYamlThermabox(t *testing.T) {
	require := require.New(t)
	str := `
heating_element:
  relay:
    active_high: false
    pins: [22]
cooling_element:
  relay:
    active_high: false
    pins: [23]
  toggle_delay_sec: 30
temperature: 45
threshold: 0.5
webserver:
  port: 8080
`

	log.SetLevel(log.DebugLevel)

	h := &Element{}
	h.relay = &FakeRelay{}
	c := &Element{}
	c.relay = &FakeRelay{}

	tbox := &Thermabox{}
	tbox.heatingElement = h
	tbox.coolingElement = c

	err := yaml.Unmarshal([]byte(str), tbox)
	require.Nil(err)

	expectedHeating := &Element{
		genFakeRelay(false, []int{22}), 0, time.Time{},
	}
	expectedCooling := &Element{
		genFakeRelay(false, []int{23}), 30 * time.Second, time.Time{},
	}
	require.Equal(expectedHeating, tbox.heatingElement)
	require.Equal(expectedCooling, tbox.coolingElement)
	require.Equal(45.0, tbox.temperature)
	require.Equal(0.5, tbox.threshold)
}
