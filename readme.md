# gioc

Simple dependency injection container for Golang. 

### Installation

 ```bash
 dep ensure --add github.com/bassbeaver/gioc
 ```
 
### Concepts

Container operates with factories and each factory knows how to build it's service. 
If some service depends on other service - factory have to declare that dependency as it's (factories) dependency
and Container will inject it to factory, so the factory will be able to inject it into the service.

#### Factory <a id="factory"></a>

Factory (see [Factory pattern](https://en.wikipedia.org/wiki/Factory_method_pattern)) is a thing which knows how to create service.
Factory can be one of two types:
1. Function. This function must return only one parameter and this parameter must be a pointer to new service instance.
2. Factory struct from `gioc` package.
Factory struct has next definition: 
```go
type Factory struct {
	Arguments []string
	Create    interface{}
}
```  

`Create` must be a function which knows how to create service (requirements to this function are same as for function from p.1).

`Arguments` is an array of definitions for `Create` function arguments. N-th element of `Arguments` array is for N-th argument of `Create` function.

Each argument definition is a string and is interpreted in next ways:
* If first symbol of this string is `@` - this definition is interpreted as service alias, so Container will
try to find service with that alias
* In other cases definition string is interpreted as value for corresponding argument of `Create` function.

Container tries to cast values of `Arguments` to required type, if cast failed - Container panics.
 
Example:
 ```go
 type Service struct {
     Field1 string
     Field2 int
 }
 
factory := Factory {
    Create: func(f1 string, f2 int) *Service {
        return &Service{Field1: f1, Field2: f2}
    },
    Arguments: []string{"field 1", "123"},
},

```
Where value of `Arguments[0]` `("field 1")` will be passed to `factory.Create()` as argument `f1` and `Arguments[1]` `("123")` will be passed to `factory.Create()` as argument `f2`.
Container tries to cast values of `Arguments` to required type (`"field 1"` will be cast to `string` and `"123"` will be cast to `int`). If cast fails - Container panics.

#### Container usage

##### Container creation

To create container you should use `gioc.NewContainer()` method.

##### Service registration

To register a service factory you can use next methods of Container:
 
```
RegisterServiceFactoryByAlias(serviceAlias string, factory interface{})
```
Where `serviceAlias` is string alias (name) which will be used to retrieve this service from container
and `factory` is the factory which knows how to create that service (see [Factory](#factory)).

```
RegisterServiceFactoryByObject(serviceObj interface{}, factory interface{})
```
Where `serviceObj` is instance of type of service (really, it should be a pointer to that type) 
and `factory` is the factory which knows how to create that service (see [Factory](#factory)).

Service can have multiple aliases. To add new alias for registered service you can use:
```
AddServiceAlias(existingAlias, newAlias string)
```

##### Service retrieval

To get service from Container you can use:
```
GetByAlias(alias string) interface{}
GetByObject(serviceObj interface{}) interface{}
```

For now all instantiated services are cached, so for the first call of `GetByAlias` or `GetByObject` service is instantiated
and putted into cache and for every next call you will get service from cache.

**Notice:** Rules for controlling that cache (to disable caching and running service instantiation every time) will be added in future releases.

##### Examples:

###### Simple service with function factory

```go
import "github.com/bassbeaver/gioc"

type Service struct {
    Field1 string
    Field2 int
}

container := gioc.NewContainer()
container.RegisterServiceFactoryByAlias(
    "service",
    func() *Service {
        return &Service{Field1: "Field1", Field2: 5,}
    },
)

service := container.GetByAlias("service").(*Service)
```
In this example we registered and retrieved service by string alias.

###### Simple service with factory type of gioc.Factory 
```go
import "github.com/bassbeaver/gioc"

type Service struct {
    Field1 string
    Field2 int
}

container := gioc.NewContainer()
container.RegisterServiceFactoryByObject(
    (*Service)(nil),
    gioc.Factory {
        Create: func(f1 string, f2 int) *Service {
            return &Service{Field1: f1, Field2: f2,}
        },
        Arguments: []string{"field 1", "5"},
    },
)

service := container.GetByObject((*Service)(nil)).(*Service)

```

In this example we registered service by object (well, actually, pointer to object). 

###### Service with dependency, alternative alias and function factory

```go
import "github.com/bassbeaver/gioc"

type Service1 struct {
    Field1 string
    Field2 int
}
type Service2 struct {
    Field1 string
    ServiceInstance1 *Service1
}

container := gioc.NewContainer()
container.RegisterServiceFactoryByObject(
    (*Service1)(nil),
    func() *Service1 {
        return &Service1{Field1: "Field1-1", Field2: 5,}
    },
).RegisterServiceFactoryByObject(
    (*Service2)(nil),
    func(s1 *Service1) *Service2 {
        return &Service2{Field1: "Field2-1", ServiceInstance1: s1,}
    },
).AddServiceAliasByObject((*Service2)(nil), "service2")

service2 := container.GetByObject((*Service2)(nil)).(*Service2)
service2ByAlias := container.GetByAlias("service2").(*Service2)
``` 

In this example Container see that `Service2` depends on `Service1` (factory for `Service2` have argument of type `*Service1`)
and injects it into that factory.
Also, this example shows how to add alternative alias for service and get service by that alias. 

###### Service with dependency, factory type of gioc.Factory and multiple services of same type

```go
import "github.com/bassbeaver/gioc"

type Service1 struct {
    Field1 string
    Field2 int
}
type Service2 struct {
    Field1 string
    ServiceInstance1 *Service1
}

container := gioc.NewContainer()
container.RegisterServiceFactoryByObject(
    (*Service1)(nil),
    func() *Service1 {
        return &Service1{Field1: "Some value", Field2: 0,}
    },
).RegisterServiceFactoryByAlias(
    "anotherService1",
    func() *Service1 {
        return &Service1{Field1: "Field1-1", Field2: 5,}
    },
).RegisterServiceFactoryByObject(
    (*Service2)(nil),
    Factory {
        Create: func(f1 string, s1 *Service1) *Service2 {
            return &Service2{Field1: f1, ServiceInstance1: s1,}
        },
        Arguments: []string{"field 2-1", "@anotherService1"},
    },
)

service2 := container.GetByObject((*Service2)(nil)).(*Service2)
```

This example shows usage of `gioc.Factory` type as the service factory. You can see that two services
with type `Service1` was registered - they are different services and can be used separately from each other.
Factory for `Service2` configured to use second variant of `Service1` (registered with alias "anotherService1"),
so `Create` function of `Service2` factory will get `s1` as: `&Service1{Field1: "Field1-1", Field2: 5}`