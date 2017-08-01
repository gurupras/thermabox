package interfaces

type TemperatureSensorInterface interface {
	GetTemperature() (float64, error)
}

type ThermaboxInterface interface {
	TemperatureSensorInterface
	SetLimits(temperature float64, threshold float64)
	GetLimits() (temperature float64, threshold float64)
}
