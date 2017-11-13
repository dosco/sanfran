package main

import (
	"errors"
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
	if v := os.Getenv("SANFRAN_NAMESPACE"); len(v) != 0 {
		return v
	}
	return v1.NamespaceDefault
}

func getControllerName() string {
	v := os.Getenv("SANFRAN_CONTROLLER_NAME")
	if len(v) == 0 {
		panic(errors.New("Env var SANFRAN_CONTROLLER_NAME not defined"))
	}
	return v
}

func getControllerUID() types.UID {
	v := os.Getenv("SANFRAN_CONTROLLER_UID")
	if len(v) == 0 {
		panic(errors.New("Env var SANFRAN_CONTROLLER_UID not defined"))
	}
	return types.UID(v)
}

func getFnLangImage() string {
	v := os.Getenv("SANFRAN_FN_LANG_IMAGE")
	if len(v) == 0 {
		panic(errors.New("Env var SANFRAN_FN_LANG_IMAGE not defined"))
	}
	return v
}

func getSidecarImage() string {
	v := os.Getenv("SANFRAN_SIDECAR_IMAGE")
	if len(v) == 0 {
		panic(errors.New("Env var SANFRAN_SIDECAR_IMAGE not defined"))
	}
	return v
}

func readConfig(fn string) string {
	if s, err := ioutil.ReadFile(fn); err != nil {
		glog.Errorf("%s, %s", fn, err.Error())
	} else {
		return string(s)
	}
	return ""
}
