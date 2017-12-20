package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/dosco/sanfran/builder/rpc"
	"github.com/dosco/sanfran/lib/clb"
	"github.com/go-cmd/cmd"
	"github.com/golang/glog"
	minio "github.com/minio/minio-go"
	"github.com/minio/minio-go/pkg/policy"
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
	"k8s.io/client-go/kubernetes"
)

const (
	inputPath   = "/data"
	storagePath = "/storage"
	bucketName  = "functions"
)

var (
	re = regexp.MustCompile(`(?im)(require|import)\((.+)\)`)
)

type server struct {
	fnstoreLB    clb.Balancer
	bucketExists bool
}

func initServer(clientset *kubernetes.Clientset, port int) {
	clbCfg := clb.Config{
		Namespace:  getNamespace(),
		HostPrefix: getHelmRelease(),
		Services: map[string]clb.Service{
			"fnstore": clb.Service{Host: "sf-fnstore", Port: "service"},
		},
	}
	lb := clb.NewClb(clientset, clbCfg)

	fnstoreLB := clb.HttpRoundRobin(lb)
	if err := fnstoreLB.Start(clbCfg.Get("fnstore")); err != nil {
		glog.Fatalln(err.Error())
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port)) // RPC port
	if err != nil {
		glog.Fatalf("failed to listen: %v", err)
	}
	g := grpc.NewServer()

	server := &server{fnstoreLB: fnstoreLB}
	rpc.RegisterBuilderServer(g, server)

	glog.Infof("SanFran/Builder Service Listening on :%d\n", port)
	g.Serve(lis)
}

func (s *server) Build(ctx context.Context, req *rpc.BuildReq) (*rpc.BuildResp, error) {
	var err error

	if err := os.Chdir(inputPath); err != nil {
		return nil, err
	}

	dir, err := ioutil.TempDir(inputPath, "builder")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	fnFile := filepath.Join(dir, "function.js")

	code := req.GetCode()
	fmode := os.FileMode(0777)

	if req.GetPackage() {
		_, err = unZip(bytes.NewReader(code), int64(len(code)), dir)
	} else {
		err = ioutil.WriteFile(fnFile, code, fmode)
	}

	if err != nil {
		return nil, err
	}

	packages := extractPackages(string(code))
	nodeModulesExists := len(packages) > 0

	if nodeModulesExists {
		f := filepath.Join(dir, "package.json")
		err = ioutil.WriteFile(f, packageJSON(), fmode)
		if err != nil {
			return nil, err
		}

		err = installImports(dir, packages)
		if err != nil {
			return nil, err
		}
	}

	dotEnvExists := len(req.GetVars()) > 0
	if dotEnvExists {
		var vars string
		for k, v := range req.GetVars() {
			vars += fmt.Sprintf("%s=%s\n", k, v)
		}

		f := filepath.Join(dir, ".env")
		err = ioutil.WriteFile(f, []byte(vars), fmode)
		if err != nil {
			return nil, err
		}
	}

	filesToZip, err := newNodeJSPackage(dir, nodeModulesExists, dotEnvExists)
	if err != nil {
		return nil, err
	}

	dirWithSlash := dir + "/"
	buf, err := buildNodeJSPackage(dirWithSlash, filesToZip)
	if err != nil {
		return nil, err
	}

	client, err := getFnstoreClient(s.fnstoreLB)
	if err != nil {
		return nil, err
	}

	if !s.bucketExists {
		s.bucketExists, err = createBucket(client, bucketName)
		if err != nil {
			return nil, err
		}
	}

	fileName := functionFilename(req.GetName(), "js", req.GetVersion())

	_, err = client.PutObject(
		bucketName,
		fileName,
		buf,
		int64(buf.Len()),
		minio.PutObjectOptions{ContentType: "application/zip"})

	if err != nil {
		return nil, err
	}

	return &rpc.BuildResp{}, nil
}

func packageJSON() []byte {
	packageJSON := `{
		"name": "function",
		"version": "1.0.0",
		"description": "",
		"dependencies": {}
	}`
	return []byte(packageJSON)
}

func installImports(installPrefix string, packages []string) error {
	cmdArgs := []string{
		"install",
		"--prefix",
		installPrefix,
		"--save",
		"--no-progress",
	}
	cmdArgs = append(cmdArgs, packages...)

	glog.Infof("Cmd: npm %s", strings.Join(cmdArgs, " "))

	// Start a long-running process, capture stdout and stderr
	c := cmd.NewCmd("npm", cmdArgs...)
	statusChan := c.Start()

	// Stop command after 1 hour
	go func() {
		<-time.After(5 * time.Minute)
		c.Stop()
	}()

	// Block waiting for command to exit, be stopped, or be killed
	finalStatus := <-statusChan

	if finalStatus.Exit == 1 {
		pkgs := strings.Join(packages, ", ")

		glog.Errorf("Error installing packages %s: %s",
			pkgs, strings.Join(finalStatus.Stderr, ","))

		return fmt.Errorf("Error installing npm packages: %s", pkgs)
	}
	return nil
}

func extractPackages(code string) []string {
	var values []string

	m := re.FindAllStringSubmatch(code, -1)
	for i := range m {
		values = append(values, strings.Trim(m[i][2], "'\" "))
	}
	return values
}

func functionFilename(name, lang string, version int64) string {
	return strings.Join([]string{
		fmt.Sprintf("%s-%d", name, version), lang, "zip"}, ".")
}

func getFnstoreClient(fnstoreLB clb.Balancer) (*minio.Client, error) {
	addr, err := fnstoreLB.Get()
	if err != nil {
		return nil, err
	}

	minioClient, err := minio.New(addr.Addr,
		getFnstoreAccessKey(), getFnstoreSecretKey(), false)
	if err != nil {
		return nil, err
	}

	return minioClient, nil
}

func createBucket(client *minio.Client, bn string) (bool, error) {
	if exists, err := client.BucketExists(bn); exists {
		return true, nil
	} else if err != nil {
		return false, err
	}

	if err := client.MakeBucket(bn, ""); err != nil {
		return false, err
	}

	err := client.SetBucketPolicy(bn, "", policy.BucketPolicyReadOnly)
	return (err == nil), err
}
