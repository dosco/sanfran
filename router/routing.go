package main

import (
	"sync"
	"time"
)

type Routes struct {
	funcMap map[string]*FnRoutes
	mux     sync.Mutex
}

func NewRoutes() Routes {
	return Routes{funcMap: make(map[string]*FnRoutes)}
}

func (r *Routes) AddRoute(name string, version int64, hostIP string) {
	r.mux.Lock()
	if fr, ok := r.funcMap[name]; !ok {
		r.funcMap[name] = NewFnRoutes(name, version, hostIP)
	} else if fr.GetVersion() < version {
		r.funcMap[name] = NewFnRoutes(name, version, hostIP)
	} else if fr.GetVersion() == version {
		fr.AddHost(hostIP)
	}
	r.mux.Unlock()
}

func (r *Routes) DeleteRoute(name string, version int64, hostIP string) {
	r.mux.Lock()
	if fr, ok := r.funcMap[name]; ok && fr.GetVersion() == version {
		fr.DeleteHost(hostIP)

		if fr.IsEmpty() {
			delete(r.funcMap, name)
		}
	}
	r.mux.Unlock()
}

func (r *Routes) GetRoute(name string, version int64) (string, bool) {
	if fr, ok := r.funcMap[name]; ok && fr.GetVersion() == version {
		fr.WaitForRoute(100 * time.Millisecond)
		return fr.GetHostIP()
	}
	return "", false
}

type FnRoutes struct {
	name     string
	version  int64
	hosts    []string
	hIndex   int
	reqCount int64
	mux      sync.Mutex
}

func NewFnRoutes(name string, version int64, hostIP string) *FnRoutes {
	r := FnRoutes{
		name:     name,
		version:  version,
		hosts:    []string{hostIP},
		hIndex:   0,
		reqCount: 0,
	}
	return &r
}

func (f *FnRoutes) AddHost(hostIP string) {
	f.mux.Lock()
	exists := false
	for i := 0; i < len(f.hosts); i++ {
		if exists = f.hosts[i] == hostIP; exists {
			break
		}
	}
	if !exists {
		f.hosts = append(f.hosts, hostIP)
		f.reqCount = 0
	}
	f.mux.Unlock()
}

func (f *FnRoutes) DeleteHost(hostIP string) {
	f.mux.Lock()
	for i := 0; i < len(f.hosts); i++ {
		if f.hosts[i] == hostIP {
			f.hosts = append(f.hosts[:i], f.hosts[i+1:]...)
		}
	}
	f.mux.Unlock()
}

func (f *FnRoutes) GetHostIP() (string, bool) {
	var hostIP string
	var ok bool

	f.mux.Lock()
	if len := len(f.hosts); len != 0 {
		hostIP, ok = f.hosts[f.hIndex], true
		f.reqCount = 0

		if f.hIndex++; f.hIndex >= len {
			f.hIndex = 0
		}
	} else {
		f.reqCount++
	}
	f.mux.Unlock()

	return hostIP, ok
}

func (f *FnRoutes) GetVersion() int64 {
	return f.version
}

func (f *FnRoutes) IsEmpty() bool {
	return len(f.hosts) == 0
}

func (f *FnRoutes) WaitForRoute(timeout time.Duration) {
	if len(f.hosts) != 0 || f.reqCount == 0 {
		return
	}

	t := time.NewTicker(timeout)
	for range t.C {
		if len(f.hosts) != 0 {
			break
		}
	}
	t.Stop()
}
