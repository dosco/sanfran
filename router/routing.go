package main

import (
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"k8s.io/apimachinery/pkg/util/wait"
)

type Routes struct {
	funcMap map[string]*FnRoutes
	conns   map[string]*grpc.ClientConn
	mux     sync.Mutex
}

func NewRoutes() Routes {
	return Routes{
		funcMap: make(map[string]*FnRoutes),
		conns:   make(map[string]*grpc.ClientConn),
	}
}

func (r *Routes) AddRoute(name string, version int64, hostIP string) *grpc.ClientConn {
	r.mux.Lock()
	if fr, ok := r.funcMap[name]; !ok {
		r.funcMap[name] = NewFnRoutes(name, version, hostIP)
	} else if fr.GetVersion() < version {
		r.funcMap[name] = NewFnRoutes(name, version, hostIP)
	} else if fr.GetVersion() == version {
		fr.AddHost(hostIP)
	}

	conn, ok := r.testConn(hostIP)
	if !ok {
		r.setupConn(hostIP)
	}
	r.mux.Unlock()

	return conn
}

func (r *Routes) DeleteRoute(name string, version int64, hostIP string) {
	r.mux.Lock()
	if fr, ok := r.funcMap[name]; ok && fr.GetVersion() == version {
		fr.DeleteHost(hostIP)

		if fr.IsEmpty() {
			delete(r.funcMap, name)
		}

		if _, ok := r.testConn(hostIP); !ok {
			delete(r.conns, hostIP)
		}
	}
	r.mux.Unlock()
}

func (r *Routes) GetConn(name string, version int64) (*grpc.ClientConn, bool) {
	if f, ok := r.funcMap[name]; ok && f.GetVersion() == version {
		f.waitForRoute(5 * time.Millisecond)

		hostIP, ok := f.GetHostIP()
		if !ok {
			return nil, false
		}

		conn, ok := r.testConn(hostIP)
		if !ok {
			return r.setupConn(hostIP)
		}

		return conn, ok
	}

	return nil, false
}

func (r *Routes) testConn(hostIP string) (*grpc.ClientConn, bool) {
	if c, ok := r.conns[hostIP]; ok && c != nil {
		s := c.GetState()

		if s == connectivity.TransientFailure ||
			s == connectivity.Shutdown {
			c.Close()
			return nil, false
		}
		return c, true
	}
	return nil, false
}

func (r *Routes) setupConn(hostIP string) (*grpc.ClientConn, bool) {
	if c, ok := r.conns[hostIP]; ok && c != nil {
		c.Close()
	}

	hp := net.JoinHostPort(hostIP, "8080")
	conn, err := grpc.Dial(hp, grpc.WithInsecure())
	if err != nil {
		conn.Close()
		r.conns[hostIP] = nil
		return nil, false
	}
	r.conns[hostIP] = conn
	return conn, true
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
	f := &FnRoutes{
		name:     name,
		version:  version,
		hosts:    []string{hostIP},
		hIndex:   0,
		reqCount: 0,
	}

	return f
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
		if f.hIndex >= len {
			f.hIndex = 0
		}
		hostIP, ok = f.hosts[f.hIndex], true
		f.reqCount = 0
		f.hIndex++
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

func (f *FnRoutes) waitForRoute(tick time.Duration) {
	if len(f.hosts) != 0 || f.reqCount == 0 {
		return
	}

	wait.Poll(5*time.Millisecond, 100*time.Millisecond, func() (bool, error) {
		if len(f.hosts) != 0 {
			return true, nil
		}
		return false, nil
	})

}
