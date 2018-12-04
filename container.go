package gioc

import (
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"reflect"
)

type Container struct {
	registry      *registry
	parameters    *parametersBag
	taskManager   *taskManager
	cyclesChecked bool
}

// Registers service factory to Container. Parameter factory must be one of two types:
// 1. Factory method (function). Function with one out parameter - pointer to new instance of service
// 2. Instance of Factory struct, where Create attribute is proper factory method (see p.1)
func (c *Container) RegisterServiceFactoryByAlias(serviceAlias string, factory interface{}, enableCaching bool) *Container {
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

	c.registry.write(serviceAlias, &registryEntry{
		factory:        factoryObj,
		cachingEnabled: enableCaching,
		cachedService:  nil,
	})

	return c
}

func (c *Container) RegisterServiceFactoryByObject(serviceObj interface{}, factory interface{}, enableCaching bool) *Container {
	serviceAlias := reflect.TypeOf(serviceObj).String()
	c.RegisterServiceFactoryByAlias(serviceAlias, factory, enableCaching)

	return c
}

func (c *Container) AddServiceAlias(existingAlias, newAlias string) bool {
	if serviceEntry := c.registry.read(existingAlias); nil != serviceEntry {
		c.registry.write(newAlias, serviceEntry)
		return true
	}
	return false
}

func (c *Container) AddServiceAliasByObject(serviceObj interface{}, newAlias string) bool {
	serviceName := reflect.TypeOf(serviceObj).String()

	return c.AddServiceAlias(serviceName, newAlias)
}

func (c *Container) GetByAlias(alias string) interface{} {
	if !c.cyclesChecked {
		if noCycles, cycledService := c.CheckCycles(); !noCycles {
			panic("Service " + cycledService + " has circular dependencies")
		}
	}

	registryEntry := c.registry.read(alias)
	if nil == registryEntry {
		panic(
			fmt.Sprintf("Failed to instantiate service '%s'. Factory for service '%s' not registered", alias, alias),
		)
	}

	if nil != registryEntry.cachedService {
		return registryEntry.cachedService
	}

	serviceCreationListener := make(chan *taskResult)
	c.taskManager.addTask(&taskDefinition{
		taskName: fmt.Sprintf("service_%d", registryEntry.id),
		listener: serviceCreationListener,
		perform: func() (interface{}, error) {
			return c.instantiate(registryEntry.factory)
		},
	})

	instantiationResult := <-serviceCreationListener
	if nil != instantiationResult.taskError {
		panic(
			fmt.Sprintf("Failed to instantiate service '%s'. Error: %s", alias, instantiationResult.taskError.Error()),
		)
	}
	service := instantiationResult.result

	if registryEntry.cachingEnabled {
		c.registry.addServiceToCache(alias, service)
	}

	return service
}

func (c *Container) GetByObject(serviceObj interface{}) interface{} {
	return c.GetByAlias(reflect.TypeOf(serviceObj).String())
}

func (c *Container) instantiate(factory *Factory) (interface{}, error) {
	factoryMethodValue := reflect.ValueOf(factory.Create)

	factoryMethodType := factoryMethodValue.Type()
	factoryInputArguments := make([]reflect.Value, factoryMethodType.NumIn())
	for argumentNum := 0; argumentNum < factoryMethodType.NumIn(); argumentNum++ {
		var argument interface{}

		argumentType := factoryMethodType.In(argumentNum)

		// If there is argument data for current argument - process it
		if argumentNum < len(factory.Arguments) {
			argumentDefinition := factory.Arguments[argumentNum]
			// Sign @ indicates that it is service alias
			if "@" == argumentDefinition[:1] {
				argument = c.GetByAlias(argumentDefinition[1:])
				// Sign # indicates that it is container parameter
			} else if "#" == argumentDefinition[:1] {
				parameter := argumentDefinition[1:]
				if !c.parameters.IsSet(parameter) {
					return nil, errors.New(fmt.Sprintf("Container's parameter '%s' not found in Container's parameters bag", parameter))
				}
				argument = getArgumentValueFromString(argumentType.Kind(), c.parameters.GetString(parameter))
			} else {
				argument = getArgumentValueFromString(argumentType.Kind(), argumentDefinition)
			}
			// If there is no data for current argument - just get it from Container
		} else {
			argument = c.GetByAlias(argumentType.String())
		}

		factoryInputArguments[argumentNum] = reflect.ValueOf(argument)
	}

	service := factoryMethodValue.Call(factoryInputArguments)[0].Interface()

	return service, nil
}

// Checks all registered services for dependency cycles.
// First returning parameter is flag showing cycle presence - true - no cycles, false - cycles detected
// Second returning parameter contains name of service with detected dependency cycle. If no cycles detected it is empty string
func (c *Container) CheckCycles() (bool, string) {
	noCycles, cycledService := checkCyclesForContainer(c)
	if noCycles {
		c.cyclesChecked = true
	}

	return noCycles, cycledService
}

func (c *Container) SetParameters(parameters *viper.Viper) {
	c.parameters.Replace(parameters)
}

func (c *Container) Close() {
	c.taskManager.stopServe()
}

// ---------------------------------------------------------------------------------------------------------------------

func NewContainer() *Container {
	c := Container{
		registry:      newRegistry(),
		parameters:    newParametersBag(),
		taskManager:   newTaskManager(),
		cyclesChecked: false,
	}
	c.taskManager.serve()

	return &c
}
