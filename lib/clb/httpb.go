package clb

import (
	"errors"
	"sync"

	"github.com/golang/glog"
	"google.golang.org/grpc/naming"
)

var ErrNoAddressAvailable = errors.New("No address available")
var ErrClientConnClosing = errors.New("Client connection closing")
var ErrBalancerClosed = errors.New("Balancer closed")

type addrInfo struct {
	addr      Address
	connected bool
}

// Address represents a server the client connects to.
// This is the EXPERIMENTAL API and may be changed or extended in the future.
type Address struct {
	// Addr is the server address on which a connection will be established.
	Addr string
	// Metadata is the information associated with Addr, which may be used
	// to make load balancing decision.
	Metadata interface{}
}

type httpRoundRobin struct {
	r     naming.Resolver
	w     naming.Watcher
	addrs []*addrInfo // all the addresses the client should potentially connect
	mu    sync.Mutex
	next  int  // index of the next address to return for Get()
	done  bool // The Balancer is closed.
}

type Balancer interface {
	Start(target string) error
	Get() (addr Address, err error)
	Close() error
}

func HttpRoundRobin(r naming.Resolver) Balancer {
	return &httpRoundRobin{r: r}
}

func (rr *httpRoundRobin) watchAddrUpdates() error {
	updates, err := rr.w.Next()
	if err != nil {
		glog.Warningf("http: the naming watcher stops working due to %v.", err)
		return err
	}
	rr.mu.Lock()
	defer rr.mu.Unlock()
	for _, update := range updates {
		addr := Address{
			Addr:     update.Addr,
			Metadata: update.Metadata,
		}
		switch update.Op {
		case naming.Add:
			var exist bool
			for _, v := range rr.addrs {
				if addr == v.addr {
					exist = true
					glog.Infoln("http: The name resolver wanted to add an existing address: ", addr)
					break
				}
			}
			if exist {
				continue
			}
			rr.addrs = append(rr.addrs, &addrInfo{addr: addr})
		case naming.Delete:
			for i, v := range rr.addrs {
				if addr == v.addr {
					copy(rr.addrs[i:], rr.addrs[i+1:])
					rr.addrs = rr.addrs[:len(rr.addrs)-1]
					break
				}
			}
		default:
			glog.Errorln("Unknown update.Op ", update.Op)
		}
	}
	// Make a copy of rr.addrs and write it onto rr.addrCh so that gRPC internals gets notified.
	if rr.done {
		return ErrClientConnClosing
	}
	return nil
}

func (rr *httpRoundRobin) Start(target string) error {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	if rr.done {
		return ErrClientConnClosing
	}
	if rr.r == nil {
		// If there is no name resolver installed, it is not needed to
		// do name resolution. In this case, target is added into rr.addrs
		// as the only address available and rr.addrCh stays nil.
		rr.addrs = append(rr.addrs, &addrInfo{addr: Address{Addr: target}})
		return nil
	}
	w, err := rr.r.Resolve(target)
	if err != nil {
		return err
	}
	rr.w = w
	go func() {
		for {
			if err := rr.watchAddrUpdates(); err != nil {
				return
			}
		}
	}()
	return nil
}

// Get returns the next addr in the rotation.
func (rr *httpRoundRobin) Get() (addr Address, err error) {
	rr.mu.Lock()
	if rr.done {
		rr.mu.Unlock()
		err = ErrClientConnClosing
		return
	}

	if len(rr.addrs) > 0 {
		if rr.next >= len(rr.addrs) {
			rr.next = 0
		}
		next := rr.next
		a := rr.addrs[next]
		next = (next + 1) % len(rr.addrs)
		addr = a.addr
		rr.next = next
	} else {
		addr = Address{}
		err = ErrNoAddressAvailable
	}
	rr.mu.Unlock()

	return addr, err
}

func (rr *httpRoundRobin) Close() error {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	if rr.done {
		return ErrBalancerClosed
	}
	rr.done = true
	if rr.w != nil {
		rr.w.Close()
	}
	return nil
}
