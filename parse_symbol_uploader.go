package main

import (
	"bufio"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/facebookgo/errgroup"
	"github.com/facebookgo/stackerr"
)

func base64MD5OfFile(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", stackerr.Wrap(err)
	}
	defer file.Close()

	h := md5.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", stackerr.Wrap(err)
	}
	if err := file.Close(); err != nil {
		return "", stackerr.Wrap(err)
	}
	return base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}

func uploadSymbolFiles(files []string, commonHeaders map[string]string, removeFiles bool, e *env) error {
	var wg errgroup.Group
	uploadFile := func(filename string, e *env) {
		defer wg.Done()
		name := filepath.Base(filepath.Clean(filename))
		file, err := os.Open(filename)
		if err != nil {
			wg.Error(stackerr.Wrap(err))
			return
		}
		defer file.Close()

		req, err := http.NewRequest("POST", path.Join("symbolFiles", name), bufio.NewReader(file))
		if err != nil {
			wg.Error(stackerr.Wrap(err))
			return
		}

		if req.Header == nil {
			req.Header = make(http.Header)
		}
		for key, val := range commonHeaders {
			req.Header.Add(key, val)
		}
		hash, err := base64MD5OfFile(filename)
		if err != nil {
			wg.Error(err)
			return
		}
		req.Header.Add("Content-MD5", hash)
		mimeType := mime.TypeByExtension(filepath.Ext(name))
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
		req.Header.Add("Content-Type", mimeType)
		res := make(map[string]interface{})
		if _, err := e.ParseAPIClient.Do(req, nil, &res); err != nil {
			wg.Error(err)
			return
		}
		if removeFiles {
			if err := file.Close(); err != nil {
				wg.Error(err)
				return
			}
			if err := os.Remove(filename); err != nil {
				wg.Error(err)
				return
			}
		}
	}
	for _, file := range files {
		wg.Add(1)
		go uploadFile(file, e)
	}
	err := wg.Wait()
	if err != nil {
		return err
	}

	fmt.Fprintln(e.Out, "Uploaded symbol files.")
	return nil
}

func readDirNames(dirname string) ([]string, error) {
	var files []string
	err := filepath.Walk(dirname, func(path string,
		info os.FileInfo,
		err error) error {
		if path == dirname {
			return nil
		}
		files = append(files, path)
		if info.IsDir() {
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return nil, stackerr.Wrap(err)
	}
	return files, nil
}
