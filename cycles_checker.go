package gioc

import (
	"fmt"
	"reflect"
)

type checkerNode struct {
	serviceAlias    string
	visited         bool
	dependenciesIds []int
}

// ---------------------------------------------------------------------------------------------------------------------

type checkerTable map[int]*checkerNode

func (t checkerTable) clearVisited() {
	for _, node := range t {
		node.visited = false
	}
}

func (t checkerTable) walkCheckerNode(node *checkerNode) bool {
	if node.visited {
		return false
	}
	node.visited = true

	for _, dependencyId := range node.dependenciesIds {
		dependencyNode := t[dependencyId]
		if !t.walkCheckerNode(dependencyNode) {
			return false
		}
	}

	return true
}

// ---------------------------------------------------------------------------------------------------------------------

func checkCyclesForContainer(c *Container) (bool, string) {
	var checker = make(checkerTable)

	// Building checker table. Checker table is indexed by registryEntry.id (which is unique for every unique service)
	// to avoid duplicate checks because of multiple aliases for one service
	for serviceAlias := range c.registry {
		registryElement := c.readRegistry(serviceAlias)
		if _, isInChecker := checker[registryElement.id]; !isInChecker {
			newCheckerNode := &checkerNode{
				serviceAlias:    serviceAlias,
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
					if "@" == argumentDefinition[:1] {
						argumentId = c.readRegistry(argumentDefinition[1:]).id
					}
					// If there is no argument data for current parameter - suppose that it is a service registered by object
				} else {
					argumentsRegistryElement := c.readRegistry(argumentType.String())
					if nil == argumentsRegistryElement {
						panic(
							fmt.Sprintf(
								"Failed to check dependencies cycles for service '%s'. Factory for service '%s' not registered",
								newCheckerNode.serviceAlias,
								argumentType.String(),
							),
						)
					}
					argumentId = argumentsRegistryElement.id
				}

				if argumentId >= 0 {
					newCheckerNode.dependenciesIds = append(newCheckerNode.dependenciesIds, argumentId)
				}
			}

			checker[registryElement.id] = newCheckerNode
		}
	}

	// Searching cycles
	for _, currentNode := range checker {
		checker.clearVisited()

		noCycles := checker.walkCheckerNode(currentNode)
		if !noCycles {
			return false, currentNode.serviceAlias
		}
	}

	return true, ""
}
