package main

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func newNodeJSPackage(codeFolder string, pkgJson, dotEnv bool) ([]string, error) {
	fileList := []string{path.Join(codeFolder, "function.js")}

	if dotEnv {
		fileList = append(fileList, path.Join(codeFolder, ".env"))
	}

	if pkgJson {
		fileList = append(fileList, path.Join(codeFolder, "package.json"))

		nodeModules := path.Join(codeFolder, "/node_modules")
		err := filepath.Walk(nodeModules, func(p string, f os.FileInfo, err error) error {
			fileList = append(fileList, p)
			return nil
		})

		if err != nil {
			return nil, err
		}
	}

	return fileList, nil
}

func buildNodeJSPackage(baseDir string, files []string) (*bytes.Buffer, error) {
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

	return &b, nil
}
