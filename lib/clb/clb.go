package clb

import (
	"fmt"
	"strings"

	"github.com/golang/glog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/naming"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

const (
	appSel      = "app in (%s)"
	defaultPort = "grpc"
)

type Clb struct {
	selector   string
	namespace  string
	ports      map[string]string
	clientset  *kubernetes.Clientset
	indexer    cache.Indexer
	controller cache.Controller
	updates    map[string](chan []*naming.Update)
}

type Config struct {
	Namespace  string
	HostPrefix string
	Services   map[string]Service
}

type Service struct {
	Host string
	Port string
}

func (cfg Config) Get(name string) string {
	if v, ok := cfg.Services[name]; ok {
		if len(cfg.HostPrefix) != 0 {
			return strings.Join([]string{cfg.HostPrefix, v.Host}, "-")
		} else {
			return v.Host
		}
	}
	return ""
}

func NewClb(cs *kubernetes.Clientset, cfg Config) *Clb {
	var appNames []string

	clb := &Clb{
		clientset: cs,
		namespace: cfg.Namespace,
		ports:     make(map[string]string),
		updates:   make(map[string](chan []*naming.Update)),
	}

	for _, v := range cfg.Services {
		var name, port string

		if len(cfg.HostPrefix) != 0 {
			name = strings.Join([]string{cfg.HostPrefix, v.Host}, "-")
		} else {
			name = v.Host
		}

		if len(v.Port) != 0 {
			port = v.Port
		} else {
			port = defaultPort
		}

		clb.updates[name] = make(chan []*naming.Update, 1)
		clb.ports[name] = port
		appNames = append(appNames, name)
	}
	clb.selector = fmt.Sprintf(appSel, strings.Join(appNames, ","))
	glog.Infof("[clb] Services selector: %s\n", clb.selector)

	clb.watchPods()

	return clb
}

type watcher struct {
	target string
	clb    *Clb
}

func (clb *Clb) Resolve(target string) (naming.Watcher, error) {
	if _, ok := clb.updates[target]; !ok {
		return nil, fmt.Errorf("Not registered in clb: %s", target)
	}
	return watcher{target: target, clb: clb}, nil
}

func (w watcher) Next() ([]*naming.Update, error) {
	u, ok := <-w.clb.updates[w.target]
	if ok {
		return u, nil
	}

	return nil, fmt.Errorf("Watcher closed")
}

func (w watcher) Close() {
	close(w.clb.updates[w.target])
}

func (clb *Clb) ClientConn(target string) *grpc.ClientConn {
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
