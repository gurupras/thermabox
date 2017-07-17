package thermabox

import (
	"testing"
	"time"

	"gopkg.in/yaml.v2"

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
	element.RelayInterface = &FakeRelay{}

	err := yaml.Unmarshal([]byte(str), element)
	require.Nil(err)

	require.Equal(false, element.RelayInterface.ActiveHigh())

	sMap := element.RelayInterface.GetSwitchMap()
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

	h := &Element{}
	h.RelayInterface = &FakeRelay{}
	c := &Element{}
	c.RelayInterface = &FakeRelay{}

	tbox := &Thermabox{}
	tbox.heatingElement = h
	tbox.coolingElement = c

	err := yaml.Unmarshal([]byte(str), tbox)
	require.Nil(err)

	expectedHeating := &Element{
		genFakeRelay(false, []int{22}), 0,
	}
	expectedCooling := &Element{
		genFakeRelay(false, []int{23}), 30 * time.Second,
	}
	require.Equal(expectedHeating, tbox.heatingElement)
	require.Equal(expectedCooling, tbox.coolingElement)
	require.Equal(45.0, tbox.temperature)
	require.Equal(0.5, tbox.threshold)
}
