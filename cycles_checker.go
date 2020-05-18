package gioc

import (
	"container/list"
	"errors"
	"fmt"
	"reflect"
)

type checkerNode struct {
	serviceName     string
	visited         bool
	level			int
	dependenciesIds []int
}

// ---------------------------------------------------------------------------------------------------------------------

type dependencyChain struct {
	list.List
}

func (c *dependencyChain) Contains(needle string) bool {
	e := c.Front()

	for nil != e {
		if e.Value.(string) == needle {
			return true
		}

		e = e.Next()
	}

	return false
}

func (c *dependencyChain) String() string {
	var result string
	e := c.Front()

	for nil != e {
		if "" != result {
			result += "->"
		}

		result += e.Value.(string)

		e = e.Next()
	}

	return result
}

func NewDependencyChain() *dependencyChain {
	return &dependencyChain{List: *list.New()}
}

// ---------------------------------------------------------------------------------------------------------------------

type checkerTable map[int]*checkerNode

func (t checkerTable) clearVisited() {
	for _, node := range t {
		node.visited = false
		node.level = 0
	}
}

func (t checkerTable) walkCheckerNode(node *checkerNode, path string) string {
	if "" != path {
		path += "->"
	}
	path += node.serviceName

	if node.visited {
		return path
	}

	// If current node is leaf (dependency without it's own dependencies) it should not be marked as visited
	// because current subtree ended and current leaf does not create loop.
	if len(node.dependenciesIds) > 0 {
		node.visited = true
	}

	for _, dependencyId := range node.dependenciesIds {
		dependencyNode := t[dependencyId]
		if loopedPath := t.walkCheckerNode(dependencyNode, path); "" != loopedPath {
			return loopedPath
		}
	}

	return ""
}

// ---------------------------------------------------------------------------------------------------------------------

func checkCyclesForContainer(c *Container) (bool, string) {
	var checker = make(checkerTable)

	// Building checker table. Checker table is indexed by registryEntry.id (which is unique for every unique service)
	// to avoid duplicate checks because of multiple aliases for one service

	for serviceAlias := range c.registry.aliasIndex {
		registryElement := c.registry.readAlias(serviceAlias)
		if _, isInChecker := checker[registryElement.id]; !isInChecker {
			newCheckerNode, checkerNodeError := createCheckerNode(c, registryElement)
			if nil != checkerNodeError {
				panic(
					fmt.Sprintf(
						"Failed to check dependencies cycles for service '%s'. Error: %s",
						serviceAlias,
						checkerNodeError,
					),
				)
			}

			newCheckerNode.serviceName = serviceAlias
			checker[registryElement.id] = newCheckerNode
		}
	}

	for serviceType := range c.registry.typeIndex {
		registryElement := c.registry.readType(serviceType)
		if _, isInChecker := checker[registryElement.id]; !isInChecker {
			serviceTypeName := serviceType.String()

			newCheckerNode, checkerNodeError := createCheckerNode(c, registryElement)
			if nil != checkerNodeError {
				panic(
					fmt.Sprintf(
						"Failed to check dependencies cycles for service with type %s. Error: %s",
						serviceTypeName,
						checkerNodeError,
					),
				)
			}

			newCheckerNode.serviceName = serviceTypeName
			checker[registryElement.id] = newCheckerNode
		}
	}

	// Searching cycles
	for _, currentNode := range checker {
		checker.clearVisited()

		loopedPath := checker.walkCheckerNode(currentNode, "")
		if "" != loopedPath {
			return false, loopedPath
		}
	}

	return true, ""
}

func createCheckerNode(c *Container, registryElement *registryEntry) (*checkerNode, error) {
	newCheckerNode := &checkerNode{
		visited:         false,
		dependenciesIds: make([]int, 0),
	}

	// Getting ids of dependencies
	factoryMethodValue := reflect.ValueOf(registryElement.factory.Create)

	factoryMethodType := factoryMethodValue.Type()
	for argumentNum := 0; argumentNum < factoryMethodType.NumIn(); argumentNum++ {
		argumentId := -1

		argumentType := factoryMethodType.In(argumentNum)

		// If there is argument data for current parameter - process it
		if argumentNum < len(registryElement.factory.Arguments) {
			argumentDefinition := registryElement.factory.Arguments[argumentNum]
			// Sign @ indicates that it is service alias
			if len(argumentDefinition) >= 1 && "@" == argumentDefinition[:1] {
				argumentId = c.registry.readAlias(argumentDefinition[1:]).id
			}
		} else {
			// If there is no argument data for current parameter - suppose that it is a service registered by object
			argumentsRegistryElement := c.registry.readType(argumentType)
			if nil == argumentsRegistryElement {
				return nil, errors.New("factory not found")
			}
			argumentId = argumentsRegistryElement.id
		}

		if argumentId >= 0 {
			newCheckerNode.dependenciesIds = append(newCheckerNode.dependenciesIds, argumentId)
		}
	}

	return newCheckerNode, nil
}
