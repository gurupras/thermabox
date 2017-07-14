package thermabox

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRelay(t *testing.T) {
	require := require.New(t)

	relay, err := NewRelay(true, []int{23, 22})
	require.Nil(err)
	require.NotNil(relay)

}

func TestRelayToggle(t *testing.T) {
	require := require.New(t)

	relay, err := NewRelay(false, []int{23, 22})
	require.Nil(err)
	require.NotNil(relay)

	// Test switch 1
	err = relay.Toggle(1)
	require.Nil(err)
	time.Sleep(500 * time.Second)
	err = relay.Toggle(1)
	require.Nil(err)

	// Test switch 2
	err = relay.Toggle(2)
	require.Nil(err)
	time.Sleep(500 * time.Second)
	err = relay.Toggle(2)
	require.Nil(err)

}
