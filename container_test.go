package gioc

import (
	"testing"
	"reflect"
	"time"
	"sync"
	"math/rand"
	"fmt"
)

func TestGetSimpleServiceByFactoryMethod(t *testing.T) {
	type Service struct {
		F1 string
		F2 int
	}

	serviceAlias := reflect.TypeOf((*Service)(nil)).String()

	c := NewContainer()
	defer c.Close()
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
	defer c.Close()
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
	defer c.Close()
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
		F2 int
	}

	c := NewContainer()
	defer c.Close()
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
	)

	var aliasAdded bool

	aliasAdded = c.AddServiceAliasByObject((*Service2)(nil), "service2alias")
	if !aliasAdded {
		t.Errorf("Failed to add service alias by object")
		return
	}

	aliasAdded = c.AddServiceAlias("service2alias", "service2alias2")
	if !aliasAdded {
		t.Errorf("Failed to add service alias")
		return
	}

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

func TestGetByAlias(t *testing.T) {
	type Service1 struct {
		F1 int
	}

	factoryCalled := 0

	c := NewContainer()
	defer c.Close()
	c.RegisterServiceFactoryByObject(
		(*Service1)(nil),
		func() *Service1 {
			factoryCalled++
			return &Service1{F1: 1,}
		},
	)

	var aliasAdded bool

	aliasAdded = c.AddServiceAliasByObject((*Service1)(nil), "service1alias")
	if !aliasAdded {
		t.Errorf("Failed to add service alias by object")
		return
	}

	aliasAdded = c.AddServiceAlias("service1alias", "service1alias2")
	if !aliasAdded {
		t.Errorf("Failed to add service alias")
		return
	}


	gotByObj := c.GetByObject((*Service1)(nil)).(*Service1)
	gotByAlias1 := c.GetByAlias("service1alias").(*Service1)
	gotByAlias2 := c.GetByAlias("service1alias2").(*Service1)

	gotByObj.F1 = 2
	if gotByObj.F1 != gotByAlias1.F1 || gotByObj.F1 != gotByAlias2.F1 || gotByAlias1.F1 != gotByAlias2.F1 || factoryCalled != 1 {
		t.Errorf(
			"Get by alias instantiated different instances. \n"+
			" gotByObj = %+v \n"+
			" gotByAlias1 = %+v \n"+
			" gotByAlias2 = %+v \n"+
			" Factory called %d times",
			gotByObj,
			gotByAlias1,
			gotByAlias2,
			factoryCalled,
		)
	}
}

func TestConcurrentTreeDependency(t *testing.T) {
	type Service1 struct {
		F1 string
		F2 int
	}
	type Service2 struct {
		F1 string
		S1 *Service1
	}
	type Service3 struct {
		F1 string
		S1 *Service1
	}

	factoriesRunsCount := make(map[string]int, 0)
	factoriesRunsChan := make(chan string)
	stopChan := make(chan bool)
	go func() {
		for{
			select {
			case factoryName := <- factoriesRunsChan:
				factoriesRunsCount[factoryName]++
			case <- stopChan:
				return
			}
		}
	}()

	c := NewContainer()
	defer c.Close()
	c.RegisterServiceFactoryByObject(
		(*Service1)(nil),
		func() *Service1 {
			// Timer is needed to make other dependent factories (for Service2 and Service3) to wait for current task to complete
			timer1 := time.NewTimer(1 * time.Second)
			<- timer1.C
			factoriesRunsChan <- "service1"
			return &Service1{F1: "Field1-1", F2: 5,}
		},
	).RegisterServiceFactoryByObject(
		(*Service2)(nil),
		func(s1 *Service1) *Service2 {
			factoriesRunsChan <- "service2"
			return &Service2{F1: "Field2-1", S1: s1,}
		},
	).RegisterServiceFactoryByObject(
		(*Service3)(nil),
		func(s1 *Service1) *Service3 {
			factoriesRunsChan <- "service3"
			return &Service3{F1: "Field3-1", S1: s1,}
		},
	)

	var s2 *Service2
	var s3 *Service3
	wg := new(sync.WaitGroup)
	wg.Add(2)
	go func() {
		s2 = c.GetByObject((*Service2)(nil)).(*Service2)
		wg.Done()
	}()
	go func() {
		s3 = c.GetByObject((*Service3)(nil)).(*Service3)
		wg.Done()
	}()
	wg.Wait()
	stopChan <- true

	for factory, runs := range factoriesRunsCount {
		if runs != 1 {
			t.Errorf("Factory %s was run %d times", factory, runs)
		}
	}

	referenceService2 := &Service2{F1: "Field2-1", S1: &Service1{F1: "Field1-1", F2: 5,},}
	referenceService3 := &Service3{F1: "Field3-1", S1: &Service1{F1: "Field1-1", F2: 5,},}
	if !reflect.DeepEqual(s2, referenceService2) {
		t.Errorf("Wrong service2 instanstiated.\nWanted:\t%#v \nInstantiated:\t%#v", referenceService2, s2)
	}
	if !reflect.DeepEqual(s3, referenceService3) {
		t.Errorf("Wrong service3 instanstiated.\nWanted:\t\t\t%#v \nInstantiated:\t%#v", referenceService3, s3)
	}
}

func TestHighloadConcurrentTreeDependency(t *testing.T) {
	type Service1 struct {
		F1 string
		F2 int
	}
	type Service2 struct {
		F1 string
		S1 *Service1
	}
	type Service3 struct {
		F1 string
		S1 *Service1
	}

	factoriesRunsCount := make(map[string]int)
	servicesGetCount := make(map[string]int)
	factoriesRunsChan := make(chan string)
	servicesGetChan := make(chan string)
	stopChan := make(chan bool)
	go func() {
		for{
			select {
			case factoryName := <- factoriesRunsChan:
				factoriesRunsCount[factoryName]++
			case serviceName := <- servicesGetChan:
				servicesGetCount[serviceName]++
			case <- stopChan:
				return
			}
		}
	}()

	c := NewContainer()
	defer c.Close()
	c.RegisterServiceFactoryByObject(
		(*Service1)(nil),
		Factory {
			Create: func(f1 string, f2 int) *Service1 {
				factoriesRunsChan <- "service1"
				<- time.NewTimer(1 * time.Second).C
				return &Service1{F1: f1, F2: f2,}
			},
			Arguments: []string{"field 1-1", "123"},
		},
	).RegisterServiceFactoryByObject(
		(*Service2)(nil),
		Factory {
			Create: func(f1 string, s1 *Service1) *Service2 {
				factoriesRunsChan <- "service2"
				return &Service2{F1: f1, S1: s1,}
			},
			Arguments: []string{"field 2-1",},
		},
	).RegisterServiceFactoryByObject(
		(*Service3)(nil),
		Factory {
			Create: func(f1 string, s1 *Service1) *Service3 {
				factoriesRunsChan <- "service3"
				return &Service3{F1: f1, S1: s1,}
			},
			Arguments: []string{"field 3-1",},
		},
	)

	referenceService1 := &Service1{F1: "field 1-1", F2: 123,}
	referenceService2 := &Service2{F1: "field 2-1", S1: &Service1{F1: "field 1-1", F2: 123,},}
	referenceService3 := &Service3{F1: "field 3-1", S1: &Service1{F1: "field 1-1", F2: 123,},}

	wg := new(sync.WaitGroup)
	for i:=0; i < 500; i++ {
		wg.Add(1)
		go func() {
			rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
			dice := rnd.Intn(100)
			if dice < 45 {
				servicesGetChan <- "service3"
				s3 := c.GetByObject((*Service3)(nil))
				if !reflect.DeepEqual(s3, referenceService3) {
					t.Errorf("Wrong service3 instanstiated.\nWanted:\t\t\t%#v \nInstantiated:\t%#v", referenceService3, s3)
				}
			} else if 45 <= dice && dice < 90 {
				servicesGetChan <- "service2"
				s2 := c.GetByObject((*Service2)(nil))
				if !reflect.DeepEqual(s2, referenceService2) {
					t.Errorf("Wrong service2 instanstiated.\nWanted:\t\t\t%#v \nInstantiated:\t%#v", referenceService2, s2)
				}
			} else {
				servicesGetChan <- "service1"
				s1 := c.GetByObject((*Service1)(nil))
				if !reflect.DeepEqual(s1, referenceService1) {
					t.Errorf("Wrong service1 instanstiated.\nWanted:\t\t\t%#v \nInstantiated:\t%#v", referenceService1, s1)
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
	stopChan <- true

	t.Logf(
		"Factories called: \n%s",
		func() string {
			result := ""
			for f, c := range factoriesRunsCount{
				result += fmt.Sprintf("  %s: %d\n", f, c)
			}
			return result
		}(),
	)

	t.Logf(
		"Services get: \n%s",
		func() string {
			result := ""
			for f, c := range servicesGetCount{
				result += fmt.Sprintf("  %s: %d\n", f, c)
			}
			return result
		}(),
	)
}