package clb

import (
	"fmt"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/naming"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type Clb struct {
	selector   string
	namespace  string
	portName   string
	clientset  *kubernetes.Clientset
	indexer    cache.Indexer
	controller cache.Controller
	updates    map[string](chan []*naming.Update)
}

func NewClb(cs *kubernetes.Clientset, apps []string, ns string) *Clb {
	clb := &Clb{
		selector:  fmt.Sprintf("app in (%s)", strings.Join(apps, ",")),
		namespace: ns,
		portName:  "grpc",
		clientset: cs,
		updates:   make(map[string](chan []*naming.Update)),
	}
	for _, v := range apps {
		clb.updates[v] = make(chan []*naming.Update, 1)
	}

	clb.watchPods()
	return clb
}

type watcher struct {
	target string
	clb    *Clb
}

func (clb *Clb) Resolve(target string) (naming.Watcher, error) {
	if _, ok := clb.updates[target]; !ok {
		return nil, fmt.Errorf("app not registered in clb")
	}
	return watcher{target: target, clb: clb}, nil
}

func (w watcher) Next() ([]*naming.Update, error) {
	u, ok := <-w.clb.updates[w.target]
	if ok {
		return u, nil
	}

	return nil, fmt.Errorf("watcher closed")
}
func (w watcher) Close() {
	close(w.clb.updates[w.target])
}

func (clb *Clb) RoundRobinClientConn(target string) *grpc.ClientConn {
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithBalancer(grpc.RoundRobin(clb)),
	}

	conn, err := grpc.Dial(target, opts...)
	if err != nil {
		panic(err.Error())
	}
	return conn
}
