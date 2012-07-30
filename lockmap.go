package wedge

import "sync"

type lockMap struct {
	sync.Mutex
	safe map[interface{}]interface{}
}

type lockJob struct {
	key   interface{}
	value interface{}
}

func NewLockMap() *lockMap {
	m := &lockMap{}
	(*m).safe = make(map[interface{}]interface{})
	return m
}

func (m *lockMap) Insert(key, value interface{}) bool {
	m.Lock()
	defer m.Unlock()
	m.safe[key] = value
	return true
}

func (m *lockMap) Find(key interface{}) (interface{}, bool) {
	m.Lock()
	defer m.Unlock()
	val, ok := m.safe[key]

	return val, ok
}
