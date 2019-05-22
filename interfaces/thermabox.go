package interfaces

type State string

const (
	HEATING_UP   State = "heating_up"
	COOLING_DOWN State = "cooling_down"
	STABLE       State = "stable"
	UNKNOWN      State = "unknown"
)

type TemperatureSensorInterface interface {
	Initialize() error
	GetTemperature() (float64, error)
	GetName() string
}

type ThermaboxState struct {
	Temperature float64                `json:"temperature"`
	Timestamp   int64                  `json:"timestamp"`
	State       State                  `json:"state"`
	Extras      map[string]interface{} `json:"extras"`
}

type ThermaboxListenerInterface interface {
	RegisterChannel(chan *ThermaboxState, string)
}

type ThermaboxInterface interface {
	TemperatureSensorInterface
	ThermaboxListenerInterface
	SetLimits(temperature float64, threshold float64)
	GetLimits() (temperature float64, threshold float64)
	GetState() string
	GetAllTemperatures() map[string]interface{}
	DisableThermabox()
	EnableThermabox()
}
