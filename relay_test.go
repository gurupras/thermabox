package thermabox

import (
	"testing"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/stretchr/testify/require"
)

func getRelay(require *require.Assertions) *Relay {
	relay, err := NewRelay(false, []int{23, 22})
	require.Nil(err)
	require.NotNil(relay)
	return relay
}

func TestParseRelayYaml(t *testing.T) {
	require := require.New(t)

	str := `
active_high: true
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
