package gioc

import (
	"reflect"
	"fmt"
	"sync"
)

type Container struct {
	factories map[string]interface{}
	servicesMutex sync.RWMutex
	services map[string]interface{}
	taskManager *taskManager
}

// Registers service factory to Container. Parameter factory must be one of two types:
// 1. Factory method (function). Function with one out parameter - pointer to new instance of service
// 2. Instance of Factory struct, where Create attribute is proper factory method (see p.1)
func (c *Container) RegisterServiceFactoryByAlias(serviceAlias string, factory interface{}) *Container {
	factoryType := reflect.TypeOf(factory)

	if factoryType.Kind() == reflect.Func {
		checkFactoryMethod(factory)
	} else if factoryType == reflect.TypeOf(Factory{}) {
		checkFactoryMethod(factory.(Factory).Create)
	} else {
		panic(
			fmt.Sprintf(
				"Invalid kind of 'factory' parameter: %s. Parameter 'factory' must be a function or %s instance",
				reflect.TypeOf(factory).Kind().String(),
				reflect.TypeOf(Factory{}).String(),
			),
		)
	}

	c.factories[serviceAlias] = factory

	return c
}

func (c *Container) RegisterServiceFactoryByObject(serviceObj interface{}, factory interface{}) *Container {
	serviceAlias := reflect.TypeOf(serviceObj).String()
	c.RegisterServiceFactoryByAlias(serviceAlias, factory)
	return c
}

func (c *Container) AddServiceAlias(existingAlias, newAlias string) *Container {
	if _, factoryIsRegistered := c.factories[existingAlias]; !factoryIsRegistered {
		return c
	}

	c.factories[newAlias] = c.factories[existingAlias]

	if service, serviceIsRegistered := c.getCachedService(existingAlias); serviceIsRegistered {
		c.setCachedService(newAlias, service)
	}

	return c
}

func (c *Container) AddServiceAliasByObject(serviceObj interface{}, newAlias string) *Container {
	return c.AddServiceAlias(reflect.TypeOf(serviceObj).String(), newAlias)
}

func (c *Container) GetByAlias(alias string) interface{} {
	if service, serviceIsRegistered := c.getCachedService(alias); serviceIsRegistered {
		return service
	}

	serviceCreationListener := make(chan interface{})
	c.taskManager.addTask(&taskDefinition{
		taskName: alias,
		listener: serviceCreationListener,
		perform: func() interface {} {
			return c.instantiate(alias)
		},
	})
	return <- serviceCreationListener
}

func (c *Container) GetByObject(serviceObj interface{}) interface{} {
	return c.GetByAlias(reflect.TypeOf(serviceObj).String())
}

func (c *Container) instantiate(alias string) interface{} {
	factory, factoryIsRegistered := c.factories[alias]
	if !factoryIsRegistered {
		panic(
			fmt.Sprintf(
				"Failed to instantiate '%s'. Factory for service '%s' not registered",
				alias,
				alias,
			),
		)
	}

	var factoryIsFunction bool
	var factoryMethod interface{}
	if reflect.TypeOf(factory) == reflect.TypeOf(Factory{}) {
		factoryIsFunction = false
		factoryMethod = factory.(Factory).Create
	} else {
		factoryIsFunction = true
		factoryMethod = factory
	}

	factoryMethodValue := reflect.ValueOf(factoryMethod)

	factoryMethodType := factoryMethodValue.Type()
	factoryInputArguments := make([]reflect.Value, factoryMethodType.NumIn())
	for argumentNum := 0; argumentNum < factoryMethodType.NumIn(); argumentNum++ {
		var argument interface{}

		argumentType := factoryMethodType.In(argumentNum)

		// If factory is Factory instance - check for Arguments data in it
		if !factoryIsFunction && argumentNum < len(factory.(Factory).Arguments) {
			argumentDefinition := factory.(Factory).Arguments[argumentNum]
			// Sign @ indicates that it is service alias
			if "@" == argumentDefinition[:1] {
				argument = c.GetByAlias(argumentDefinition[1:])
			} else {
				argument = getArgumentValueFromString(argumentType.Kind(), argumentDefinition)
			}
			// If factory is method of there are no data for current parameter - just get it from Container
		} else {
			argument = c.GetByAlias(argumentType.String())
		}

		factoryInputArguments[argumentNum] = reflect.ValueOf(argument)
	}

	service := factoryMethodValue.Call(factoryInputArguments)[0].Interface()

	// Save instantiated service to services hash-map
	c.setCachedService(alias, service)

	return service
}

func (c *Container) setCachedService(alias string, service interface{}) {
	c.servicesMutex.Lock()
	defer c.servicesMutex.Unlock()
	c.services[alias] = service
}

func (c *Container) getCachedService(alias string) (interface{}, bool) {
	c.servicesMutex.RLock()
	defer c.servicesMutex.RUnlock()
	service, serviceIsSet := c.services[alias]
	return service, serviceIsSet
}

func (c *Container) Close() {
	c.taskManager.stopServe()
}

// ---------------------------------------------------------------------------------------------------------------------

func NewContainer() *Container {
	c := Container{
		factories: make(map[string]interface{}, 0),
		services: make(map[string]interface{}, 0),
		taskManager: NewTaskManager(),
	}
	c.taskManager.serve()

	return &c
}
