package gioc

import "sync"

type registryEntry struct {
	factory        *Factory
	cachingEnabled bool
	cachedService  interface{}
	id             int
}

// ---------------------------------------------------------------------------------------------------------------------

type registry struct {
	mutex           sync.RWMutex
	content         map[string]*registryEntry
	servicesCounter int
}

func (r *registry) write(alias string, entry *registryEntry) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.servicesCounter++
	entry.id = r.servicesCounter
	r.content[alias] = entry
}

func (r *registry) read(alias string) *registryEntry {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.content[alias]
}

func (r *registry) addServiceToCache(alias string, service interface{}) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.content[alias].cachedService = service
}

// ---------------------------------------------------------------------------------------------------------------------

func newRegistry() *registry {
	return &registry{
		content: make(map[string]*registryEntry, 0),
	}
}
