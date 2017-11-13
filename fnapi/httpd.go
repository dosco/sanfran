package main

import (
	"net/http"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
)

func fetchCode(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	//TODO: Need version here
	name := ps.ByName("name")

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
