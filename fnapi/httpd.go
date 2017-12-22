package main

import (
	"net/http"
	"strings"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
)

func fetchCode(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	//TODO: Need version here
	nameParam := ps.ByName("name")

	if len(nameParam) == 0 {
		http.NotFound(w, r)
		return
	}
	p := strings.Split(nameParam, ".")
	name := p[0]

	fn, err := ds.GetFn(name)
	if err != nil {
		httpError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Transfer-Encoding", "binary")

	if _, err := w.Write(fn.GetCode()); err != nil {
		httpError(w, err)
		return
	}
}

func httpError(w http.ResponseWriter, err error) {
	glog.Errorln(err.Error())
	http.Error(w, err.Error(), 500)
}
