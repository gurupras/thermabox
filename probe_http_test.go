package thermabox

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gorilla/mux"
	stoppablenetlistener "github.com/gurupras/go-stoppable-net-listener"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

func TestUnmarshalYamlProbeHTTP(t *testing.T) {
	require := require.New(t)
	str := `
url: "http://thermabox-probe/temp"
`
	probe := HTTPProbe{}
	err := yaml.Unmarshal([]byte(str), &probe)
	require.Nil(err)
	require.Equal("http://thermabox-probe/temp", probe.Url)
}

func TestProbeHTTP(t *testing.T) {
	require := require.New(t)

	testTemp := 42.2
	// Set up a fake endpoint that returns temperature
	tempHandler := func(w http.ResponseWriter, req *http.Request) {
		tempStr := fmt.Sprintf("%.2f", testTemp)
		w.Write([]byte(tempStr))
	}
	r := mux.NewRouter()
	r.HandleFunc("/temp", tempHandler)

	mux := http.NewServeMux()
	mux.Handle("/", r)
	server := http.Server{}
	server.Handler = mux
	snl, err := stoppablenetlistener.New(31121)
	require.Nil(err)
	require.NotNil(snl)
	go func() {
		server.Serve(snl)
	}()

	probe := HTTPProbe{"http://localhost:31121/temp"}
	temp, err := probe.GetTemperature()
	require.Nil(err)
	require.Equal(testTemp, temp)
}
