package main

import (
	"fmt"
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

func getControllerName() string {
	return getEnv("SANFRAN_CONTROLLER_NAME", true)
}

func getControllerUID() types.UID {
	v := getEnv("SANFRAN_CONTROLLER_UID", true)
	return types.UID(v)
}

func getFnLangImage() string {
	return getEnv("SANFRAN_FN_LANG_IMAGE", true)
}

func getSidecarImage() string {
	return getEnv("SANFRAN_SIDECAR_IMAGE", true)
}

func readConfig(fn string) string {
	if s, err := ioutil.ReadFile(fn); err != nil {
		glog.Errorf("%s, %s", fn, err.Error())
	} else {
		return string(s)
	}
	return ""
}
