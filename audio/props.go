package audio

import (
	"fmt"
	"sync/atomic"
)

// Props stores device configuration that can be updated without locks. All properties
// should be registered before any reads take place.
type Props struct {
	properties map[string]*atomic.Value
	setters    map[string]setter
}

func NewProps() *Props {
	return &Props{
		properties: make(map[string]*atomic.Value),
		setters:    make(map[string]setter),
	}
}

// Set updates the property with value. The key has to be registered first using Register.
func (p *Props) Set(key string, value interface{}) error {
	prop, ok := p.properties[key]
	if !ok {
		return fmt.Errorf("unknown property %s", key)
	}
	set, ok := p.setters[key]
	if !ok {
		return fmt.Errorf("unknown property %s", key)
	}
	if err := set(value, prop); err != nil {
		return fmt.Errorf("set property %s: %w", key, err)
	}
	return nil
}

func (p *Props) Get(key string) (interface{}, error) {
	prop, ok := p.properties[key]
	if !ok {
		return nil, fmt.Errorf("unknown property %s", key)
	}
	return prop.Load(), nil
}

// Register adds a new property.
func (p *Props) Register(key string, set setter, init interface{}) (*atomic.Value, error) {
	var prop atomic.Value
	p.properties[key] = &prop
	p.setters[key] = set
	return &prop, set(init, &prop)
}

func (p *Props) MustRegister(key string, set setter, init interface{}) *atomic.Value {
	if prop, err := p.Register(key, set, init); err != nil {
		panic(err)
	} else {
		return prop
	}
}

type setter func(val interface{}, dest *atomic.Value) error

var (
	setEnvParam = setFloat64(0.0005, 15)
	setLevel    = setFloat64(-40, 10)
)

func setFloat64(min, max float64) setter {
	return func(v interface{}, dest *atomic.Value) error {
		var f float64
		switch n := v.(type) {
		case float64:
			f = n
		case int:
			f = float64(n)
		default:
			return fmt.Errorf("value is not a float64: %v", v)
		}
		if f < min || f > max {
			return fmt.Errorf("property value is not in valid range %v - %v: %v", min, max, f)
		}
		dest.Store(f)
		return nil
	}
}

func setInt(v interface{}, dest *atomic.Value) error {
	switch n := v.(type) {
	case float64:
		dest.Store(int(n))
	case int:
		dest.Store(n)
	default:
		return fmt.Errorf("value is not an int: %v", v)
	}
	return nil
}

func setString(v interface{}, dest *atomic.Value) error {
	if s, ok := v.(string); ok {
		dest.Store(s)
		return nil
	}
	return fmt.Errorf("value is not a string: %v", v)
}
