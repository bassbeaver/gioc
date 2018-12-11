package gioc

import (
	"errors"
	"fmt"
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
	factoryObj := createFactoryFromInterface(factory)
	c.registry.writeAlias(
		serviceAlias,
		&registryEntry{
			factory:        factoryObj,
			cachingEnabled: enableCaching,
			cachedService:  nil,
		},
	)

	return c
}

// Registers service factory to Container. Parameter factory must be one of two types:
// 1. Factory method (function). Function with one out parameter - pointer to new instance of service
// 2. Instance of Factory struct, where Create attribute is proper factory method (see p.1)
func (c *Container) RegisterServiceFactoryByObject(serviceObj interface{}, factory interface{}, enableCaching bool) *Container {
	factoryObj := createFactoryFromInterface(factory)
	c.registry.writeType(
		reflect.TypeOf(serviceObj),
		&registryEntry{
			factory:        factoryObj,
			cachingEnabled: enableCaching,
			cachedService:  nil,
		},
	)

	return c
}

func (c *Container) AddServiceAlias(existingAlias, newAlias string) bool {
	if serviceEntry := c.registry.readAlias(existingAlias); nil != serviceEntry {
		c.registry.writeAlias(newAlias, serviceEntry)

		return true
	}

	return false
}

func (c *Container) AddServiceAliasByObject(serviceObj interface{}, newAlias string) bool {
	serviceType := reflect.TypeOf(serviceObj)
	if serviceEntry := c.registry.readType(serviceType); nil != serviceEntry {
		c.registry.writeAlias(newAlias, serviceEntry)

		return true
	}

	return false
}

func (c *Container) BindObjectToAlias(existingAlias string, serviceObj interface{}) bool {
	var serviceEntry *registryEntry

	if serviceEntry = c.registry.readAlias(existingAlias); nil == serviceEntry {
		return false
	}

	serviceType := reflect.TypeOf(serviceObj)

	factoryMethodType := reflect.TypeOf(serviceEntry.factory.Create)
	if serviceType != factoryMethodType.Out(0) {
		panic("serviceObj passed to function is not same type as service with alias " + existingAlias)
	}

	return false
}

func (c *Container) GetByAlias(alias string) interface{} {
	if !c.cyclesChecked {
		if noCycles, cycledService := c.CheckCycles(); !noCycles {
			panic("Service " + cycledService + " has circular dependencies")
		}
	}

	registryEntry := c.registry.readAlias(alias)
	if nil == registryEntry {
		panic(fmt.Sprintf("Failed to instantiate service '%s'. Factory for service not registered", alias))
	}

	service, serviceError := c.getByRegistryEntry(registryEntry)
	if nil != serviceError {
		panic(fmt.Sprintf("Failed to instantiate service '%s'. Error: %s", alias, serviceError.Error()))
	}

	return service
}

func (c *Container) GetByObject(serviceObj interface{}) interface{} {
	if !c.cyclesChecked {
		if noCycles, cycledService := c.CheckCycles(); !noCycles {
			panic("Service " + cycledService + " has circular dependencies")
		}
	}

	serviceType := reflect.TypeOf(serviceObj)

	return c.getByReflectType(serviceType)
}

func (c *Container) getByReflectType(serviceType reflect.Type) interface{} {
	registryEntry := c.registry.readType(serviceType)
	if nil == registryEntry {
		serviceTypeName := serviceType.String()
		panic(
			fmt.Sprintf("Failed to instantiate service with type '%s'. Factory for service type %s not registered", serviceTypeName, serviceTypeName),
		)
	}

	service, serviceError := c.getByRegistryEntry(registryEntry)
	if nil != serviceError {
		panic(
			fmt.Sprintf("Failed to instantiate service with type '%s'. Error: %s", serviceType.String(), serviceError.Error()),
		)
	}

	return service
}

func (c *Container) getByRegistryEntry(entry *registryEntry) (interface{}, error) {
	if nil != entry.cachedService {
		return entry.cachedService, nil
	}

	serviceCreationListener := make(chan *taskResult)
	c.taskManager.addTask(&taskDefinition{
		taskName: fmt.Sprintf("service_%d", entry.id),
		listener: serviceCreationListener,
		perform: func() (interface{}, error) {
			return c.instantiate(entry.factory)
		},
	})

	instantiationResult := <-serviceCreationListener
	if nil != instantiationResult.taskError {
		return nil, instantiationResult.taskError
	}
	service := instantiationResult.result

	if entry.cachingEnabled {
		entry.cachedService = service
	}

	return service, nil
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
			argumentDefinitionLen := len(argumentDefinition)

			if argumentDefinitionLen >= 1 && "@" == argumentDefinition[:1] {
				// Sign @ indicates that it is service alias
				argument = c.GetByAlias(argumentDefinition[1:])
			} else if argumentDefinitionLen >= 1 && "#" == argumentDefinition[:1] {
				// Sign # indicates that it is container parameter
				parameter := argumentDefinition[1:]
				if !c.parameters.IsSet(parameter) {
					return nil, errors.New(fmt.Sprintf("Container's parameter '%s' not found in Container's parameters bag", parameter))
				}
				argument = getArgumentValueFromString(argumentType.Kind(), c.parameters.GetString(parameter))
			} else {
				argument = getArgumentValueFromString(argumentType.Kind(), argumentDefinition)
			}
		} else {
			// If there is no data for current argument - just get it from Container
			argument = c.getByReflectType(argumentType)
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

func (c *Container) SetParameters(parameters map[string]string) {
	for key, val := range parameters {
		c.parameters.set(key, val)
	}
}

func (c *Container) Parameters() ParametersAccessor {
	return c.parameters
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
