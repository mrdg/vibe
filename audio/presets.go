package audio

import "fmt"

type Device interface {
	Set(key string, val interface{}) error
	Get(key string) (interface{}, error)
}

type preset map[string]interface{}

var presets = map[string]preset{
	"lame-bass": preset{
		"level":       3.,
		"env.decay":   0.1,
		"env.sustain": 0.,
		"osc1.wave":   "saw",
		"osc2.wave":   "saw",
		"cutoff":      900.0,
	},
}

func LoadPreset(name string, d Device) error {
	p, ok := presets[name]
	if !ok {
		return fmt.Errorf("unknown preset: %v", name)
	}
	for k, v := range p {
		if err := d.Set(k, v); err != nil {
			return err
		}
	}
	return nil
}
