package thermabox

import temperusb "github.com/gurupras/go-TEMPerUSB"

type ProbeTemperUSB struct {
  *temperusb.Temper
  Name string
}

func (p *ProbeTemperUSB) GetName() string {
  return p.Name
}
