package webserver

import (
	"encoding/json"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"

	yaml "gopkg.in/yaml.v2"

	"github.com/gorilla/websocket"
	"github.com/gurupras/go-stoppable-net-listener"
	thermabox_interfaces "github.com/gurupras/thermabox/interfaces"
	websockets "github.com/homesound/simple-websockets"
	"github.com/parnurzeal/gorequest"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createWebserver () (*Webserver, error) {
	conf := `
webserver:
  port: 8080
  path: ./..
`
	w := New()
	err := yaml.Unmarshal([]byte(conf), w)
	if err != nil {
		return nil, err
	}
	return w, nil
}

func TestYAMLUnmarshal(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	conf := `
webserver:
  port: 8080
  path: ./..
`
	w := New()
	err := yaml.Unmarshal([]byte(conf), w)
	require.Nil(err)

	assert.Equal(8080, w.Port)
	assert.Equal("./..", w.Path)

	conf = `
port: 8080
path: ./..
`
	w = New()
	err = yaml.Unmarshal([]byte(conf), w)
	require.Nil(err)

	assert.Equal(8080, w.Port)
	assert.Equal("./..", w.Path)

	// Now test with some extra stuff
	conf = `
random:
  a: 1
  b: 2
webserver:
  port: 8080
  path: ./..
`
	w = New()
	err = yaml.Unmarshal([]byte(conf), w)
	require.Nil(err)

	assert.Equal(8080, w.Port)
	assert.Equal("./..", w.Path)

	conf = `
webserver:
  port: 8080
  path: ./..
  publish:
    protocol: https
    host: test
    path: /api/test
`
	w = New()
	err = yaml.Unmarshal([]byte(conf), w)
	require.Nil(err)

	assert.NotNil(w.Publish)
	assert.Equal(w.Publish["protocol"], "https")
	assert.Equal(w.Publish["host"], "test")
	assert.Equal(w.Publish["path"], "/api/test")
}

func TestWebServer(t *testing.T) {
	require := require.New(t)

	w, err := createWebserver()
	require.Nil(err)
	handler, err := InitializeWebServer(".", "/", nil, nil, w)
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

type DummyThermaboxInterface struct {
	currentTemp float64
	temperature float64
	threshold   float64
}

func NewDummyThermaboxInterface() *DummyThermaboxInterface {
	d := DummyThermaboxInterface{}
	d.currentTemp = 114.14
	d.temperature = 114.14
	d.threshold = 0.2
	return &d
}

func (d *DummyThermaboxInterface) GetName () string {
	return "dummy-thermabox"
}

func (d *DummyThermaboxInterface) GetTemperature() (float64, error) {
	return d.currentTemp, nil
}
func (d *DummyThermaboxInterface) GetLimits() (temp float64, threshold float64) {
	return d.temperature, d.threshold
}
func (d *DummyThermaboxInterface) SetLimits(temp float64, threshold float64) {
	d.temperature = temp
	d.threshold = threshold
}

func (d *DummyThermaboxInterface) RegisterChannel (channel chan *thermabox_interfaces.ThermaboxState, name string) {
}

func (d *DummyThermaboxInterface) DisableThermabox() {
}

func (d *DummyThermaboxInterface) EnableThermabox() {
}

func (d *DummyThermaboxInterface) GetAllTemperatures() map[string]interface{} {
	return nil
}


func (d *DummyThermaboxInterface) GetState() string {
	temp, _ := d.GetTemperature()
	if temp < d.temperature-d.threshold {
		return "heating_up"
	} else if temp >= d.temperature-d.threshold && temp <= d.temperature+d.threshold {
		return "stable"
	} else {
		return "cooling_down"
	}
}

func TestWebsockets(t *testing.T) {
	require := require.New(t)

	w, err := createWebserver()
	require.Nil(err)
	handler, err := InitializeWebServer(".", "/", NewDummyThermaboxInterface(), nil, w)
	require.Nil(err)
	require.NotNil(handler)

	mux := http.NewServeMux()
	mux.Handle("/", handler)
	server := http.Server{}
	server.Handler = mux
	snl, err := stoppablenetlistener.New(31122)
	require.Nil(err)
	require.NotNil(snl)
	defer snl.Stop()
	go func() {
		server.Serve(snl)
	}()

	time.Sleep(100 * time.Millisecond)

	u := url.URL{
		Scheme: "ws",
		Host:   "localhost:31122",
		Path:   "/ws",
	}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	defer c.Close()
	require.Nil(err)
	require.NotNil(c)
	client := websockets.NewClient(c)
	go client.ProcessMessages()

	wg := sync.WaitGroup{}
	client.On("get-temperature", func(w *websockets.WebsocketClient, data interface{}) {
		log.Debugf("[client] Received get-temperature: %t", data)
		temp := data.(float64)
		require.Equal(114.14, temp)
		wg.Done()
	})
	for i := 0; i < 10; i++ {
		wg.Add(1)
		err = client.Emit("get-temperature", nil)
		require.Nil(err)
		wg.Wait()
		log.Debugf("get-temperature passed")
	}

	wg.Add(1)
	m := make(map[string]interface{})
	m["temperature"] = 42.2
	m["threshold"] = 0.4
	client.On("set-limits", func(w *websockets.WebsocketClient, data interface{}) {
		client.On("get-limits", func(w *websockets.WebsocketClient, data interface{}) {
			require.Equal(m, data)
			wg.Done()
		})
		client.Emit("get-limits", nil)
	})
	err = client.Emit("set-limits", m)
	require.Nil(err)
	wg.Wait()

	// Now get-state
	// It should be heating up
	log.Debugf("Testing get-state")
	wg.Add(1)
	client.On("get-state", func(w *websockets.WebsocketClient, data interface{}) {
		require.Equal("cooling_down", data)
		wg.Done()
	})
	client.Emit("get-state", nil)
	wg.Wait()
}

// Test whether we are able to handle webserver under paths other than "/"
// FIXME: Don't know how to write this test. We need to be able to run
// two independent servers without registering paths via http.Handle*
func TestSubWebServer(t *testing.T) {
	require := require.New(t)

	w, err := createWebserver()
	require.Nil(err)
	handler, err := InitializeWebServer(".", "/webserver", nil, nil, w)
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

func TestPOSTSetLimits(t *testing.T) {
	require := require.New(t)

	tbox := NewDummyThermaboxInterface()
	w, err := createWebserver()
	require.Nil(err)
	handler, err := InitializeWebServer(".", "/", tbox, nil, w)
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

	data := make(map[string]interface{})
	data["temperature"] = 14.4
	data["threshold"] = 2.5

	b, err := json.Marshal(data)
	require.Nil(err)

	agent := gorequest.New()
	resp, errs, body := agent.Post("http://localhost:31123/set-limits").Type("form").Send(string(b)).EndBytes()
	_ = errs
	_ = body
	log.Debugf("resp: \n%v\n", resp)
	log.Debugf("errs: \n%v\n", errs)
	log.Debugf("body: \n%v\n", body)
	require.Equal(200, resp.StatusCode)

	// Now get the limits via get-limits
	resp, errs, body = gorequest.New().Get("http://localhost:31123/get-limits").EndBytes()
	require.Equal(200, resp.StatusCode)

	m := make(map[string]interface{})
	err = json.Unmarshal(b, &m)
	require.Nil(err)
	require.Equal(data["temperature"], m["temperature"])
	require.Equal(data["threshold"], m["threshold"])

	// Also verify that they match in the dummy object
	temp, threshold := tbox.GetLimits()
	require.Equal(data["temperature"], temp)
	require.Equal(data["threshold"], threshold)
	snl.Stop()
}
