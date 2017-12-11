package webserver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	yaml "gopkg.in/yaml.v2"

	"github.com/gorilla/mux"
	"github.com/gurupras/go-stoppable-net-listener"
	thermabox_interfaces "github.com/gurupras/thermabox/interfaces"
	websockets "github.com/homesound/simple-websockets"
	"github.com/parnurzeal/gorequest"
	"github.com/rs/cors"
	log "github.com/sirupsen/logrus"
)

type Webserver struct {
	Port    int    `yaml:"port"`
	Path    string `yaml:"path"`
	Forward string `yaml:"forward"`
	snl     *stoppablenetlistener.StoppableNetListener
}

// Expects configuration to be under 'webserver'
func (w *Webserver) UnmarshalYAML(unmarshal func(i interface{}) error) error {
	m := make(map[string]interface{})
	if err := unmarshal(&m); err != nil {
		return err
	}
	var b []byte
	var err error
	data, ok := m["webserver"]
	if !ok {
		log.Debugf("No key 'webserver' found while unmarshalling Webserver")
		goto unmarshal
	}
	b, err = yaml.Marshal(data)
	if err != nil {
		return err
	}
	m = make(map[string]interface{})
	if err := yaml.Unmarshal(b, &m); err != nil {
		return err
	}
unmarshal:
	if port, ok := m["port"]; ok {
		w.Port = port.(int)
	} else {
		w.Port = 80
	}

	if path, ok := m["path"]; ok {
		w.Path = path.(string)
	} else {
		w.Path = "."
	}
	if forward, ok := m["forward"]; ok {
		w.Forward = forward.(string)
	} else {
		w.Forward = ""
	}
	return nil
}

func (w *Webserver) Stop() {
	if w.snl != nil {
		log.Info("Stopping webserver on port: %v", w.Port)
		w.snl.Stop()
		w.snl = nil
	}
}

func (w *Webserver) SetLimits(tbox thermabox_interfaces.ThermaboxInterface, temp float64, threshold float64) {
	if strings.Compare(w.Forward, "") != 0 {
		req := gorequest.New()
		data := make(map[string]float64)
		data["temperature"] = temp
		data["threshold"] = threshold
		req.Post(w.Forward).Send(data).End()
	}
	tbox.SetLimits(temp, threshold)
}

func (w *Webserver) Start(tbox thermabox_interfaces.ThermaboxInterface) {
	handler, err := InitializeWebServer(w.Path, "/", tbox, nil, w)
	if err != nil {
		log.Fatalf("%v", err)
	}
	mux := http.NewServeMux()
	mux.Handle("/", handler)
	corsHandler := cors.Default().Handler(mux)
	server := http.Server{}
	server.Handler = corsHandler
	snl, err := stoppablenetlistener.New(w.Port)
	if err != nil {
		log.Fatalf("%v", err)
	}
	w.snl = snl
	log.Info("Starting webserver on port: %v", w.Port)
	server.Serve(snl)
}

func IndexHandler(path string, w http.ResponseWriter, req *http.Request) error {
	indexFile := filepath.Join(path, "static", "html", "index.html")
	f, err := os.Open(indexFile)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(w, f)
	return err
}

func GetTemperatureHandler(webserver *Webserver, tbox thermabox_interfaces.ThermaboxInterface, w http.ResponseWriter, req *http.Request) error {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(200)
	temp, err := tbox.GetTemperature()
	if err != nil {
		return err
	}
	w.Write([]byte(fmt.Sprintf("%v", temp)))
	return nil
}

func GetTemperatureLimits(webserver *Webserver, tbox thermabox_interfaces.ThermaboxInterface, w http.ResponseWriter, req *http.Request) error {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(200)
	temp, threshold := tbox.GetLimits()
	m := make(map[string]interface{})
	m["temperature"] = temp
	m["threshold"] = threshold
	b, _ := json.Marshal(m)
	w.Write(b)
	return nil
}

func SetTemperatureLimits(webserver *Webserver, tbox thermabox_interfaces.ThermaboxInterface, w http.ResponseWriter, req *http.Request) error {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(200)
	temp, err := strconv.ParseFloat(req.FormValue("temperature"), 64)
	if err != nil {
		return fmt.Errorf("Failed to parse float64: %v: %v", req.FormValue("temperature"), err)
	}
	threshold, err := strconv.ParseFloat(req.FormValue("threshold"), 64)
	if err != nil {
		return fmt.Errorf("Failed to parse float64: %v: %v", req.FormValue("threshold"), err)
	}
	webserver.SetLimits(tbox, temp, threshold)
	return nil
}

func GetStateHandler(webserver *Webserver, tbox thermabox_interfaces.ThermaboxInterface, w http.ResponseWriter, req *http.Request) error {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(200)
	state := tbox.GetState()
	w.Write([]byte(state))
	return nil
}

func InitializeWebServer(path string, webserverBasePath string, tbox thermabox_interfaces.ThermaboxInterface, ws *websockets.WebsocketServer, webserver *Webserver) (http.Handler, error) {
	r := mux.NewRouter()
	if ws == nil {
		ws = websockets.NewServer(r)
	}

	// Set up websocket routes
	ws.On("get-limits", func(w *websockets.WebsocketClient, data interface{}) {
		temp, threshold := tbox.GetLimits()
		m := make(map[string]interface{})
		m["temperature"] = temp
		m["threshold"] = threshold
		w.Emit("get-limits", m)
	})
	ws.On("set-limits", func(w *websockets.WebsocketClient, data interface{}) {
		log.Infof("[websockets]: [set-limits]: type=%t", data)
		m := data.(map[string]interface{})
		temp := m["temperature"].(float64)
		threshold := m["threshold"].(float64)
		webserver.SetLimits(tbox, temp, threshold)
		log.Infof("[websockets]: [set-limits]: Set limits to %v (+/- %v)", temp, threshold)
		w.Emit("set-limits", "OK")
	})

	ws.On("get-temperature", func(w *websockets.WebsocketClient, data interface{}) {
		temp, err := tbox.GetTemperature()
		if err != nil {
			log.Errorf("Failed to get temperature: %v", err)
			return
		}
		log.Debugf("[websockets]: [get-temperature]: Sending back temp: %v", temp)
		w.Emit("get-temperature", temp)
	})
	ws.On("get-state", func(w *websockets.WebsocketClient, data interface{}) {
		state := tbox.GetState()
		w.Emit("get-state", state)
	})

	staticPath := "static"
	webserverBasePath += "/"
	webserverBasePath = filepath.Clean(webserverBasePath)
	staticPath = filepath.Join(webserverBasePath, "static") + "/"
	log.Infof("webserverBasePath=%v staticPath=%v", webserverBasePath, staticPath)

	r.HandleFunc(webserverBasePath, func(w http.ResponseWriter, req *http.Request) {
		if err := IndexHandler(path, w, req); err != nil {
			msg := fmt.Sprintf("Failed to handle '/': %v", err)
			log.Errorf(msg)
			w.WriteHeader(503)
			w.Write([]byte(msg))
		}
	})
	// Extra paths
	r.HandleFunc(filepath.Join(webserverBasePath, "get-temperature/"), func(w http.ResponseWriter, req *http.Request) {
		if err := GetTemperatureHandler(webserver, tbox, w, req); err != nil {
			msg := fmt.Sprintf("Failed to handle '/get-temperature': %v", err)
			log.Errorf(msg)
			w.WriteHeader(503)
			w.Write([]byte(msg))
		}
	})
	r.HandleFunc(filepath.Join(webserverBasePath, "get-limits/"), func(w http.ResponseWriter, req *http.Request) {
		if err := GetTemperatureLimits(webserver, tbox, w, req); err != nil {
			msg := fmt.Sprintf("Failed to handle '/get-limits': %v", err)
			log.Errorf(msg)
			w.WriteHeader(503)
			w.Write([]byte(msg))
		}
	})
	r.HandleFunc(filepath.Join(webserverBasePath, "set-limits/"), func(w http.ResponseWriter, req *http.Request) {
		if err := SetTemperatureLimits(webserver, tbox, w, req); err != nil {
			msg := fmt.Sprintf("Failed to handle '/set-limits': %v", err)
			log.Errorf(msg)
			w.WriteHeader(503)
			w.Write([]byte(msg))
		}
	})
	r.HandleFunc(filepath.Join(webserverBasePath, "get-state/"), func(w http.ResponseWriter, req *http.Request) {
		if err := GetStateHandler(webserver, tbox, w, req); err != nil {
			msg := fmt.Sprintf("Failed to handle '/get-state': %v", err)
			log.Errorf(msg)
			w.WriteHeader(503)
			w.Write([]byte(msg))
		}
	})

	r.PathPrefix(staticPath).Handler(http.StripPrefix(staticPath, http.FileServer(http.Dir(filepath.Join(path, "static")))))
	return r, nil
}
