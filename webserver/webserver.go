package webserver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	yaml "gopkg.in/yaml.v2"

	"github.com/googollee/go-socket.io"
	"github.com/gorilla/mux"
	"github.com/gurupras/go-stoppable-net-listener"
	log "github.com/sirupsen/logrus"
)

type Webserver struct {
	Port int    `yaml:"port"`
	Path string `yaml:"path"`
	snl  *stoppablenetlistener.StoppableNetListener
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
	w.Port = m["port"].(int)
	w.Path = m["path"].(string)
	return nil
}

func (w *Webserver) Stop() {
	if w.snl != nil {
		w.snl.Stop()
	}
}
func (w *Webserver) Start(tbox ThermaboxInterface) {
	handler, err := InitializeWebServer(w.Path, "/", tbox, nil)
	if err != nil {
		log.Fatalf("%v", err)
	}
	mux := http.NewServeMux()
	mux.Handle("/", handler)
	server := http.Server{}
	server.Handler = mux
	snl, err := stoppablenetlistener.New(w.Port)
	if err != nil {
		log.Fatalf("%v", err)
	}
	w.snl = snl
	server.Serve(snl)
}

type ThermaboxInterface interface {
	GetTemperature() float64
	SetLimits(temperature float64, threshold float64)
	GetLimits() (temperature float64, threshold float64)
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

func GetTemperatureHandler(tbox ThermaboxInterface, w http.ResponseWriter, req *http.Request) error {
	w.WriteHeader(200)
	w.Write([]byte(fmt.Sprintf("%v", tbox.GetTemperature())))
	return nil
}

func GetTemperatureLimits(tbox ThermaboxInterface, w http.ResponseWriter, req *http.Request) error {
	w.WriteHeader(200)
	temp, threshold := tbox.GetLimits()
	m := make(map[string]interface{})
	m["temperature"] = temp
	m["threshold"] = threshold
	b, _ := json.Marshal(m)
	w.Write(b)
	return nil
}

func SetTemperatureLimits(tbox ThermaboxInterface, w http.ResponseWriter, req *http.Request) error {
	w.WriteHeader(200)
	temp, err := strconv.ParseFloat(req.FormValue("temperature"), 64)
	if err != nil {
		return fmt.Errorf("Failed to parse float64: %v: %v", req.FormValue("temperature"), err)
	}
	threshold, err := strconv.ParseFloat(req.FormValue("threshold"), 64)
	if err != nil {
		return fmt.Errorf("Failed to parse float64: %v: %v", req.FormValue("threshold"), err)
	}
	tbox.SetLimits(temp, threshold)
	return nil
}

func InitializeWebServer(path string, webserverBasePath string, tbox ThermaboxInterface, io *socketio.Server) (http.Handler, error) {
	if io == nil {
		var err error
		if io, err = socketio.NewServer(nil); err != nil {
			return nil, fmt.Errorf("Failed to initialize socket.io: %v", err)
		}
		//http.Handle("/socket.io/", io)
	}

	// Set up socket.io routes
	io.OnConnect("/", func(s socketio.Conn) error {
		log.Infof("Received connection")
		s.SetContext("")
		return nil
	})
	io.OnEvent("/", "set-limits", func(s socketio.Conn, msg string) {
		log.Infof("socket.io [set-limits]: type=%t", msg)
		m := make(map[string]interface{})
		err := json.Unmarshal([]byte(msg), &m)
		if err != nil {
			log.Errorf("socket.io [set-limits]: Failed to unmarshal message '%v': %v", msg, err)
			return
		}
		temp := m["temp"].(float64)
		threshold := m["threshold"].(float64)
		tbox.SetLimits(temp, threshold)
		log.Infof("socket.io [set-limits]: Set limits to %v (+/- %v)", temp, threshold)
		s.Emit("set-limits", "haha")
		s.Close()
	})

	io.OnEvent("/", "get-temperature", func(s socketio.Conn) {
		temp := tbox.GetTemperature()
		m := make(map[string]interface{})
		m["temp"] = temp
		log.Debugf("socket.io [get-temperature]: Sending back temp: %v", temp)
		b, _ := json.Marshal(m)
		s.Emit("get-temperature", string(b))
		s.Close()
	})

	staticPath := "static"
	webserverBasePath += "/"
	webserverBasePath = filepath.Clean(webserverBasePath)
	staticPath = filepath.Join(webserverBasePath, "static") + "/"
	log.Infof("webserverBasePath=%v staticPath=%v", webserverBasePath, staticPath)

	r := mux.NewRouter()
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
		if err := GetTemperatureHandler(tbox, w, req); err != nil {
			msg := fmt.Sprintf("Failed to handle '/get-temperature': %v", err)
			log.Errorf(msg)
			w.WriteHeader(503)
			w.Write([]byte(msg))
		}
	})
	r.HandleFunc(filepath.Join(webserverBasePath, "get-limits/"), func(w http.ResponseWriter, req *http.Request) {
		if err := GetTemperatureLimits(tbox, w, req); err != nil {
			msg := fmt.Sprintf("Failed to handle '/get-limits': %v", err)
			log.Errorf(msg)
			w.WriteHeader(503)
			w.Write([]byte(msg))
		}
	})
	r.HandleFunc(filepath.Join(webserverBasePath, "set-limits/"), func(w http.ResponseWriter, req *http.Request) {
		if err := SetTemperatureLimits(tbox, w, req); err != nil {
			msg := fmt.Sprintf("Failed to handle '/set-limits': %v", err)
			log.Errorf(msg)
			w.WriteHeader(503)
			w.Write([]byte(msg))
		}
	})

	r.PathPrefix(staticPath).Handler(http.StripPrefix(staticPath, http.FileServer(http.Dir(filepath.Join(path, "static")))))
	socketioPath := filepath.Join(webserverBasePath, "socket.io") + "/"
	r.Handle(socketioPath, io)
	return r, nil
}
