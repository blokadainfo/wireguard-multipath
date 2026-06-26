package routine

import (
	"maps"
	"sync"
)

type RoutineMap struct {
	m map[string]*Routine
	l sync.RWMutex
}

func NewRoutineMap() *RoutineMap {
	return &RoutineMap{
		m: make(map[string]*Routine),
		l: sync.RWMutex{},
	}
}

func (rm *RoutineMap) GetRoutines() map[string]*Routine {
	rm.l.RLock()
	defer rm.l.RUnlock()

	m := make(map[string]*Routine, len(rm.m))
	maps.Copy(m, rm.m)
	return m
}

func (rm *RoutineMap) GetRoutine(ifname string) (*Routine, bool) {
	rm.l.RLock()
	defer rm.l.RUnlock()

	r, ok := rm.m[ifname]
	return r, ok
}

func (rm *RoutineMap) SetRoutine(ifname string, rtn *Routine) error {
	rm.l.Lock()
	defer rm.l.Unlock()

	var err error
	if r, ok := rm.m[ifname]; ok {
		err = r.Close()
	}

	rm.m[ifname] = rtn
	return err
}

func (rm *RoutineMap) DelRoutine(ifname string) (bool, error) {
	rm.l.Lock()
	defer rm.l.Unlock()

	if r, ok := rm.m[ifname]; ok {
		err := r.Close()
		delete(rm.m, ifname)
		return true, err
	}

	return false, nil
}
