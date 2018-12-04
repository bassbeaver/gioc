package gioc

import (
	"github.com/spf13/viper"
	"sync"
)

type parametersBag struct {
	mutex      sync.RWMutex
	parameters *viper.Viper
}

func (p *parametersBag) set(key string, value interface{}) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.parameters.Set(key, value)
}

func (p *parametersBag) GetString(key string) string {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.parameters.GetString(key)
}

func (p *parametersBag) IsSet(key string) bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.parameters.IsSet(key)
}

func (p *parametersBag) Replace(newParameters *viper.Viper) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.parameters = newParameters
}

// ---------------------------------------------------------------------------------------------------------------------

func newParametersBag() *parametersBag {
	return &parametersBag{
		parameters: viper.New(),
	}
}
