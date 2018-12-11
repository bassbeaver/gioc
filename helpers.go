package gioc

import (
	"errors"
	"reflect"
	"strconv"
)

func createFactoryFromInterface(factory interface{}) *Factory {
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

	return factoryObj
}

func checkFactoryMethod(factoryMethod interface{}) {
	factoryMethodType := reflect.TypeOf(factoryMethod)

	if factoryMethodType.Kind() != reflect.Func {
		panic("Invalid kind of factory method. Factory method can be only a function")
	}

	// factory method must return only one parameter - pointer to new service instance
	if factoryMethodType.NumOut() != 1 {
		panic("Factory must return only one parameter")
	} else if factoryMethodType.Out(0).Kind() != reflect.Ptr && factoryMethodType.Out(0).Kind() != reflect.Interface {
		panic("Factory must return pointer")
	}
}

func getArgumentValueFromString(kind reflect.Kind, stringValue string) interface{} {
	errorProcessor := func(err error, kind reflect.Kind, stringValue string) {
		if err != nil {
			panic("Failed to convert '" + stringValue + "' to " + kind.String())
		}
	}

	switch kind {
	case reflect.String:
		return stringValue
	case reflect.Int:
		fallthrough
	case reflect.Int8:
		fallthrough
	case reflect.Int16:
		fallthrough
	case reflect.Int32:
		fallthrough
	case reflect.Int64:
		intVal, conversionError := strconv.ParseInt(stringValue, 0, 64)
		errorProcessor(conversionError, kind, stringValue)
		switch kind {
		case reflect.Int:
			return int(intVal)
		case reflect.Int8:
			return int8(intVal)
		case reflect.Int16:
			return int16(intVal)
		case reflect.Int32:
			return int32(intVal)
		}
		return intVal
	case reflect.Uint:
		fallthrough
	case reflect.Uint8:
		fallthrough
	case reflect.Uint16:
		fallthrough
	case reflect.Uint32:
		fallthrough
	case reflect.Uint64:
		intVal, conversionError := strconv.ParseUint(stringValue, 0, 64)
		errorProcessor(conversionError, kind, stringValue)
		switch kind {
		case reflect.Uint:
			return uint(intVal)
		case reflect.Uint8:
			return uint8(intVal)
		case reflect.Uint16:
			return uint16(intVal)
		case reflect.Uint32:
			return uint32(intVal)
		}
		return intVal
	case reflect.Float32:
		fallthrough
	case reflect.Float64:
		floatVal, conversionError := strconv.ParseFloat(stringValue, 64)
		errorProcessor(conversionError, kind, stringValue)
		switch kind {
		case reflect.Float32:
			return float32(floatVal)
		}
		return floatVal
	case reflect.Bool:
		boolVal, conversionError := strconv.ParseBool(stringValue)
		errorProcessor(conversionError, kind, stringValue)
		return boolVal
	default:
		errorProcessor(errors.New("no conversion logic found"), kind, stringValue)
	}
	return nil
}
