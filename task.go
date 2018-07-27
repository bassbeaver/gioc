package gioc

import (
	"sync"
)

type taskManager struct {
	addTaskChan chan *taskDefinition
	removeTaskChan chan *taskDefinition
	stopServeChan chan bool

	onServeMutex sync.Mutex
	onServe bool
}

func (tm *taskManager) addTask(task *taskDefinition) {
	tm.addTaskChan <- task
}

func (tm *taskManager) serve() {
	tm.onServeMutex.Lock()
	defer tm.onServeMutex.Unlock()

	if !tm.onServe {
		return
	}

	tm.onServe = true

	go func() {
		runningTasksListeners := newRunningTasksListenersMap()

		for {
			select {
			case newTaskDef := <- tm.addTaskChan:
				// If this task manager was told to stop serving - do not process new task
				if !tm.onServe {
					continue
				}

				// If such task is already running - just add new listener to running task
				if _, taskAlreadyRunning := runningTasksListeners.get(newTaskDef.taskName); taskAlreadyRunning {
					runningTasksListeners.append(newTaskDef.taskName, newTaskDef.listener)
					continue
				}

				// Run task
				runningTasksListeners.append(newTaskDef.taskName, newTaskDef.listener)
				go func() {
					result := newTaskDef.perform()
					listeners, _ := runningTasksListeners.get(newTaskDef.taskName)
					for _, listener := range listeners {
						listener <- result
					}
					tm.removeTaskChan <- newTaskDef
				}()
			case processedTask := <- tm.removeTaskChan:
				runningTasksListeners.delete(processedTask.taskName)
			case <- tm.stopServeChan:
				// This stopServeChan channel used to stop this goroutine after stopServe() call if no task are running
				tm.onServe = false
			}

			if !tm.onServe && runningTasksListeners.len() == 0 {
				return
			}
		}
	}()
}

func (tm *taskManager) stopServe() {
	tm.stopServeChan <- true
}

type taskDefinition struct {
	taskName string
	listener chan interface{}
	perform func() interface{}
}

type runningTasksListenersMap struct {
	runningTasks map[string][]chan interface{}
	mutex sync.RWMutex
}

func (m *runningTasksListenersMap) get(key string) ([]chan interface{}, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	val, isSet := m.runningTasks[key]
	return val, isSet
}

func (m *runningTasksListenersMap) append(key string, val chan interface{}) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, keyIsPresent := m.runningTasks[key]; !keyIsPresent {
		m.runningTasks[key] = make([]chan interface{}, 0)
	}

	m.runningTasks[key] = append(m.runningTasks[key], val)
}

func (m *runningTasksListenersMap) len() int {
	return len(m.runningTasks)
}

func (m *runningTasksListenersMap) delete(key string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	delete(m.runningTasks, key)
}

// --------------------------------------------

func newRunningTasksListenersMap() *runningTasksListenersMap {
	return &runningTasksListenersMap{
		runningTasks: make(map[string][]chan interface{}, 0),
	}
}

func newTaskManager() *taskManager {
	return &taskManager{
		addTaskChan: make(chan *taskDefinition),
		removeTaskChan: make(chan *taskDefinition),
		stopServeChan: make(chan bool),
		onServe: true,
	}
}