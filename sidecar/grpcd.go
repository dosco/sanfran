package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/dosco/sanfran/sidecar/rpc"
	"github.com/golang/glog"
	"github.com/sethgrid/pester"
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

type server struct {
	lastReqTS time.Time
	terminate bool
	mux       sync.Mutex
}

const (
	appURLPrefix = "http://localhost:8081"
	funcPath     = "/shared/func"
)

func initServer(port int) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port)) // RPC port
	if err != nil {
		glog.Fatalf("failed to listen: %v", err)
	}
	g := grpc.NewServer()

	server := &server{lastReqTS: time.Now()}
	rpc.RegisterSidecarServer(g, server)

	glog.Infof("SanFran/Sidecar Service Listening on :%d\n", port)
	g.Serve(lis)
}

func getClient() *pester.Client {
	client := pester.New()
	client.Concurrency = 1
	client.MaxRetries = 3
	client.Backoff = pester.LinearJitterBackoff
	client.KeepLog = false
	return client
}

func (s *server) Activate(ctx context.Context, req *rpc.ActivateReq) (*rpc.ActivateResp, error) {
	var err error

	if s.terminate {
		return nil, fmt.Errorf("terminate = true")
	}

	if err := resetFuncFolder(funcPath); err != nil {
		return nil, err
	}

	if len(req.GetCode()) != 0 {
		err = activateFromCode(funcPath, req.GetCode())
	} else if len(req.GetLink()) != 0 {
		err = activateFromLink(funcPath, req.GetLink())
	} else {
		err = fmt.Errorf("No 'code' or 'link' specified")
	}

	if err != nil {
		s.terminate = true
		return nil, err
	}

	reqLink := fmt.Sprintf("%s/api/activate", appURLPrefix)
	httpResp, err := getClient().Get(reqLink)
	if err != nil {
		s.terminate = true
		return nil, err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode == 500 {
		s.terminate = true

		body, err := ioutil.ReadAll(httpResp.Body)
		if err != nil {
			return nil, err
		}
		return nil, errors.New(string(body))
	}

	s.mux.Lock()
	s.lastReqTS = time.Now()
	s.mux.Unlock()

	return &rpc.ActivateResp{}, nil
}

func (s *server) Execute(ctx context.Context, req *rpc.ExecuteReq) (*rpc.ExecuteResp, error) {
	if s.terminate {
		return nil, fmt.Errorf("terminate = true")
	}

	var reqLink string
	if len(req.Query) != 0 {
		reqLink = fmt.Sprintf("%s%s", appURLPrefix, buildQueryString(req))
	} else {
		reqLink = appURLPrefix
	}

	httpReq, err := http.NewRequest(req.Method, reqLink, bytes.NewReader(req.GetBody()))
	if err != nil {
		s.terminate = true
		return nil, err
	}

	httpReq.Form = url.Values{}
	for k, v := range req.GetQuery() {
		httpReq.Form[k] = v.Value
	}

	httpReq.Header = http.Header{}
	hdrs := req.GetHeader()
	for k, v := range hdrs {
		httpReq.Header[k] = v.Value
	}

	httpResp, err := getClient().Do(httpReq)
	if err != nil {
		s.terminate = true
		return nil, err
	}
	defer httpResp.Body.Close()

	body, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		s.terminate = true
		return nil, err
	}

	resp := rpc.ExecuteResp{
		Body: body,
	}

	resp.Header = make(map[string]*rpc.ListOfString)
	for k, v := range httpResp.Header {
		resp.Header[k] = &rpc.ListOfString{Value: v}
	}

	s.mux.Lock()
	s.lastReqTS = time.Now()
	s.mux.Unlock()

	return &resp, nil
}

type metrics struct {
	LoadAvg []float32 `json:"load_avg"`
	FreeMem float32   `json:"free_mem"`
}

func (s *server) Metrics(context.Context, *rpc.MetricsReq) (*rpc.MetricsResp, error) {
	if s.terminate {
		return &rpc.MetricsResp{Terminate: true}, nil
	}

	reqLink := fmt.Sprintf("%s/api/ping", appURLPrefix)

	httpResp, err := getClient().Get(reqLink)
	if err != nil {
		s.terminate = true
		return nil, err
	}
	defer httpResp.Body.Close()

	body, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		s.terminate = true
		return nil, err
	}

	var m metrics
	if err := json.Unmarshal(body, &m); err != nil {
		s.terminate = true
		return nil, err
	}

	s.mux.Lock()
	resp := &rpc.MetricsResp{
		LoadAvg:   m.LoadAvg,
		FreeMem:   m.FreeMem,
		LastReq:   time.Now().Sub(s.lastReqTS).Seconds(),
		Terminate: false,
	}
	s.mux.Unlock()

	return resp, nil
}

func buildQueryString(req *rpc.ExecuteReq) string {
	var qv []string

	for k, v := range req.Query {
		for i := range v.Value {
			qv = append(qv, fmt.Sprintf("%s=%s", k, v.Value[i]))
		}
	}

	if len(qv) != 0 {
		return "?" + strings.Join(qv, "&")
	}
	return ""
}
