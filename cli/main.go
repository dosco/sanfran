package main

import (
	"archive/zip"
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang/glog"
	fnapi "gitlab.com/dosco/sanfran/fnapi/rpc"
	"google.golang.org/grpc"
)

func main() {
	var action, name, fileName, host string

	crudCommand := flag.NewFlagSet("crud", flag.ExitOnError)
	listCommand := flag.NewFlagSet("list", flag.ExitOnError)

	listCommand.StringVar(&host, "host", "",
		"host:port of the sanfran-fnapi service")
	crudCommand.StringVar(&host, "host", "",
		"host:port of the sanfran-fnapi service")
	crudCommand.StringVar(&fileName, "file", "",
		"source file of the javascript function module")

	if len(os.Args) < 2 {
		fmt.Println("You need to specify an action (create, update, delete, list)")
		os.Exit(1)
	}
	action = os.Args[1]

	switch action {
	case "create", "update", "delete":
		if len(os.Args) < 3 {
			fmt.Println("You need to specify a function name")
			os.Exit(1)
		}
		name = os.Args[2]
		crudCommand.Parse(os.Args[3:])
	case "list":
		listCommand.Parse(os.Args[2:])
	default:
		flag.PrintDefaults()
		os.Exit(1)
	}

	flag.Parse()
	defer glog.Flush()

	if len(host) == 0 {
		fmt.Println("You need to specify the service host")
		os.Exit(1)
	}

	conn, err := grpc.Dial(fmt.Sprintf("%s:80", host), grpc.WithInsecure())
	if err != nil {
		glog.Fatalln(err.Error())
	}
	defer conn.Close()
	fnapiClient := fnapi.NewFnAPIClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if action == "create" || action == "update" {
		if len(fileName) == 0 {
			fmt.Println("You need to specify the function source filename")
			os.Exit(1)
		}
		filesToZip, err := newNodePackage(fileName)
		if err != nil {
			glog.Fatalln(err.Error())
		}

		baseDir := filepath.Dir(fileName) + "/"
		zipdCode, err := buildNodePackage(baseDir, filesToZip)
		if err != nil {
			glog.Fatalln(err.Error())
		}

		fn := fnapi.Function{
			Name:    name,
			Lang:    "js",
			Code:    zipdCode,
			Package: true,
		}

		if action == "create" {
			req := fnapi.CreateReq{Function: &fn}
			_, err = fnapiClient.Create(ctx, &req)
		}

		if action == "update" {
			req := fnapi.UpdateReq{Function: &fn}
			_, err = fnapiClient.Update(ctx, &req)
		}

		if err != nil {
			glog.Fatalln(err.Error())
		}

		fmt.Printf("> http://sanfran-routing-service/fn/%s\n", name)
	}

	glog.Flush()
}

func newNodePackage(fileName string) ([]string, error) {
	fileList := []string{fileName}

	nodeModules := filepath.Dir(fileName) + "/node_modules"

	err := filepath.Walk(nodeModules, func(path string, f os.FileInfo, err error) error {
		fileList = append(fileList, path)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return fileList, nil
}

func buildNodePackage(baseDir string, files []string) ([]byte, error) {
	var b bytes.Buffer

	zw := zip.NewWriter(&b)

	// Add files to zip
	for i, file := range files {
		f, err := os.Open(file)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		// Get the file information
		info, err := f.Stat()
		if err != nil {
			return nil, err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return nil, err
		}
		header.Name = strings.TrimPrefix(file, baseDir)
		header.Method = zip.Store

		if !info.IsDir() {
			header.Method = zip.Deflate
			if i == 0 {
				header.Name = "function.js"
			}
		} else {
			header.Name += "/"
		}

		w, err := zw.CreateHeader(header)
		if err != nil {
			return nil, err
		}

		if !info.IsDir() {
			if _, err = io.Copy(w, f); err != nil {
				return nil, err
			}
		}
		f.Close()
	}
	zw.Close()

	return b.Bytes(), nil
}
