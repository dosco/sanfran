package main

import (
	"archive/zip"
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/sethgrid/pester"
)

func activateFromCode(funcPath, code string) error {
	out, err := os.Create(funcPath + "/function.js")
	if err != nil {
		return err
	}
	defer out.Close()
	out.WriteString(code)

	return nil
}

func activateFromLink(funcPath, link string) error {
	client := pester.New()
	client.Concurrency = 1
	client.MaxRetries = 3
	client.Backoff = pester.LinearJitterBackoff
	client.KeepLog = false

	resp, err := client.Get(link)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	r := bytes.NewReader(body)
	len := int64(len(body))
	if _, err := unZip(r, len, funcPath); err != nil {
		return err
	}

	return nil
}

func resetFuncFolder(funcPath string) error {
	if err := os.RemoveAll(funcPath); err != nil {
		return err
	}

	if err := os.Mkdir(funcPath, os.FileMode(0777)); err != nil {
		return err
	}

	return nil
}

func unZip(zipReader io.ReaderAt, size int64, dest string) ([]string, error) {
	var filenames []string

	r, err := zip.NewReader(zipReader, size)
	if err != nil {
		return filenames, err
	}

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}
		defer rc.Close()

		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, f.Name)
		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {

			// Make Folder
			os.MkdirAll(fpath, os.ModePerm)

		} else {

			// Make File
			var fdir string
			if lastIndex := strings.LastIndex(fpath, string(os.PathSeparator)); lastIndex > -1 {
				fdir = fpath[:lastIndex]
			}

			err = os.MkdirAll(fdir, os.ModePerm)
			if err != nil {
				log.Fatal(err)
				return filenames, err
			}
			f, err := os.OpenFile(
				fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return filenames, err
			}
			defer f.Close()

			_, err = io.Copy(f, rc)
			if err != nil {
				return filenames, err
			}

		}
	}
	return filenames, nil
}
