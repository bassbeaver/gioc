package gioc

import (
	"sync"
)

type ParametersAccessor interface {
	GetString(key string) string
	IsSet(key string) bool
}

type parametersBag struct {
	mutex      sync.RWMutex
	parameters map[string]string
}

func (p *parametersBag) set(key string, value string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.parameters[key] = value
}

func (p *parametersBag) GetString(key string) string {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.parameters[key]
}

func (p *parametersBag) IsSet(key string) bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	_, isset := p.parameters[key]

	return isset
}

// ---------------------------------------------------------------------------------------------------------------------

func newParametersBag() *parametersBag {
	return &parametersBag{
		parameters: make(map[string]string),
	}
}
