package main

import (
	"sync"
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
	}
	r.mux.Unlock()
}

func (r *Routes) GetRoute(name string, version int64) (string, bool) {
	r.mux.Lock()
	defer r.mux.Unlock()

	if fr, ok := r.funcMap[name]; ok && fr.GetVersion() == version {
		return fr.GetHostIP()
	}
	return "", false
}

type FnRoutes struct {
	name    string
	version int64
	hosts   []string
	hIndex  int
}

func NewFnRoutes(name string, version int64, hostIP string) *FnRoutes {
	r := FnRoutes{
		name:    name,
		version: version,
		hosts:   []string{hostIP},
		hIndex:  0,
	}
	return &r
}

func (f *FnRoutes) AddHost(hostIP string) {
	for i := 0; i < len(f.hosts); i++ {
		if f.hosts[i] == hostIP {
			return
		}
	}

	f.hosts = append(f.hosts, hostIP)
}

func (f *FnRoutes) DeleteHost(hostIP string) {
	for i := 0; i < len(f.hosts); i++ {
		if f.hosts[i] == hostIP {
			f.hosts = append(f.hosts[:i], f.hosts[i+1:]...)
		}
	}
}

func (f *FnRoutes) GetHostIP() (string, bool) {
	len := len(f.hosts)
	if len == 0 {
		return "", false
	}

	hostIP := f.hosts[f.hIndex]
	if f.hIndex++; f.hIndex >= len {
		f.hIndex = 0
	}
	return hostIP, true
}

func (f *FnRoutes) GetVersion() int64 {
	return f.version
}
