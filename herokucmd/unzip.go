package herokucmd

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"

	"github.com/facebookgo/stackerr"
)

func unzip(zfile string, base string, target string) error {
	r, err := zip.OpenReader(zfile)
	if err != nil {
		return stackerr.Wrap(err)
	}
	defer r.Close()

	for _, f := range r.Reader.File {
		z, err := f.Open()
		if err != nil {
			return stackerr.Wrap(err)
		}
		defer z.Close()

		name, err := filepath.Rel(base, f.Name)
		if err != nil {
			return stackerr.Wrap(err)
		}
		path := filepath.Join(target, name)
		if f.FileInfo().IsDir() {
			err := os.MkdirAll(path, f.Mode())
			if err != nil {
				return stackerr.Wrap(err)
			}
		} else {
			w, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, f.Mode())
			if err != nil {
				return stackerr.Wrap(err)
			}
			defer w.Close()

			_, err = io.Copy(w, z)
			if err != nil {
				return stackerr.Wrap(err)
			}
		}
	}
	return nil
}
