package webserver

import (
	"encoding/json"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/gurupras/go-stoppable-net-listener"
	"github.com/homesound/golang-socketio"
	"github.com/homesound/golang-socketio/transport"
	"github.com/parnurzeal/gorequest"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestWebServer(t *testing.T) {
	require := require.New(t)

	handler, err := InitializeWebServer(".", "/", nil, nil)
	require.Nil(err)
	require.NotNil(handler)

	mux := http.NewServeMux()
	mux.Handle("/", handler)
	server := http.Server{}
	server.Handler = mux
	snl, err := stoppablenetlistener.New(31121)
	require.Nil(err)
	require.NotNil(snl)
	go func() {
		server.Serve(snl)
	}()

	time.Sleep(100 * time.Millisecond)
	resp, errs, body := gorequest.New().Get("http://localhost:31121/").End()
	_ = errs
	_ = body
	log.Debugf("resp: \n%v\n", resp)
	log.Debugf("errs: \n%v\n", errs)
	log.Debugf("body: \n%v\n", body)
	require.Equal(200, resp.StatusCode)
	snl.Stop()
}

type DummyThermaBoxInterface struct{}

func (d DummyThermaBoxInterface) GetTemperature() float64 {
	return 114.14
}
func (d DummyThermaBoxInterface) GetLimits() (temp float64, threshold float64) {
	return 114.14, 0.2
}
func (d DummyThermaBoxInterface) SetLimits(temp float64, threshold float64) {
}

// FIXME: Socket.io client code is broken. Cannot run more than 1 request at the moment
func TestSocketIo(t *testing.T) {
	t.Skip()
	require := require.New(t)

	//io, err := socketio.NewServer(nil)
	//require.Nil(err)

	handler, err := InitializeWebServer(".", "/", DummyThermaBoxInterface{}, nil)
	require.Nil(err)
	require.NotNil(handler)

	mux := http.NewServeMux()
	mux.Handle("/", handler)
	// Attach socket.io
	//mux.Handle("/socket.io/", io)
	server := http.Server{}
	server.Handler = mux
	snl, err := stoppablenetlistener.New(31122)
	require.Nil(err)
	require.NotNil(snl)
	go func() {
		server.Serve(snl)
	}()

	time.Sleep(100 * time.Millisecond)

	c, err := gosocketio.Dial(
		gosocketio.GetUrl("localhost", 31122, false),
		transport.GetDefaultWebsocketTransport(),
	)
	require.Nil(err)
	require.NotNil(c)

	wg := sync.WaitGroup{}
	c.On("get-temperature", func(ch *gosocketio.Channel, s string) {
		m := make(map[string]interface{})
		err := json.Unmarshal([]byte(s), &m)
		require.Nil(err, "Failed to unmarshal data from server: %v: %v", s, err)
		temp := m["temp"]
		require.Equal(114.14, temp)
		wg.Done()
	})
	for i := 0; i < 10; i++ {
		wg.Add(1)
		err = c.Emit("get-temperature", nil)
		require.Nil(err)
		wg.Wait()
		log.Infof("get-temperature passed")
	}

	wg.Add(1)
	m := make(map[string]interface{})
	m["temperature"] = 42.2
	m["limits"] = 0.4
	b, err := json.Marshal(m)
	require.Nil(err)
	c.On("set-limits", func(ch *gosocketio.Channel, s string) {
		wg.Done()
	})
	_ = b
	err = c.Emit("set-limits", nil)
	require.Nil(err)
	wg.Wait()
}

// Test whether we are able to handle webserver under paths other than "/"
// FIXME: Don't know how to write this test. We need to be able to run
// two independent servers without registering paths via http.Handle*
func TestSubWebServer(t *testing.T) {
	require := require.New(t)

	handler, err := InitializeWebServer(".", "/webserver", nil, nil)
	require.Nil(err)
	require.NotNil(handler)

	mux := http.NewServeMux()
	mux.Handle("/", handler)
	server := http.Server{}
	server.Handler = mux
	snl, err := stoppablenetlistener.New(31123)
	require.Nil(err)
	require.NotNil(snl)
	go func() {
		server.Serve(snl)
	}()

	time.Sleep(100 * time.Millisecond)
	resp, errs, body := gorequest.New().Get("http://localhost:31123/webserver").End()
	_ = errs
	_ = body
	log.Debugf("resp: \n%v\n", resp)
	log.Debugf("errs: \n%v\n", errs)
	log.Debugf("body: \n%v\n", body)
	require.Equal(200, resp.StatusCode)
	snl.Stop()
}