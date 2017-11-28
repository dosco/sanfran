package main

import (
	"fmt"
	"os"

	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
)

func getNamespace() string {
	if ns := os.Getenv("SANFRAN_NAMESPACE"); len(ns) != 0 {
		return ns
	} else {
		return v1.NamespaceDefault
	}
}

func getEnv(name string, required bool) string {
	if v := os.Getenv(name); len(v) != 0 {
		return v
	}
	if required {
		glog.Fatalln(fmt.Errorf("%s not defined", name))
	}
	return ""

}

func getHelmRelease() string {
	return getEnv("SANFRAN_HELM_RELEASE", true)
}
