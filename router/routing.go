package main

import (
	"net"
	"sync"
	"time"

	"github.com/golang/glog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

type FnRoutes struct {
	sync.Mutex
	name    string
	version int64
	hosts   []string
	next    int
	wait    chan bool
}

type Routes struct {
	funcMapMux sync.Mutex
	funcMap    map[string]*FnRoutes

	connsMux sync.Mutex
	conns    map[string]*grpc.ClientConn
}

func NewRoutes() Routes {
	return Routes{
		funcMap: make(map[string]*FnRoutes),
		conns:   make(map[string]*grpc.ClientConn),
	}
}

func (r *Routes) AddRoute(name string, version int64, hostIP string) *grpc.ClientConn {
	r.funcMapMux.Lock()
	fr, ok := r.funcMap[name]
	if !ok {
		glog.Fatalf("Nil funcMap for function: %s\n", name)
	}
	r.funcMapMux.Unlock()

	if ok {
		fr.Lock()
		if fr.version == version {
			fr.addHost(hostIP)
		} else {
			fr.version = version
			fr.hosts = []string{hostIP}
			fr.next = 0

			if fr.wait != nil {
				close(fr.wait)
			}
		}

		fr.Unlock()
	}

	r.connsMux.Lock()
	conn, ok := r.testConn(hostIP)
	if !ok {
		conn, _ = r.setupConn(hostIP)
	}
	r.connsMux.Unlock()

	return conn
}

func (r *Routes) DeleteRoute(name string, version int64, hostIP string) {
	r.funcMapMux.Lock()
	fr, ok := r.funcMap[name]
	r.funcMapMux.Unlock()

	if !ok {
		return
	}

	fr.Lock()
	fr.deleteHost(hostIP)
	l := len(fr.hosts)
	fr.Unlock()

	if l == 0 {
		r.funcMapMux.Lock()
		delete(r.funcMap, name)
		r.funcMapMux.Unlock()
	}

	r.connsMux.Lock()
	if _, ok := r.testConn(hostIP); !ok {
		delete(r.conns, hostIP)
	}
	r.connsMux.Unlock()
}

func (r *Routes) GetConn(name string) (*grpc.ClientConn, bool) {
	r.funcMapMux.Lock()
	fr, ok := r.funcMap[name]

	if !ok {
		r.funcMap[name] = NewEmptyFnRoutes(name)
		r.funcMapMux.Unlock()
		return nil, false
	}
	r.funcMapMux.Unlock()

	if fr.wait != nil {
		select {
		case <-fr.wait:
		case <-time.After(300 * time.Millisecond):
			return nil, false
		}
	}

	fr.Lock()
	hostIP, ok := fr.getHostIP()
	fr.Unlock()

	if !ok {
		return nil, false
	}

	r.connsMux.Lock()
	conn, ok := r.testConn(hostIP)
	if !ok {
		conn, ok = r.setupConn(hostIP)
	}
	r.connsMux.Unlock()

	return conn, ok
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

func NewFnRoutes(name string, version int64, hostIP string) *FnRoutes {
	f := &FnRoutes{
		name:    name,
		version: version,
		hosts:   []string{hostIP},
		next:    0,
	}

	return f
}

func NewEmptyFnRoutes(name string) *FnRoutes {
	f := &FnRoutes{
		name: name,
		next: 0,
		wait: make(chan bool),
	}

	return f
}

func (f *FnRoutes) addHost(hostIP string) {
	var exists bool
	for i := 0; i < len(f.hosts); i++ {
		if exists = f.hosts[i] == hostIP; exists {
			break
		}
	}

	if !exists {
		f.hosts = append(f.hosts, hostIP)
	}
}

func (f *FnRoutes) deleteHost(hostIP string) {
	for i := 0; i < len(f.hosts); i++ {
		if f.hosts[i] == hostIP {
			f.hosts = append(f.hosts[:i], f.hosts[i+1:]...)
		}
	}
}

func (f *FnRoutes) getHostIP() (string, bool) {
	var addr string
	l := len(f.hosts)

	if l == 1 {
		f.next = 0
		addr = f.hosts[0]
	} else if l > 0 {
		if f.next >= len(f.hosts) {
			f.next = 0
		}
		next := f.next
		addr = f.hosts[next]
		next = (next + 1) % len(f.hosts)
		f.next = next
	}

	return addr, len(addr) > 0
}
