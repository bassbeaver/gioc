package gioc

import (
	"testing"
	"reflect"
)

func TestGetSimpleServiceByFactoryMethod(t *testing.T) {
	type Service struct {
		F1 string
		F2 int
	}

	serviceAlias := reflect.TypeOf((*Service)(nil)).String()

	c := NewContainer()
	c.RegisterServiceFactoryByAlias(
		serviceAlias,
		func() *Service {
			return &Service{F1: "Field1", F2: 5,}
		},
	)

	s := c.GetByAlias(serviceAlias).(*Service)

	referenceService := &Service{F1: "Field1", F2: 5,}
	if !reflect.DeepEqual(s, referenceService) {
		t.Errorf("Wrong service instanstiated. Wanted: %v. Instantiated: %v", referenceService, s)
	}
}

func TestGetSimpleServiceByFactory(t *testing.T) {
	type Service struct {
		F1 string
		F2 int
	}

	serviceAlias := reflect.TypeOf((*Service)(nil)).String()

	c := NewContainer()
	c.RegisterServiceFactoryByAlias(
		serviceAlias,
		Factory {
			Create: func(f1 string, f2 int) *Service {
				return &Service{F1: f1, F2: f2,}
			},
			Arguments: []string{"field 1", "6"},
		},
	)

	s := c.GetByAlias(serviceAlias).(*Service)
	referenceService := &Service{F1: "field 1", F2: 6,}
	if !reflect.DeepEqual(s, referenceService) {
		t.Errorf("Wrong service instanstiated. Wanted: %v. Instantiated: %v", referenceService, s)
	}
}

func TestGetDependentService(t *testing.T) {
	type Service1 struct {
		F1 string
		F2 int
	}
	type Service2 struct {
		F1 string
		S1 *Service1
	}

	c := NewContainer()
	c.RegisterServiceFactoryByObject(
		(*Service1)(nil),
		func() *Service1 {
			return &Service1{F1: "Field1-1", F2: 5,}
		},
	).RegisterServiceFactoryByObject(
		(*Service2)(nil),
		func(s1 *Service1) *Service2 {
			return &Service2{F1: "Field2-1", S1: s1,}
		},
	)

	s2 := c.GetByObject((*Service2)(nil)).(*Service2)

	referenceService2 := &Service2{F1: "Field2-1", S1: &Service1{F1: "Field1-1", F2: 5,},}
	if !reflect.DeepEqual(s2, referenceService2) {
		t.Errorf("Wrong service instanstiated. Wanted: %v. Instantiated: %v", referenceService2, s2)
	}
}

func TestGetDependentServiceByFactory(t *testing.T) {
	type Service1 struct {
		F1 string
		F2 int
	}
	type Service2 struct {
		F1 string
		S1 *Service1
	}

	c := NewContainer()
	c.RegisterServiceFactoryByAlias(
		"service1alias",
		func() *Service1 {
			return &Service1{F1: "Field1-1", F2: 5,}
		},
	).RegisterServiceFactoryByObject(
		(*Service1)(nil),
		func() *Service1 {
			return &Service1{F1: "Some value", F2: 0,}
		},
	).RegisterServiceFactoryByObject(
		(*Service2)(nil),
		Factory {
			Create: func(f1 string, s1 *Service1) *Service2 {
				return &Service2{F1: f1, S1: s1,}
			},
			Arguments: []string{"field 2-1", "@service1alias"},
		},
	).AddServiceAliasByObject(
		(*Service2)(nil),
		"service2alias",
	).AddServiceAlias(
		"service2alias",
		"service2alias2",
	)

	s2 := c.GetByObject((*Service2)(nil)).(*Service2)

	referenceService2 := &Service2{F1: "field 2-1", S1: &Service1{F1: "Field1-1", F2: 5,},}

	if !reflect.DeepEqual(s2, referenceService2) {
		t.Errorf("Wrong service instanstiated. Wanted: %v with S1: %v. Instantiated: %v with S1: %v", referenceService2, referenceService2.S1, s2, s2.S1)
	}

	s2 = c.GetByAlias("service2alias").(*Service2)
	if !reflect.DeepEqual(s2, referenceService2) {
		t.Errorf("Wrong service got by alias. Wanted: %v with S1: %v. Instantiated: %v with S1: %v", referenceService2, referenceService2.S1, s2, s2.S1)
	}

	s2 = c.GetByAlias("service2alias2").(*Service2)
	if !reflect.DeepEqual(s2, referenceService2) {
		t.Errorf("Wrong service got by second alias. Wanted: %v with S1: %v. Instantiated: %v with S1: %v", referenceService2, referenceService2.S1, s2, s2.S1)
	}
}