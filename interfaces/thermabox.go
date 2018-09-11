package interfaces

type State string

const (
	HEATING_UP   State = "heating_up"
	COOLING_DOWN State = "cooling_down"
	STABLE       State = "stable"
	UNKNOWN      State = "unknown"
)

type TemperatureSensorInterface interface {
	GetTemperature() (float64, error)
}

type ThermaboxState struct {
	Temperature float64 `json:"temperature"`
	Timestamp   int64   `json:"timestamp"`
	State       State   `json:"state"`
}

type ThermaboxListenerInterface interface {
	RegisterChannel(chan *ThermaboxState)
}

type ThermaboxInterface interface {
	TemperatureSensorInterface
	ThermaboxListenerInterface
	SetLimits(temperature float64, threshold float64)
	GetLimits() (temperature float64, threshold float64)
	GetState() string
}
