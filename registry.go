package gioc

import (
	"reflect"
	"sync"
)

type registryEntry struct {
	factory        *Factory
	cachingEnabled bool
	cachedService  interface{}
	id             int
}

// ---------------------------------------------------------------------------------------------------------------------

type registry struct {
	mutex           sync.RWMutex
	aliasIndex      map[string]*registryEntry
	typeIndex       map[reflect.Type]*registryEntry
	servicesCounter int
}

func (r *registry) writeAlias(alias string, entry *registryEntry) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.servicesCounter++
	entry.id = r.servicesCounter
	r.aliasIndex[alias] = entry
}

func (r *registry) writeType(typeObj reflect.Type, entry *registryEntry) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.servicesCounter++
	entry.id = r.servicesCounter
	r.typeIndex[typeObj] = entry
}

func (r *registry) readAlias(alias string) *registryEntry {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	entry, entryExists := r.aliasIndex[alias]
	if !entryExists {
		panic("service with alias '" + alias + "' not found in Containers registry")
	}

	return entry
}

func (r *registry) readType(typeObj reflect.Type) *registryEntry {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	entry, entryExists := r.typeIndex[typeObj]
	if !entryExists {
		panic("service with type '" + typeObj.String() + "' not found in Containers registry")
	}

	return entry
}

func (r *registry) addServiceToCache(alias string, service interface{}) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.aliasIndex[alias].cachedService = service
}

// ---------------------------------------------------------------------------------------------------------------------

func newRegistry() *registry {
	return &registry{
		aliasIndex: make(map[string]*registryEntry, 0),
		typeIndex:  make(map[reflect.Type]*registryEntry, 0),
	}
}
