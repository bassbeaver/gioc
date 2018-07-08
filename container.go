package gioc

import (
	"reflect"
	"fmt"
)

type container struct {
	factories map[string]interface{}
	services map[string]interface{}
}

// Registers service factory to container. Parameter factory must be one of two types:
// 1. Factory method (function). Function with one out parameter - pointer to new instance of service
// 2. Instance of Factory struct, where Create attribute is proper factory method (see p.1)
func (c *container) RegisterServiceFactoryByAlias(serviceAlias string, factory interface{}) *container {
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

func (c *container) RegisterServiceFactoryByObject(serviceObj interface{}, factory interface{}) *container {
	serviceAlias := reflect.TypeOf(serviceObj).String()
	c.RegisterServiceFactoryByAlias(serviceAlias, factory)
	return c
}

func (c *container) AddServiceAlias(existingAlias, newAlias string) *container {
	if _, factoryIsRegistered := c.factories[existingAlias]; !factoryIsRegistered {
		return c
	}

	c.factories[newAlias] = c.factories[existingAlias]

	// Add new alias for already instantiated services
	if _, serviceIsRegistered := c.services[existingAlias]; serviceIsRegistered {
		c.services[newAlias] = c.services[existingAlias]
	}

	return c
}

func (c *container) AddServiceAliasByObject(serviceObj interface{}, newAlias string) *container {
	return c.AddServiceAlias(reflect.TypeOf(serviceObj).String(), newAlias)
}

func (c *container) GetByAlias(alias string) interface{} {
	if service, serviceIsRegistered := c.services[alias]; serviceIsRegistered {
		return service
	}

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
	c.services[alias] = service

	return service
}

func (c *container) GetByObject(serviceObj interface{}) interface{} {
	return c.GetByAlias(reflect.TypeOf(serviceObj).String())
}

// ---------------------------------------------------------------------------------------------------------------------

func NewContainer() *container {
	c := container{
		factories: make(map[string]interface{}, 0),
		services: make(map[string]interface{}, 0),
	}

	return &c
}
