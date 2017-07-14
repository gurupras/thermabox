package thermabox

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRelay(t *testing.T) {
	require := require.New(t)

	relay, err := NewRelay(true, []int{23, 22})
	require.Nil(err)
	require.NotNil(relay)

}
