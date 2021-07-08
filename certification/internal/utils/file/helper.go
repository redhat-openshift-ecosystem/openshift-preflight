package file

import (
	"compress/bzip2"
	"io"
	"net/http"
	"os"
)

func DownloadFile(filename string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

func Unzip(bzipfile string, destination string) error {

	f, err := os.Open(bzipfile)
	if err != nil {
		return err
	}
	defer f.Close()

	in := bzip2.NewReader(f)

	out, err := os.Create(destination)

	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)

	if err != nil {
		return err
	}
	out.Close()
	return nil
}
