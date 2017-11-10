package main

import (
	"io/ioutil"
	"os"
	"strconv"

	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	poolSizeConfig = "/etc/sanfran-config/controller.poolsize"
)

func getPoolSize(defaultValue int) int {
	i, err := strconv.Atoi(readConfig(poolSizeConfig))
	if err != nil {
		glog.Errorf("%s, %s", poolSizeConfig, err.Error())
		return defaultValue
	}
	return i
}

func getNamespace() string {
	if ns := os.Getenv("SANFRAN_NAMESPACE"); len(ns) != 0 {
		return ns
	}
	return v1.NamespaceDefault
}

func getControllerName() string {
	if ns := os.Getenv("SANFRAN_CONTROLLER_NAME"); len(ns) != 0 {
		return ns
	}
	return "sanfran-controller"
}

func getControllerUID() types.UID {
	if ns := os.Getenv("SANFRAN_CONTROLLER_UID"); len(ns) != 0 {
		return types.UID(ns)
	}
	return types.UID("no-uid-found")
}

func readConfig(fn string) string {
	if s, err := ioutil.ReadFile(fn); err != nil {
		glog.Errorf("%s, %s", fn, err.Error())
	} else {
		return string(s)
	}
	return ""
}
