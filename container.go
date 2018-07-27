package gioc

import (
	"reflect"
	"fmt"
	"sync"
)

type Container struct {
	registryMutex   sync.RWMutex
	registry        map[string]*registryEntry
	servicesCounter int
	taskManager *taskManager
}

type registryEntry struct {
	factory *Factory
	cachedService interface{}
	id int
}

// Registers service factory to Container. Parameter factory must be one of two types:
// 1. Factory method (function). Function with one out parameter - pointer to new instance of service
// 2. Instance of Factory struct, where Create attribute is proper factory method (see p.1)
func (c *Container) RegisterServiceFactoryByAlias(serviceAlias string, factory interface{}) *Container {
	var factoryObj *Factory

	if reflect.TypeOf(factory) == reflect.TypeOf(Factory{}) {
		castedFactory := factory.(Factory)
		factoryObj = &castedFactory
	} else {
		factoryObj = &Factory{
			Create: factory,
		}
	}

	checkFactoryMethod(factoryObj.Create)

	c.writeRegistry(serviceAlias, &registryEntry{
		factory: factoryObj,
		cachedService: nil,
	})

	return c
}

func (c *Container) RegisterServiceFactoryByObject(serviceObj interface{}, factory interface{}) *Container {
	serviceAlias := reflect.TypeOf(serviceObj).String()
	c.RegisterServiceFactoryByAlias(serviceAlias, factory)

	return c
}

func (c *Container) AddServiceAlias(existingAlias, newAlias string) bool {
	if serviceEntry := c.readRegistry(existingAlias); nil != serviceEntry {
		c.writeRegistry(newAlias, serviceEntry)
		return true
	}
	return false
}

func (c *Container) AddServiceAliasByObject(serviceObj interface{}, newAlias string) bool {
	serviceName := reflect.TypeOf(serviceObj).String()

	return c.AddServiceAlias(serviceName, newAlias)
}

func (c *Container) GetByAlias(alias string) interface{} {
	registryEntry := c.readRegistry(alias)
	if nil == registryEntry {
		panic(
			fmt.Sprintf("Failed to instantiate service '%s'. Factory for service '%s' not registered", alias, alias),
		)
	}

	if nil != registryEntry.cachedService {
		return registryEntry.cachedService
	}

	serviceCreationListener := make(chan interface{})
	c.taskManager.addTask(&taskDefinition{
		taskName: fmt.Sprintf("service_%d", registryEntry.id),
		listener: serviceCreationListener,
		perform: func() interface {} {
			return c.instantiate(registryEntry.factory)
		},
	})

	service := <- serviceCreationListener

	c.addServiceToCache(alias, service)

	return service
}

func (c *Container) GetByObject(serviceObj interface{}) interface{} {
	return c.GetByAlias(reflect.TypeOf(serviceObj).String())
}

func (c *Container) instantiate(factory *Factory) interface{} {
	factoryMethodValue := reflect.ValueOf(factory.Create)

	factoryMethodType := factoryMethodValue.Type()
	factoryInputArguments := make([]reflect.Value, factoryMethodType.NumIn())
	for argumentNum := 0; argumentNum < factoryMethodType.NumIn(); argumentNum++ {
		var argument interface{}

		argumentType := factoryMethodType.In(argumentNum)

		// If there is argument data for current parameter - process it
		if argumentNum < len(factory.Arguments) {
			argumentDefinition := factory.Arguments[argumentNum]
			// Sign @ indicates that it is service alias
			if "@" == argumentDefinition[:1] {
				argument = c.GetByAlias(argumentDefinition[1:])
			} else {
				argument = getArgumentValueFromString(argumentType.Kind(), argumentDefinition)
			}
		// If there is no data for current parameter - just get it from Container
		} else {
			argument = c.GetByAlias(argumentType.String())
		}

		factoryInputArguments[argumentNum] = reflect.ValueOf(argument)
	}

	service := factoryMethodValue.Call(factoryInputArguments)[0].Interface()

	return service
}

func (c *Container) writeRegistry(alias string, entry *registryEntry) {
	c.registryMutex.Lock()
	defer c.registryMutex.Unlock()

	c.servicesCounter++
	entry.id = c.servicesCounter
	c.registry[alias] = entry
}

func (c *Container) addServiceToCache(alias string, service interface{}) {
	c.registryMutex.Lock()
	defer c.registryMutex.Unlock()
	c.registry[alias].cachedService = service
}

func (c *Container) readRegistry(alias string) *registryEntry {
	c.registryMutex.RLock()
	defer c.registryMutex.RUnlock()

	return c.registry[alias]
}

func (c *Container) Close() {
	c.taskManager.stopServe()
}

// ---------------------------------------------------------------------------------------------------------------------

func NewContainer() *Container {
	c := Container{
		registry:    make(map[string]*registryEntry, 0),
		taskManager: newTaskManager(),
	}
	c.taskManager.serve()

	return &c
}
